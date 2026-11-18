package sites

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/sites/sitesdb"
)

// Store tracks which sites exist and who owns them. A site's row is its claim
// on the name and is written before its files, so the only inconsistent states
// are harmless: a row without files is an empty site that 404s and can be
// uploaded to, and files without a row are adopted as unowned on startup
type Store struct {
	q   *sitesdb.Queries
	dir string
}

type Site struct {
	Name      string
	OwnerID   int64  // 0 if unowned, use Owned
	OwnerName string // empty if unowned
	OwnerRole string // empty if unowned
}

// Owned reports whether the site has an owner
func (s Site) Owned() bool {
	return s.OwnerID != 0
}

// OwnedBy reports whether the site is owned by the user with this ID
func (s Site) OwnedBy(userID int64) bool {
	return s.Owned() && s.OwnerID == userID
}

func NewStore(db *sql.DB, cfg config.Config) *Store {
	return &Store{q: sitesdb.New(db), dir: cfg.SitesDir}
}

// Claim creates the site owned by userID if the name is free. Either way it
// returns the owner, so the caller can decide whether userID may modify it.
// A directory without a row (added by hand while running) is not free: it is
// adopted as unowned instead
func (s *Store) Claim(ctx context.Context, name string, userID int64) (site Site, created bool, err error) {
	n, err := s.q.ClaimSite(ctx, sitesdb.ClaimSiteParams{
		Name:      name,
		OwnerID:   sql.NullInt64{Int64: userID, Valid: true},
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		return Site{}, false, err
	}
	if n == 1 {
		if _, err := os.Lstat(filepath.Join(s.dir, name)); err == nil {
			return Site{Name: name}, false, s.q.UnownSite(ctx, name)
		}
		return Site{Name: name, OwnerID: userID}, true, nil
	}
	site, err = s.Get(ctx, name)
	return site, false, err
}

// Release undoes a Claim whose deploy failed
func (s *Store) Release(ctx context.Context, name string) error {
	return s.q.DeleteSite(ctx, name)
}

var ErrNotFound = errors.New("site not found")

func (s *Store) Get(ctx context.Context, name string) (Site, error) {
	row, err := s.q.GetSite(ctx, name)
	if errors.Is(err, sql.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	return Site(row), err
}

func (s *Store) List(ctx context.Context) ([]Site, error) {
	rows, err := s.q.ListSites(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Site, len(rows))
	for i, row := range rows {
		out[i] = Site(row)
	}
	return out, nil
}

// Delete removes the site's row, then its files
func (s *Store) Delete(ctx context.Context, name string) error {
	if err := s.q.DeleteSite(ctx, name); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.dir, name))
}

// Adopt gives every site directory that has no row an unowned one
func (s *Store) Adopt(ctx context.Context) error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() || !ValidName(e.Name()) {
			continue
		}
		if err := s.q.AdoptSite(ctx, sitesdb.AdoptSiteParams{
			Name:      e.Name(),
			CreatedAt: time.Now().Unix(),
		}); err != nil {
			return err
		}
	}
	return nil
}
