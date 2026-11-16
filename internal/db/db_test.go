package db

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestOpenMigratesAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	for range 2 {
		db, err := Open(ctx, dir)
		if err != nil {
			t.Fatal(err)
		}
		var tables int
		err = db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name IN ('users','sessions','invites','sites')`).Scan(&tables)
		db.Close()
		if err != nil {
			t.Fatal(err)
		}
		if tables != 4 {
			t.Fatalf("tables=%d", tables)
		}
	}
}

func TestOpenCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does", "not", "exist")
	db, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
}

func TestOpenEnablesPragmas(t *testing.T) {
	db, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode=%q, want wal", mode)
	}
	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys=%d, want 1", fk)
	}
}

func TestForeignKeys(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	count := func(q string) (n int) {
		t.Helper()
		if err := db.QueryRowContext(ctx, q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (x'01', 999, 0)`); err == nil {
		t.Error("session for a nonexistent user was accepted")
	}

	exec(`INSERT INTO users (id, name, role, created_at) VALUES (1, 'a', 'user', 0)`)
	exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (x'01', 1, 0)`)
	exec(`INSERT INTO sites (name, owner_id, created_at) VALUES ('s', 1, 0)`)
	exec(`DELETE FROM users WHERE id = 1`)

	if n := count(`SELECT count(*) FROM sessions`); n != 0 {
		t.Errorf("sessions left after deleting their user: %d", n)
	}
	// The site must outlive its owner, just without one
	if n := count(`SELECT count(*) FROM sites WHERE name = 's' AND owner_id IS NULL`); n != 1 {
		t.Errorf("site should remain with a NULL owner, matching rows: %d", n)
	}
}

func TestOpenRejectsNewerDatabase(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	db, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO goose_db_version (version_id, is_applied) VALUES (999, 1)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	_, err = Open(ctx, dir)
	if err == nil || !strings.Contains(err.Error(), "database is at version 999") {
		t.Fatalf("expected a newer-database error, got: %v", err)
	}
}

func TestMigrationsRollBackAndForward(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fsys, _ := fs.Sub(migrationFiles, "migrations")
	p, err := goose.NewProvider(goose.DialectSQLite3, db, fsys)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestInviteOutlivesCreator(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exec := func(q string) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users (id, name, role, created_at) VALUES (1, 'admin', 'admin', 0), (2, 'user', 'guest', 0)`)
	// A guest invite made by the admin, and a share link for the guest
	exec(`INSERT INTO invites (token_hash, user_id, created_by, expires_at) VALUES (x'01', NULL, 1, 0), (x'02', 2, 2, 0)`)

	exec(`DELETE FROM users WHERE id = 1`)
	var n int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM invites WHERE token_hash = x'01' AND created_by IS NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Error("an invite should outlive the admin who created it")
	}

	exec(`DELETE FROM users WHERE id = 2`)
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM invites WHERE token_hash = x'02'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("a share link for a deleted user should be removed with them")
	}
}

func TestGuestOutlivesInviter(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, role, created_at) VALUES (1, 'a', 'member', 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, role, invited_by, created_at) VALUES (2, 'g', 'guest', 1, 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE id = 2 AND invited_by IS NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Error("a guest should outlive the member who invited them, with invited_by cleared")
	}
}
