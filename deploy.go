package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	extract "github.com/hashicorp/go-extract"
)

// UserError is a problem caused by the uploaded content. 4xx not 500
type UserError struct{ Msg string }

func (e *UserError) Error() string { return e.Msg }

func userErrorf(format string, a ...any) error {
	return &UserError{Msg: fmt.Sprintf(format, a...)}
}

func Deploy(cfg Config, subdomain string, file multipart.File, header *multipart.FileHeader) error {
	tmp, err := os.MkdirTemp(cfg.SitesDir, ".upload-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	switch strings.ToLower(filepath.Ext(header.Filename)) {
	case ".html", ".htm":
		err = saveIndex(filepath.Join(tmp, "index.html"), file, cfg.MaxUnzipped)
	default:
		err = unpack(cfg, tmp, file)
	}
	if err != nil {
		return err
	}

	root, err := siteRoot(tmp)
	if err != nil {
		return err
	}
	if err := checkTree(root); err != nil {
		return err
	}
	return swap(cfg.SitesDir, subdomain, root)
}

func saveIndex(dst string, src io.Reader, limit int64) error {
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePerm)
	if err != nil {
		return err
	}
	// Read one past the limit so we can tell if we are exactly at it or over
	n, err := io.Copy(f, io.LimitReader(src, limit+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n > limit {
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

// zipping a folder usually wraps everything in a directory,
// so we go into it if there is nothing else at the top
func siteRoot(dir string) (string, error) {
	for range 2 {
		if isRegular(filepath.Join(dir, "index.html")) {
			return dir, nil
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return "", err
		}
		if len(entries) != 1 || !entries[0].IsDir() {
			break
		}
		dir = filepath.Join(dir, entries[0].Name())
	}
	return "", userErrorf("upload has no index.html at its root")
}

func isRegular(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode().IsRegular()
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

// swap atomically replaces SitesDir/subdomain with root, keeping the old
// version around until the new one is in place so it can be restored
func swap(sitesDir, subdomain, root string) error {
	final := filepath.Join(sitesDir, subdomain)

	old, err := os.MkdirTemp(sitesDir, ".old-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(old)
	backup := filepath.Join(old, "site")

	hadOld := true
	if err := os.Rename(final, backup); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		hadOld = false
	}
	if err := os.Rename(root, final); err != nil {
		if hadOld {
			_ = os.Rename(backup, final) // roll back
		}
		return err
	}
	return nil
}
