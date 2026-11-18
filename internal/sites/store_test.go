package sites

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/db"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	database, err := db.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	// Sites are owned by users, so tests can use owner IDs 1 to 9
	for id := 1; id <= 9; id++ {
		if _, err := database.Exec(`INSERT INTO users (id, name, role, created_at) VALUES (?, 'u', 'member', 0)`, id); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	return NewStore(database, config.Config{SitesDir: dir}), dir
}

// deployed claims name for userID and gives it an index.html
func deployed(t *testing.T, s *Store, dir, name string, userID int64) {
	t.Helper()
	if _, _, err := s.Claim(context.Background(), name, userID); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, name), dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name, "index.html"), []byte(name), filePerm); err != nil {
		t.Fatal(err)
	}
}

func TestRenameMovesRowAndFiles(t *testing.T) {
	ctx := context.Background()
	s, dir := newTestStore(t)
	deployed(t, s, dir, "old", 7)

	if err := s.Rename(ctx, "old", "new"); err != nil {
		t.Fatal(err)
	}
	if site, err := s.Get(ctx, "new"); err != nil || site.OwnerID != 7 {
		t.Errorf("new: %+v, %v, want owned by 7", site, err)
	}
	if _, err := s.Get(ctx, "old"); !errors.Is(err, ErrNotFound) {
		t.Errorf("old still has a row: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "new", "index.html")); err != nil || string(b) != "old" {
		t.Errorf("files did not move: %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "old")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("old files are still there: %v", err)
	}
}

func TestRenameToTakenName(t *testing.T) {
	ctx := context.Background()
	s, dir := newTestStore(t)
	deployed(t, s, dir, "a", 1)
	deployed(t, s, dir, "b", 2)
	// Files without a row, added by hand
	if err := os.Mkdir(filepath.Join(dir, "manual"), dirPerm); err != nil {
		t.Fatal(err)
	}

	for _, to := range []string{"b", "manual"} {
		if err := s.Rename(ctx, "a", to); !errors.Is(err, ErrTaken) {
			t.Errorf("rename to %s: %v, want ErrTaken", to, err)
		}
	}
	// Nothing moved
	if site, err := s.Get(ctx, "b"); err != nil || site.OwnerID != 2 {
		t.Errorf("b: %+v, %v, want still owned by 2", site, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a", "index.html")); err != nil {
		t.Errorf("a lost its files: %v", err)
	}
}

func TestRenameMissingSite(t *testing.T) {
	s, _ := newTestStore(t)
	if err := s.Rename(context.Background(), "nope", "new"); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// A site that was claimed but never deployed has no files to move
func TestRenameWithoutFiles(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)
	if _, _, err := s.Claim(ctx, "empty", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Rename(ctx, "empty", "other"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "other"); err != nil {
		t.Errorf("other: %v", err)
	}
}
