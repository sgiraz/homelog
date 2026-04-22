package database

import (
	"log"

	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// searchIndexVersion must be incremented whenever the hooks change the shape
// of indexed data (new synonyms, new fields, changed tokenizer). On boot, if
// the stored version is lower than this constant, the index is cleared and
// repopulated from scratch so existing installs see the fix.
//
// History:
//
//	1 — initial
//	2 — add Italian synonyms for Utility.Type; fix Project backfill
const searchIndexVersion = 2

// initSearchIndex provisions the unified FTS5 table used by global search.
// One virtual table, four indexable entity types (expense, bill, project,
// utility). property_id lets the handler filter by household membership
// before returning rows — the column is UNINDEXED because FTS5 treats
// indexed columns as tokenized text, which is pointless for an int key.
//
// It also maintains a single-row `search_index_meta` table that tracks the
// current index version, so hook changes can trigger a rebuild.
func initSearchIndex(db *gorm.DB) error {
	if err := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
			entity_type UNINDEXED,
			entity_id UNINDEXED,
			property_id UNINDEXED,
			title,
			body,
			tokenize = 'unicode61 remove_diacritics 2'
		)
	`).Error; err != nil {
		return err
	}

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS search_index_meta (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			version INTEGER NOT NULL DEFAULT 0
		)
	`).Error; err != nil {
		return err
	}
	db.Exec(`INSERT OR IGNORE INTO search_index_meta (id, version) VALUES (1, 0)`)

	var current int
	db.Raw(`SELECT version FROM search_index_meta WHERE id = 1`).Scan(&current)
	if current < searchIndexVersion {
		log.Printf("🔄 Search index version %d → %d: rebuilding...", current, searchIndexVersion)
		if err := db.Exec(`DELETE FROM search_index`).Error; err != nil {
			return err
		}
		if err := db.Exec(`UPDATE search_index_meta SET version = ? WHERE id = 1`, searchIndexVersion).Error; err != nil {
			return err
		}
	}
	return nil
}

// backfillSearchIndex populates the FTS table from existing rows by invoking
// each model's AfterSave hook directly. Using the hooks keeps a single source
// of truth: whatever logic the hook applies at write time (joins, field
// shaping, language-aware labels) is exactly what gets written during the
// backfill.
//
// Per-entity guard: runs the backfill for any entity type missing from the
// index. This auto-heals installs where an earlier backfill partially failed
// (e.g. a SQL column reference against a field that did not exist).
func backfillSearchIndex(db *gorm.DB) {
	hasRows := func(t string) bool {
		var c int64
		db.Raw("SELECT count(*) FROM search_index WHERE entity_type = ?", t).Scan(&c)
		return c > 0
	}

	logged := false
	logOnce := func() {
		if !logged {
			log.Println("🔄 Backfilling search index...")
			logged = true
		}
	}

	if !hasRows("expense") {
		logOnce()
		var rows []models.Expense
		db.Find(&rows)
		for i := range rows {
			_ = rows[i].AfterSave(db)
		}
	}
	if !hasRows("bill") {
		logOnce()
		var rows []models.Bill
		db.Find(&rows)
		for i := range rows {
			_ = rows[i].AfterSave(db)
		}
	}
	if !hasRows("project") {
		logOnce()
		var rows []models.Project
		db.Find(&rows)
		for i := range rows {
			_ = rows[i].AfterSave(db)
		}
	}
	if !hasRows("utility") {
		logOnce()
		var rows []models.Utility
		db.Find(&rows)
		for i := range rows {
			_ = rows[i].AfterSave(db)
		}
	}

	if logged {
		log.Println("✅ Search index backfill completed")
	}
}
