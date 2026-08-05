package kernel

import (
	"bufio"
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
// MeteringWAL — segmented Write-Ahead Log for transient failure recovery.
// ---------------------------------------------------------------------------

// WALEntry represents a single failed write operation captured for retry.
type WALEntry struct {
	Timestamp   time.Time       `json:"ts"`
	Operation   string          `json:"op"`      // "record_event" | "register_sub"
	BackendName string          `json:"backend"` // which backend failed
	Payload     json.RawMessage `json:"payload"`
	RetryCount  int             `json:"retries"`
}

// WALConfig holds the tuning parameters for the segmented WAL.
type WALConfig struct {
	Dir            string        // directory for WAL segments (default: "{DataDir}/wal")
	MaxSegmentSize int64         // bytes per segment before rotation (default: 10MB)
	MaxSegmentAge  time.Duration // max age before rotation (default: 10 minutes)
	RetryInterval  time.Duration // how often to replay sealed segments (default: 5s)
	Enabled        bool          // master switch (default: true)
}

// DefaultWALConfig returns production-sensible defaults.
func DefaultWALConfig() WALConfig {
	return WALConfig{
		Dir:            "./data/wal",
		MaxSegmentSize: 10 * 1024 * 1024, // 10 MB
		MaxSegmentAge:  10 * time.Minute,
		RetryInterval:  5 * time.Second,
		Enabled:        true,
	}
}

// MeteringWAL captures failed write operations and replays them when backends
// recover. Uses segmented files to prevent single-file I/O bottleneck under
// high write load.
//
// Segment lifecycle:
//  1. Active segment receives appends via Append()
//  2. When segment exceeds MaxSegmentSize or MaxSegmentAge → rotate to sealed
//  3. Retry loop processes sealed segments oldest-first
//  4. Successfully replayed segments are deleted
type MeteringWAL struct {
	cfg             WALConfig
	mu              sync.Mutex
	activeFile      *os.File
	activeSize      int64
	activeCreatedAt time.Time
	segmentCounter  int
	stopCh          chan struct{}
	stopped         bool
}

// NewMeteringWAL creates a new segmented WAL. Call Start() to begin the
// background retry loop.
func NewMeteringWAL(cfg WALConfig) (*MeteringWAL, error) {
	if !cfg.Enabled {
		return &MeteringWAL{cfg: cfg, stopCh: make(chan struct{})}, nil
	}

	if cfg.Dir == "" {
		cfg.Dir = "./data/wal"
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
		return nil, fmt.Errorf("wal: failed to create directory %s: %w", cfg.Dir, err)
	}

	w := &MeteringWAL{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}

	// Count existing segments to set the counter correctly.
	w.segmentCounter = w.countExistingSegments()

	if err := w.rotateSegment(); err != nil {
		return nil, fmt.Errorf("wal: failed to create initial segment: %w", err)
	}

	return w, nil
}

// Append writes a failed operation to the active WAL segment. If the segment
// has exceeded its size or age limit, it is rotated first.
//
// Thread-safe. Returns error only if the WAL file itself cannot be written
// (disk full, permissions), which is a system-level failure.
func (w *MeteringWAL) Append(entry WALEntry) error {
	if !w.cfg.Enabled {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return fmt.Errorf("wal: already stopped")
	}

	// Rotate if the active segment is too large or too old.
	if w.shouldRotate() {
		if err := w.rotateSegment(); err != nil {
			return fmt.Errorf("wal: rotation failed: %w", err)
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("wal: marshal failed: %w", err)
	}
	data = append(data, '\n')

	n, err := w.activeFile.Write(data)
	if err != nil {
		return fmt.Errorf("wal: write failed: %w", err)
	}
	w.activeSize += int64(n)

	return nil
}

// StartRetryLoop starts the background goroutine that replays sealed WAL
// segments. The replayFn is called for each entry — it should attempt to
// write the entry to the target backend and return nil on success.
func (w *MeteringWAL) StartRetryLoop(replayFn func(entry WALEntry) error) {
	if !w.cfg.Enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(w.cfg.RetryInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.replaySealed(replayFn)
			case <-w.stopCh:
				return
			}
		}
	}()
}

// Depth returns the total number of pending WAL entries across all segments
// (sealed + active). Useful for health reporting.
func (w *MeteringWAL) Depth() int {
	if !w.cfg.Enabled {
		return 0
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	count := 0
	segments := w.listSegments()
	for _, seg := range segments {
		count += w.countLines(seg)
	}
	return count
}

// ActiveSegmentName returns the filename of the current active segment.
func (w *MeteringWAL) ActiveSegmentName() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.activeFile == nil {
		return ""
	}
	return filepath.Base(w.activeFile.Name())
}

// TotalSegments returns the count of all WAL segment files (sealed + active).
func (w *MeteringWAL) TotalSegments() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.listSegments())
}

// Stop flushes and closes the active segment, then terminates the retry
// goroutine. Safe to call multiple times.
func (w *MeteringWAL) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}
	w.stopped = true

	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}

	if w.activeFile != nil {
		_ = w.activeFile.Sync()
		_ = w.activeFile.Close()
		w.activeFile = nil
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (w *MeteringWAL) shouldRotate() bool {
	if w.activeFile == nil {
		return true
	}
	if w.activeSize >= w.cfg.MaxSegmentSize {
		return true
	}
	if time.Since(w.activeCreatedAt) >= w.cfg.MaxSegmentAge {
		return true
	}
	return false
}

func (w *MeteringWAL) rotateSegment() error {
	// Close the current active segment (it becomes "sealed").
	if w.activeFile != nil {
		_ = w.activeFile.Sync()
		_ = w.activeFile.Close()
	}

	w.segmentCounter++
	name := fmt.Sprintf("segment_%04d.jsonl", w.segmentCounter)
	path := filepath.Join(w.cfg.Dir, name)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	w.activeFile = f
	w.activeSize = 0
	w.activeCreatedAt = time.Now()
	return nil
}

func (w *MeteringWAL) listSegments() []string {
	entries, err := os.ReadDir(w.cfg.Dir)
	if err != nil {
		return nil
	}

	var segments []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "segment_") && strings.HasSuffix(e.Name(), ".jsonl") {
			segments = append(segments, filepath.Join(w.cfg.Dir, e.Name()))
		}
	}
	sort.Strings(segments)
	return segments
}

func (w *MeteringWAL) countExistingSegments() int {
	entries, err := os.ReadDir(w.cfg.Dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "segment_") {
			count++
		}
	}
	return count
}

func (w *MeteringWAL) countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if len(strings.TrimSpace(scanner.Text())) > 0 {
			count++
		}
	}
	return count
}

// replaySealed processes all sealed segments (not the active one) oldest-first.
// Successfully replayed segments are deleted.
func (w *MeteringWAL) replaySealed(replayFn func(entry WALEntry) error) {
	w.mu.Lock()
	segments := w.listSegments()
	activePath := ""
	if w.activeFile != nil {
		activePath = w.activeFile.Name()
	}
	w.mu.Unlock()

	for _, seg := range segments {
		// Skip the active segment — it's still being written to.
		if seg == activePath {
			continue
		}

		if w.replaySegment(seg, replayFn) {
			// All entries replayed successfully — delete the segment.
			_ = os.Remove(seg)
		}
	}
}

// replaySegment reads and replays all entries in a single segment file.
// Returns true if all entries were successfully replayed.
func (w *MeteringWAL) replaySegment(path string, replayFn func(entry WALEntry) error) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	allSucceeded := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry WALEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Corrupt entry — skip but don't delete the segment.
			allSucceeded = false
			continue
		}

		entry.RetryCount++
		if err := replayFn(entry); err != nil {
			allSucceeded = false
			// Re-append to a new WAL entry for next cycle.
			// This happens naturally — the segment won't be deleted,
			// so it will be retried on the next cycle.
		}
	}

	return allSucceeded
}
