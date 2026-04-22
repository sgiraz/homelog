// Package search centralises writes to the FTS5 search_index virtual table.
// Model hooks call into it so the index stays in sync with source rows.
package search

import (
	"gorm.io/gorm"
)

// EntityType tags each row in search_index so the result handler can dispatch
// to the right frontend destination (e.g. /expenses vs /utilities/:id).
type EntityType string

const (
	TypeExpense EntityType = "expense"
	TypeBill    EntityType = "bill"
	TypeProject EntityType = "project"
	TypeUtility EntityType = "utility"
)

// Upsert replaces (or inserts) the row for this entity. FTS5 virtual tables
// don't support ON CONFLICT, so we delete-then-insert inside the caller's
// transaction — safe under GORM's AfterSave hook because tx is the same txn
// as the triggering Save.
func Upsert(tx *gorm.DB, etype EntityType, id, propertyID uint, title, body string) error {
	if err := tx.Exec(`DELETE FROM search_index WHERE entity_type = ? AND entity_id = ?`, etype, id).Error; err != nil {
		return err
	}
	return tx.Exec(
		`INSERT INTO search_index (entity_type, entity_id, property_id, title, body) VALUES (?, ?, ?, ?, ?)`,
		etype, id, propertyID, title, body,
	).Error
}

// Remove deletes the row. Called from AfterDelete; GORM fires this for both
// hard and soft deletes, so a soft-deleted expense disappears from search too.
func Remove(tx *gorm.DB, etype EntityType, id uint) error {
	return tx.Exec(`DELETE FROM search_index WHERE entity_type = ? AND entity_id = ?`, etype, id).Error
}
