package kernel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// CatalogWAL — dedicated Write-Ahead Log for catalog registration failures.
//
// Segment lifecycle mirrors MeteringWAL:
//  1. Active segment receives Append() calls (synchronous, low-frequency)
//  2. When segment exceeds MaxSegmentSize or MaxSegmentAge → rotate to sealed
//  3. ReplayLoop processes sealed segments oldest-first via a caller-supplied fn
//  4. Successfully replayed segments are deleted
// ---------------------------------------------------------------------------

// CatalogWALEntry captures a single failed catalog write for later replay.
type CatalogWALEntry struct {
	Timestamp   time.Time       `json:"ts"`
	Operation   string          `json:"op"`      // "register_service" | "register_metric" | "register_plan"
	BackendName string          `json:"backend"` // which backend failed: "redis" | "postgres"
	TenantKey   string          `json:"tenant_key"`
	Payload     json.RawMessage `json:"payload"`  // serialized descriptor (TenantServiceDescriptor etc.)
	RetryCount  int             `json:"retries"`
}

// CatalogWALConfig holds tuning parameters for the CatalogWAL.
type CatalogWALConfig struct {
	Dir            string        // segment directory (default: "./data/wal/catalog")
	MaxSegmentSize int64         // bytes before rotation (default: 10 MB)
	MaxSegmentAge  time.Duration // age before rotation (default: 10 min)
	RetryInterval  time.Duration // replay loop interval (default: 5 s)
	Enabled        bool          // master switch
}

// DefaultCatalogWALConfig returns production-sensible defaults.
func DefaultCatalogWALConfig(dataDir string) CatalogWALConfig {
	return CatalogWALConfig{
		Dir:            filepath.Join(dataDir, "wal", "catalog"),
		MaxSegmentSize: 10 * 1024 * 1024, // 10 MB
		MaxSegmentAge:  10 * time.Minute,
		RetryInterval:  5 * time.Second,
		Enabled:        true,
	}
}

// CatalogWAL is a segmented append-only log for catalog backend failures.
type CatalogWAL struct {
	cfg             CatalogWALConfig
	mu              sync.Mutex
	activeFile      *os.File
	activeSize      int64
	activeCreatedAt time.Time
	segmentCounter  int
	stopCh          chan struct{}
	stopped         bool
}

// NewCatalogWAL creates and initialises a CatalogWAL. When cfg.Enabled is
// false it returns a no-op stub (all operations are safely ignored).
func NewCatalogWAL(cfg CatalogWALConfig) (*CatalogWAL, error) {
	if !cfg.Enabled {
		return &CatalogWAL{cfg: cfg, stopCh: make(chan struct{})}, nil
	}

	if cfg.Dir == "" {
		cfg.Dir = "./data/wal/catalog"
	}
	if cfg.MaxSegmentSize <= 0 {
		cfg.MaxSegmentSize = 10 * 1024 * 1024
	}
	if cfg.MaxSegmentAge <= 0 {
		cfg.MaxSegmentAge = 10 * time.Minute
	}
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = 5 * time.Second
	}

	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("catalog wal: create directory %s: %w", cfg.Dir, err)
	}

	w := &CatalogWAL{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
	w.segmentCounter = w.countExistingSegments()

	if err := w.rotateSegment(); err != nil {
		return nil, fmt.Errorf("catalog wal: create initial segment: %w", err)
	}

	return w, nil
}

// Append writes a failed catalog operation to the active WAL segment.
// It is synchronous and thread-safe. The active segment is rotated if it
// exceeds size or age limits.
func (w *CatalogWAL) Append(entry CatalogWALEntry) error {
	if !w.cfg.Enabled {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return fmt.Errorf("catalog wal: already stopped")
	}

	if w.shouldRotate() {
		if err := w.rotateSegment(); err != nil {
			return fmt.Errorf("catalog wal: rotation failed: %w", err)
		}
	}

	entry.Timestamp = time.Now().UTC()
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("catalog wal: marshal entry: %w", err)
	}
	data = append(data, '\n')

	n, err := w.activeFile.Write(data)
	if err != nil {
		return fmt.Errorf("catalog wal: write: %w", err)
	}
	w.activeSize += int64(n)
	return nil
}

// ReplayLoop starts a background goroutine that periodically replays sealed
// segments by calling replayFn for each entry. Successfully replayed segments
// are deleted. Stops when ctx is cancelled or Stop() is called.
//
// replayFn should return nil on success (entry replayed) or an error if the
// backend is still unavailable (entry will be retried on the next cycle).
func (w *CatalogWAL) ReplayLoop(ctx context.Context, replayFn func(context.Context, CatalogWALEntry) error) {
	if !w.cfg.Enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(w.cfg.RetryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				w.replaySealed(ctx, replayFn)
			}
		}
	}()
}

// Stop signals the ReplayLoop goroutine to exit and closes the active segment.
// Safe to call multiple times.
func (w *CatalogWAL) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}
	w.stopped = true
	close(w.stopCh)

	// Seal the active segment so it can be replayed on next startup.
	if w.activeFile != nil {
		_ = w.sealActiveSegment()
		w.activeFile = nil
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (w *CatalogWAL) shouldRotate() bool {
	if w.activeFile == nil {
		return true
	}
	sizeExceeded := w.cfg.MaxSegmentSize > 0 && w.activeSize >= w.cfg.MaxSegmentSize
	ageExceeded := w.cfg.MaxSegmentAge > 0 && time.Since(w.activeCreatedAt) >= w.cfg.MaxSegmentAge
	return sizeExceeded || ageExceeded
}

func (w *CatalogWAL) rotateSegment() error {
	if w.activeFile != nil {
		if err := w.sealActiveSegment(); err != nil {
			return err
		}
	}

	w.segmentCounter++
	name := fmt.Sprintf("catalog_wal_%d_%d.active", time.Now().UnixNano(), w.segmentCounter)
	path := filepath.Join(w.cfg.Dir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open new segment %s: %w", path, err)
	}
	w.activeFile = f
	w.activeSize = 0
	w.activeCreatedAt = time.Now()
	return nil
}

// sealActiveSegment renames the active segment from .active → .sealed.
func (w *CatalogWAL) sealActiveSegment() error {
	if w.activeFile == nil {
		return nil
	}
	oldPath := w.activeFile.Name()
	if err := w.activeFile.Close(); err != nil {
		return fmt.Errorf("close active segment: %w", err)
	}
	w.activeFile = nil
	newPath := strings.Replace(oldPath, ".active", ".sealed", 1)
	return os.Rename(oldPath, newPath)
}

func (w *CatalogWAL) countExistingSegments() int {
	entries, err := os.ReadDir(w.cfg.Dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sealed") || strings.HasSuffix(e.Name(), ".active") {
			count++
		}
	}
	return count
}

func (w *CatalogWAL) replaySealed(ctx context.Context, replayFn func(context.Context, CatalogWALEntry) error) {
	w.mu.Lock()
	// Seal current active segment so all entries are in sealed files.
	if err := w.sealActiveSegment(); err == nil {
		_ = w.rotateSegment() // open a fresh active segment
	}
	w.mu.Unlock()

	segments := w.listSealedSegments()
	sort.Strings(segments) // oldest first (timestamp prefix in filename)

	for _, seg := range segments {
		if w.replaySegment(ctx, seg, replayFn) {
			// All entries replayed — delete the segment.
			_ = os.Remove(seg)
		}
	}
}

// replaySegment replays all entries in a sealed segment file.
// Returns true only if every entry was replayed successfully.
func (w *CatalogWAL) replaySegment(ctx context.Context, path string, replayFn func(context.Context, CatalogWALEntry) error) bool {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("[catalog wal] failed to open segment %s: %v\n", path, err)
		return false
	}
	defer f.Close()

	allOK := true
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1 MB per line max
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry CatalogWALEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			fmt.Printf("[catalog wal] skipping malformed entry in %s: %v\n", path, err)
			continue
		}
		entry.RetryCount++
		if err := replayFn(ctx, entry); err != nil {
			fmt.Printf("[catalog wal] replay failed for op=%s backend=%s tenant=%s: %v\n",
				entry.Operation, entry.BackendName, entry.TenantKey, err)
			allOK = false
		}
	}
	return allOK
}

func (w *CatalogWAL) listSealedSegments() []string {
	entries, err := os.ReadDir(w.cfg.Dir)
	if err != nil {
		return nil
	}
	var result []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sealed") {
			result = append(result, filepath.Join(w.cfg.Dir, e.Name()))
		}
	}
	return result
}
