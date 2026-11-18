package database

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Connection pragmas. SQLite scopes them to a connection and database/sql
// keeps a pool, so they travel in the DSN: every connection the pool opens
// applies them, not only the first one. (journal_mode is stored in the file,
// but setting it per connection costs nothing.)
var connectionPragmas = []string{
	"foreign_keys(1)",
	"journal_mode(WAL)",
	"synchronous(NORMAL)",
}

// Open opens the SQLite database at path (or ":memory:") with HomeLog's
// connection settings. Every caller goes through it, tests included, so they
// all run under the same rules as production.
func Open(path string, config *gorm.Config) (*gorm.DB, error) {
	if strings.ContainsAny(path, "?#") {
		return nil, fmt.Errorf("database path %q must not contain '?' or '#'", path)
	}
	params := make([]string, len(connectionPragmas))
	for i, p := range connectionPragmas {
		params[i] = "_pragma=" + p
	}
	db, err := gorm.Open(sqlite.Open(path+"?"+strings.Join(params, "&")), config)
	if err != nil {
		return nil, err
	}
	if path == ":memory:" {
		// Each connection to :memory: is a separate, empty database.
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxOpenConns(1)
	}
	return db, nil
}

// WithoutForeignKeys runs fn with foreign keys off, as SQLite requires for
// rebuilding a table that other tables reference. The pragma is per connection
// and ignored inside a transaction, so fn gets one pinned connection: it must
// use the handle it receives (transactions on it are fine) and nothing else.
// Foreign keys are back on before the connection returns to the pool, and any
// row left violating them is logged.
func WithoutForeignKeys(db *gorm.DB, fn func(conn *gorm.DB) error) error {
	return db.Connection(func(pinned *gorm.DB) (err error) {
		conn := pinned.Session(&gorm.Session{})
		if err := conn.Exec("PRAGMA foreign_keys=OFF").Error; err != nil {
			return err
		}
		defer func() {
			if onErr := conn.Exec("PRAGMA foreign_keys=ON").Error; onErr != nil {
				err = errors.Join(err, fmt.Errorf("re-enable foreign keys: %w", onErr))
			}
		}()
		if err := fn(conn); err != nil {
			return err
		}
		logForeignKeyViolations(conn)
		return nil
	})
}

// logForeignKeyViolations reports rows whose foreign keys point nowhere. It
// warns instead of failing: an instance that ran without enforcement may carry
// orphans from before, and refusing to start would not remove them.
func logForeignKeyViolations(conn *gorm.DB) {
	var rows []struct {
		Table  string `gorm:"column:table"`
		Parent string `gorm:"column:parent"`
	}
	if err := conn.Raw(`SELECT "table", "parent" FROM pragma_foreign_key_check`).Scan(&rows).Error; err != nil {
		log.Printf("⚠️  foreign key check failed: %v", err)
		return
	}
	if len(rows) == 0 {
		return
	}
	counts := map[string]int{}
	for _, r := range rows {
		counts[r.Table+" -> "+r.Parent]++
	}
	edges := make([]string, 0, len(counts))
	for edge, n := range counts {
		edges = append(edges, fmt.Sprintf("%s (%d)", edge, n))
	}
	sort.Strings(edges)
	log.Printf("⚠️  %d rows violate foreign keys: %s", len(rows), strings.Join(edges, ", "))
}
