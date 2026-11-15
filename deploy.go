package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	extract "github.com/hashicorp/go-extract"
)

const (
	// 0o755: owner can read/write/enter, everyone else can read/enter.
	dirPerm os.FileMode = 0o755
	// 0o644: owner can read/write, everyone else can read.
	filePerm os.FileMode = 0o644
)

func prepareSitesDir(dir string) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		log.Fatal(err)
	}
	// <SitesDir>/.upload-*/ holds sites currently being extracted
	// They might not get deleted normally if the process is killed or crashes
	for _, pat := range []string{".upload-*", ".old-*"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pat))
		for _, m := range matches {
			os.RemoveAll(m)
		}
	}
}

// UserError is a problem caused by the uploaded content. 4xx not 500
type UserError struct{ Msg string }

func (e *UserError) Error() string { return e.Msg }

func userErrorf(format string, a ...any) error {
	return &UserError{Msg: fmt.Sprintf(format, a...)}
}

func Deploy(cfg Config, subdomain string, file multipart.File, header *multipart.FileHeader) error {
	tmp, err := os.MkdirTemp(cfg.SitesDir, ".upload-*")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("sites directory %q was removed while running: %w", cfg.SitesDir, err)
		}
		return err
	}
	defer os.RemoveAll(tmp)

	staging := filepath.Join(tmp, "new")
	if err := os.Mkdir(staging, dirPerm); err != nil {
		return err
	}

	switch strings.ToLower(filepath.Ext(header.Filename)) {
	case ".html", ".htm":
		err = saveIndex(cfg, filepath.Join(staging, "index.html"), file)
	default:
		err = unpack(cfg, staging, file)
	}
	if err != nil {
		return err
	}

	if err := checkTree(staging); err != nil {
		return err
	}
	siteDir, err := findIndexDir(staging)
	if err != nil {
		return err
	}
	return swap(filepath.Join(cfg.SitesDir, subdomain), siteDir, filepath.Join(tmp, "old"))
}

func saveIndex(cfg Config, dst string, src io.Reader) error {
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

func unpack(cfg Config, dst string, src io.Reader) error {
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

// swap replaces live with next, parking the old version at prev to be restored
// on failure. Note: Not atomic. Live will be gone for a moment while it swaps
func swap(live, next, prev string) error {
	hadOld := true
	if err := os.Rename(live, prev); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		hadOld = false // First deploy
	}
	if err := os.Rename(next, live); err != nil {
		if hadOld {
			err = errors.Join(err, os.Rename(prev, live))
		}
		return err
	}
	return nil
}
