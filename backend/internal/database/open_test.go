package database

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openFile(t *testing.T) (*gorm.DB, *sql.DB) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "homelog.db"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db, sqlDB
}

// eachConn holds n distinct pooled connections at once, so each one is
// checked rather than the same connection n times, then hands them back.
func eachConn(t *testing.T, sqlDB *sql.DB, n int, check func(i int, c *sql.Conn)) {
	t.Helper()
	held := make([]*sql.Conn, n)
	for i := range held {
		c, err := sqlDB.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		held[i] = c
	}
	for i, c := range held {
		check(i, c)
	}
	for _, c := range held {
		c.Close()
	}
}

// eachOpenConn checks every connection the pool currently has open.
func eachOpenConn(t *testing.T, sqlDB *sql.DB, check func(i int, c *sql.Conn)) {
	t.Helper()
	eachConn(t, sqlDB, sqlDB.Stats().OpenConnections, check)
}

func pragma(t *testing.T, c *sql.Conn, name string) string {
	t.Helper()
	var v string
	if err := c.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestOpen_EveryConnectionGetsThePragmas(t *testing.T) {
	_, sqlDB := openFile(t)
	eachConn(t, sqlDB, 3, func(i int, c *sql.Conn) {
		if got := pragma(t, c, "foreign_keys"); got != "1" {
			t.Errorf("connection %d: foreign_keys = %s, want 1", i, got)
		}
		if got := pragma(t, c, "synchronous"); got != "1" {
			t.Errorf("connection %d: synchronous = %s, want 1 (NORMAL)", i, got)
		}
		if got := pragma(t, c, "journal_mode"); got != "wal" {
			t.Errorf("connection %d: journal_mode = %s, want wal", i, got)
		}
		_, err := c.ExecContext(context.Background(),
			"INSERT INTO expense_splits (expense_id, member_id, amount) VALUES (999, 999, 1)")
		if err == nil {
			t.Errorf("connection %d accepted a split for a missing expense", i)
		}
	})
}

// Each connection to :memory: is a new, empty database: the pool must never
// open a second one.
func TestOpen_MemoryStaysOneDatabase(t *testing.T) {
	db, err := Open(":memory:", &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	if got := sqlDB.Stats().MaxOpenConnections; got != 1 {
		t.Errorf("MaxOpenConnections = %d, want 1", got)
	}
}

func TestOpen_RefusesPathsThatWouldBreakTheDSN(t *testing.T) {
	for _, p := range []string{"data/home?.db", "data/home#1.db"} {
		if _, err := Open(p, &gorm.Config{}); err == nil {
			t.Errorf("Open(%q) succeeded, want an error", p)
		}
	}
}

func TestWithoutForeignKeys_OnlyOnThePinnedConnection(t *testing.T) {
	db, sqlDB := openFile(t)
	if err := db.Exec("INSERT INTO users (id, email, password_hash, name, role, is_active) VALUES (1, 'a@example.com', 'x', 'A', 'admin', true)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO categories (id, user_id, name) VALUES (1, 1, 'Mine')").Error; err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(prev) })

	err := WithoutForeignKeys(db, func(conn *gorm.DB) error {
		var fk int
		conn.Raw("PRAGMA foreign_keys").Scan(&fk)
		if fk != 0 {
			t.Errorf("pinned connection: foreign_keys = %d, want 0", fk)
		}
		eachConn(t, sqlDB, 2, func(i int, c *sql.Conn) {
			if got := pragma(t, c, "foreign_keys"); got != "1" {
				t.Errorf("other connection %d: foreign_keys = %s, want 1", i, got)
			}
		})
		// Statements on the pinned connection run unchecked.
		return conn.Exec("DELETE FROM users WHERE id = 1").Error
	})
	if err != nil {
		t.Fatalf("WithoutForeignKeys: %v", err)
	}
	if !strings.Contains(logs.String(), "violate foreign keys") || !strings.Contains(logs.String(), "categories -> users") {
		t.Errorf("the orphaned category was not reported; log: %q", logs.String())
	}

	// The pinned connection is back in the pool: every open one must enforce.
	eachOpenConn(t, sqlDB, func(i int, c *sql.Conn) {
		if got := pragma(t, c, "foreign_keys"); got != "1" {
			t.Errorf("after WithoutForeignKeys, connection %d: foreign_keys = %s, want 1", i, got)
		}
	})
}

func TestWithoutForeignKeys_ReenablesAfterAnError(t *testing.T) {
	db, sqlDB := openFile(t)
	boom := errors.New("boom")
	if err := WithoutForeignKeys(db, func(*gorm.DB) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	eachOpenConn(t, sqlDB, func(i int, c *sql.Conn) {
		if got := pragma(t, c, "foreign_keys"); got != "1" {
			t.Errorf("after a failed run, connection %d: foreign_keys = %s, want 1", i, got)
		}
	})
}
