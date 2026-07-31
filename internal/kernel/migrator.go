package kernel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Migration is one ordered, one-way-forward change. In a real system Up/Down
// would touch a database; in this lab they seed the in-memory Store, but the
// ordering + "never re-apply" guarantee works identically either way.
type Migration struct {
	Version int
	Name    string
	Up      func(ctx context.Context) error
	Down    func(ctx context.Context) error
}

// Migrator persists the applied version number to a small file on disk so
// that re-running the binary doesn't reseed anything already applied.
type Migrator struct {
	migrations  []Migration
	versionFile string
}

func NewMigrator(versionFile string) *Migrator {
	return &Migrator{versionFile: versionFile}
}

func (m *Migrator) Register(mig Migration) {
	m.migrations = append(m.migrations, mig)
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})
}

// currentVersion reads the last applied migration version from disk.
//
// Parameters:
//   none
//
// Returns:
//   int: the last persisted migration version, or 0 if the version file is missing or invalid.
//
// Behavior:
//   Reads the version from the configured version file and returns the stored value.
//
// Constraints:
//   The version must be stored as a plain integer in the file; invalid content results in a fallback value of 0.
func (m *Migrator) currentVersion() int {
	data, err := os.ReadFile(m.versionFile)
	if err != nil {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return v
}

// setVersion persists the supplied migration version to disk.
//
// Parameters:
//   v int: the migration version to store.
//
// Returns:
//   error: an error if the version file cannot be created or written.
//
// Behavior:
//   Ensures the destination directory exists and writes the supplied version as a plain integer string.
//
// Constraints:
//   The target file must be writable, and the parent directory must be creatable.
func (m *Migrator) setVersion(v int) error {
	if err := os.MkdirAll(filepath.Dir(m.versionFile), 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.versionFile, []byte(strconv.Itoa(v)), 0o644)
}

// Run applies every migration newer than the last-applied version, in order.
func (m *Migrator) Run(ctx context.Context) error {
	current := m.currentVersion()
	applied := 0

	for _, mig := range m.migrations {
		if mig.Version <= current {
			continue
		}
		if err := mig.Up(ctx); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", mig.Version, mig.Name, err)
		}
		if err := m.setVersion(mig.Version); err != nil {
			return err
		}
		fmt.Printf("[migrate] applied v%d: %s\n", mig.Version, mig.Name)
		applied++
	}

	if applied == 0 {
		fmt.Printf("[migrate] up to date at v%d\n", current)
	}
	return nil
}
