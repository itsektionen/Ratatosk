package sites

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	extract "github.com/hashicorp/go-extract"
	"github.com/itsektionen/ratatosk/internal/config"
)

const (
	// 0o755: owner can read/write/enter, everyone else can read/enter.
	dirPerm os.FileMode = 0o755
	// 0o644: owner can read/write, everyone else can read.
	filePerm os.FileMode = 0o644
)

func PrepareDir(dir string) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		slog.Error("create sites dir", "dir", dir, "err", err)
		os.Exit(1)
	}
	// <SitesDir>/.upload-*/ holds a site upload currently being extracted.
	// If it exists, the program crashed and it can't be assumed complete.
	matches, _ := filepath.Glob(filepath.Join(dir, ".upload-*"))
	for _, m := range matches {
		os.RemoveAll(m)
	}
	// <SitesDir>/.old-*/ is the site backup made during a swap.
	// upload -> live move is atomic so live must be good if it exists.
	olds, _ := filepath.Glob(filepath.Join(dir, ".old-*"))
	for _, old := range olds {
		name := strings.TrimPrefix(filepath.Base(old), ".old-")
		live := filepath.Join(dir, name)
		_, err := os.Stat(live)
		switch {
		case errors.Is(err, fs.ErrNotExist) && ValidName(name):
			if err := os.Rename(old, live); err != nil {
				slog.Error("restore site", "name", name, "err", err)
			}
		case err == nil:
			os.RemoveAll(old)
		case !errors.Is(err, fs.ErrNotExist):
			slog.Error("stat site", "path", live, "err", err)
		}
	}
}

var dnsRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func ValidName(name string) bool {
	return dnsRe.MatchString(name)
}

// UserError is a problem caused by the uploaded content. 4xx not 500
type UserError struct{ Msg string }

func (e *UserError) Error() string {
	return e.Msg
}

func userErrorf(format string, a ...any) error {
	return &UserError{Msg: fmt.Sprintf(format, a...)}
}

var swapLocks sync.Map // site name -> *sync.Mutex

func Deploy(cfg config.Config, name string, file multipart.File, header *multipart.FileHeader) error {
	// <SitesDir>/.upload-XXXX/ per upload
	workDir, err := os.MkdirTemp(cfg.SitesDir, ".upload-*")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("sites directory %q was removed while running: %w", cfg.SitesDir, err)
		}
		return err
	}
	defer os.RemoveAll(workDir)

	// <SitesDir>/.upload-XXXX/new/ to use our perms instead of MkdirTemp defaults
	extractDir := filepath.Join(workDir, "new")
	if err := os.Mkdir(extractDir, dirPerm); err != nil {
		return err
	}

	switch strings.ToLower(filepath.Ext(header.Filename)) {
	case ".html", ".htm":
		err = saveIndex(cfg, filepath.Join(extractDir, "index.html"), file)
	default:
		err = unpack(cfg, extractDir, file)
	}
	if err != nil {
		return err
	}

	if err := checkTree(extractDir); err != nil {
		return err
	}
	// Extracted folder holding index.html (site root). All parents will be discarded
	newDir, err := findIndexDir(extractDir)
	if err != nil {
		return err
	}

	// Swaps, swaps, one at a time!
	mu, _ := swapLocks.LoadOrStore(name, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	liveDir := filepath.Join(cfg.SitesDir, name)
	// Parked outside workDir, so it survives the cleanup of workDir if we crash mid-swap
	backupDir := filepath.Join(cfg.SitesDir, ".old-"+name)
	if err := swap(liveDir, newDir, backupDir); err != nil {
		return err
	}
	// New version is live
	os.RemoveAll(backupDir)
	return nil
}

func saveIndex(cfg config.Config, dst string, src io.Reader) error {
	file, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePerm)
	if err != nil {
		return err
	}
	// Read one past the limit so we can tell if we are exactly at it or over
	n, err := io.Copy(file, io.LimitReader(src, cfg.MaxUnzipped+1))
	if cerr := file.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n > cfg.MaxUnzipped {
		return userErrorf("file too large")
	}
	return nil
}

func unpack(cfg config.Config, dst string, src io.Reader) error {
	ecfg := extract.NewConfig(
		extract.WithMaxExtractionSize(cfg.MaxUnzipped), // zip-bomb guard
		extract.WithMaxFiles(cfg.MaxFiles),
		extract.WithDenySymlinkExtraction(true),
		extract.WithInsecureTraverseSymlinks(false),
		extract.WithDropFileAttributes(true), // ignore setuid bits, owners, etc.
		extract.WithCustomCreateDirMode(dirPerm),
		extract.WithCustomDecompressFileMode(filePerm),
	)
	err := extract.Unpack(context.Background(), dst, src, ecfg)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, extract.ErrMaxExtractionSizeExceeded):
		return userErrorf("archive too large when extracted")
	case errors.Is(err, extract.ErrMaxFilesExceeded):
		return userErrorf("archive contains too many files")
	case errors.Is(err, extract.ErrNoExtractorFound),
		errors.Is(err, extract.ErrUnsupportedFileType),
		errors.Is(err, extract.ErrFailedToReadHeader):
		return userErrorf("unsupported file type: upload an .html file or an archive")
	default:
		return userErrorf("could not extract archive: %v", err)
	}
}

func checkTree(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && !d.Type().IsRegular() {
			return userErrorf("upload contains a non-regular file (symlink or special file)")
		}
		return nil
	})
}

// Archives often wrap everything in a single folder. Unwrap it
func findIndexDir(dir string) (string, error) {
	if hasIndex(dir) {
		return dir, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		if inner := filepath.Join(dir, entries[0].Name()); hasIndex(inner) {
			return inner, nil
		}
	}
	return "", userErrorf("upload has no index.html at its root")
}

// Does not consider symlinks. Safe only after checkTree
func hasIndex(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !fi.IsDir()
}

// Replaces the directories at current with mext, moving the old one to prev.
// current -> prev (if exists)
// next -> current
// Note: not atomic. There is a moment where current is gone. 
// On failiure, prev is moved back.
func swap(current, next, prev string) error {
	hadCurrent := true
	if err := os.Rename(current, prev); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		hadCurrent = false
	}
	if err := os.Rename(next, current); err != nil {
		if hadCurrent {
			err = errors.Join(err, os.Rename(prev, current))
		}
		return err
	}
	return nil
}
