package models

import (
	"strings"

	"github.com/sgiraz/homelog/internal/search"
	"gorm.io/gorm"
)

// The hooks below maintain the FTS5 search_index table. Each hook resolves
// whatever extra context it needs (category name, utility provider, etc.)
// from the same transaction the save is running in, so the index update is
// atomic with the underlying row change.
//
// Two invariants to preserve:
//   - AfterSave must be idempotent (upsert, not insert) because it fires on
//     both create and update.
//   - AfterDelete must run on soft delete too (GORM fires it on both paths).

// truncate clamps a title to n runes. Byte-slicing (s[:n]) would split
// multi-byte UTF-8 runes; ranging over the string yields byte offsets of
// rune boundaries, so s[:i] is always valid UTF-8.
func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

// ── Expense ──────────────────────────────────────────────────────────────

func (e *Expense) AfterSave(tx *gorm.DB) error {
	var categoryName, subcategoryName string
	tx.Raw(`SELECT COALESCE(name, '') FROM categories WHERE id = ?`, e.CategoryID).Scan(&categoryName)
	if e.SubcategoryID != nil {
		tx.Raw(`SELECT COALESCE(name, '') FROM subcategories WHERE id = ?`, *e.SubcategoryID).Scan(&subcategoryName)
	}
	propertyID := uint(0)
	if e.PropertyID != nil {
		propertyID = *e.PropertyID
	}
	title := truncate(e.Description, 120)
	body := strings.TrimSpace(e.Description + " " + categoryName + " " + subcategoryName)
	return search.Upsert(tx, search.TypeExpense, e.ID, propertyID, e.UserID, title, body)
}

func (e *Expense) AfterDelete(tx *gorm.DB) error {
	return search.Remove(tx, search.TypeExpense, e.ID)
}

// ── Bill ─────────────────────────────────────────────────────────────────

func (b *Bill) AfterSave(tx *gorm.DB) error {
	var provider string
	var propertyID uint
	tx.Raw(`SELECT COALESCE(provider, ''), property_id FROM utilities WHERE id = ?`, b.UtilityID).Row().Scan(&provider, &propertyID)

	var commContent string
	tx.Raw(`SELECT COALESCE(content, '') FROM service_communications WHERE bill_id = ? AND deleted_at IS NULL LIMIT 1`, b.ID).Scan(&commContent)

	title := truncate(strings.TrimSpace(provider+" "+b.BillNumber), 120)
	body := strings.TrimSpace(b.BillNumber + " " + provider + " " + commContent)
	return search.Upsert(tx, search.TypeBill, b.ID, propertyID, 0, title, body)
}

func (b *Bill) AfterDelete(tx *gorm.DB) error {
	return search.Remove(tx, search.TypeBill, b.ID)
}

// ── Project ──────────────────────────────────────────────────────────────

func (p *Project) AfterSave(tx *gorm.DB) error {
	propertyID := uint(0)
	if p.PropertyID != nil {
		propertyID = *p.PropertyID
	}
	title := truncate(p.Name, 120)
	body := strings.TrimSpace(p.Name + " " + p.Description)
	return search.Upsert(tx, search.TypeProject, p.ID, propertyID, p.UserID, title, body)
}

func (p *Project) AfterDelete(tx *gorm.DB) error {
	return search.Remove(tx, search.TypeProject, p.ID)
}

// ── Utility ──────────────────────────────────────────────────────────────

// utilityTypeSynonyms maps each Utility.Type code (stored in English) to the
// Italian words users actually type when searching. Without this, searching
// "luce" misses `electricity` utilities because the indexed body only carries
// the English code. Extend when adding new utility types.
var utilityTypeSynonyms = map[string]string{
	"electricity": "luce elettricità elettrica",
	"gas":         "gas metano",
	"water":       "acqua idrico",
	"waste":       "rifiuti tari spazzatura",
	"internet":    "internet adsl fibra telefono",
	"insurance":   "assicurazione polizza",
	"affitto":     "affitto",
	"mutuo":       "mutuo",
}

func (u *Utility) AfterSave(tx *gorm.DB) error {
	typeIT := utilityTypeSynonyms[u.Type]
	title := truncate(strings.TrimSpace(u.Type+" — "+u.Provider), 120)
	body := strings.TrimSpace(
		u.Type + " " + typeIT + " " +
			u.Provider + " " + u.Address + " " + u.CustomerCode + " " + u.Notes,
	)
	return search.Upsert(tx, search.TypeUtility, u.ID, u.PropertyID, 0, title, body)
}

func (u *Utility) AfterDelete(tx *gorm.DB) error {
	return search.Remove(tx, search.TypeUtility, u.ID)
}
