package db

import (
	"context"
	"database/sql"
	"io/fs"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

func openTest(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

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
	db := openTest(t)
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

// Deleting a user removes their sessions and share links, but leaves what they
// made or invited behind without them
func TestDeletingUser(t *testing.T) {
	db := openTest(t)
	exec := func(q string) {
		t.Helper()
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	count := func(q string) (n int) {
		t.Helper()
		if err := db.QueryRow(q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if _, err := db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (x'ff', 999, 0)`); err == nil {
		t.Error("session for a nonexistent user was accepted")
	}

	exec(`INSERT INTO users (id, name, role, invited_by, created_at) VALUES (1, 'a', 'member', NULL, 0), (2, 'g', 'guest', 1, 0)`)
	exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (x'01', 1, 0)`)
	exec(`INSERT INTO sites (name, owner_id, created_at) VALUES ('s', 1, 0)`)
	// A guest invite and a share link
	exec(`INSERT INTO invites (token_hash, user_id, created_by, expires_at) VALUES (x'01', NULL, 1, 0), (x'02', 1, 1, 0)`)
	exec(`DELETE FROM users WHERE id = 1`)

	for q, want := range map[string]int{
		`SELECT count(*) FROM sessions`:                                                0,
		`SELECT count(*) FROM sites WHERE owner_id IS NULL`:                            1,
		`SELECT count(*) FROM invites WHERE token_hash = x'01' AND created_by IS NULL`: 1,
		`SELECT count(*) FROM invites WHERE token_hash = x'02'`:                        0,
		`SELECT count(*) FROM users WHERE id = 2 AND invited_by IS NULL`:               1,
	} {
		if got := count(q); got != want {
			t.Errorf("%s: got %d, want %d", q, got, want)
		}
	}
}
