package handlers

import (
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// cascadeDeleteProperty hard-deletes a home and everything hanging off it:
// utilities (with bills, installments, readings, rates, price changes,
// communications), its Home expenses and their splits, projects, bill
// templates and notifications. A household left without homes goes with it
// (cascadeDeleteHousehold). Run it inside a transaction: it stops at the first
// failing statement so the caller can roll back. Bill attachments left on disk
// are reclaimed by SweepUploads.
//
// The schema does part of the work: detail rows go with their parent (ON
// DELETE CASCADE) and links from rows that outlive it are cleared (ON DELETE
// SET NULL), see the models' constraint tags. Everything else the database
// refuses to orphan, so this deletes it here, children first. Soft-deleted
// rows still hold their foreign keys, so every lookup and delete is Unscoped.
// TestCascadeDeleteProperty_CoversSchema fails when a foreign key into this
// tree is left to neither.
//
// search_index rows are removed explicitly: the FTS table has no foreign keys,
// and batch deletes hand AfterDelete a zero-value struct.
func cascadeDeleteProperty(tx *gorm.DB, propertyID uint) error {
	// A Session makes the handle safe to reuse: each chain starts clean.
	all := tx.Unscoped().Session(&gorm.Session{})
	var property models.Property
	if err := all.Select("id", "household_id").First(&property, propertyID).Error; err != nil {
		return err
	}
	var utilityIDs, billIDs, expenseIDs, projectIDs []uint
	lookups := []func() error{
		func() error {
			return all.Model(&models.Utility{}).Where("property_id = ?", propertyID).Pluck("id", &utilityIDs).Error
		},
		func() error {
			return all.Model(&models.Bill{}).Where("utility_id IN (?)",
				all.Model(&models.Utility{}).Select("id").Where("property_id = ?", propertyID)).Pluck("id", &billIDs).Error
		},
		func() error {
			return all.Model(&models.Expense{}).Where("property_id = ?", propertyID).Pluck("id", &expenseIDs).Error
		},
		func() error {
			return all.Model(&models.Project{}).Where("property_id = ?", propertyID).Pluck("id", &projectIDs).Error
		},
	}
	for _, lookup := range lookups {
		if err := lookup(); err != nil {
			return err
		}
	}

	var otherHomes int64
	if err := all.Model(&models.Property{}).
		Where("household_id = ? AND id <> ?", property.HouseholdID, propertyID).
		Count(&otherHomes).Error; err != nil {
		return err
	}

	del := func(model any, query string, args ...any) func() error {
		return func() error { return all.Where(query, args...).Delete(model).Error }
	}
	unindex := func(entityType string, ids []uint) func() error {
		return func() error { return deleteFromSearchIndex(tx, entityType, ids) }
	}
	steps := []func() error{}
	if otherHomes == 0 {
		// The household goes too: its settlements and expenses first, since
		// expenses go before the bills they pay.
		steps = append(steps, func() error { return clearHouseholdLedger(tx, property.HouseholdID) })
	}
	steps = append(steps,
		func() error { return deleteExpenses(tx, expenseIDs) },
		// Bills go before the readings they point at, which go with the utility.
		del(&models.Bill{}, "id IN ?", billIDs),
		unindex("bill", billIDs),
		del(&models.Utility{}, "id IN ?", utilityIDs),
		unindex("utility", utilityIDs),
		func() error { return deleteProjects(tx, projectIDs) },
		del(&models.BillTemplate{}, "property_id = ?", propertyID),
		del(&models.Notification{}, "property_id = ?", propertyID),
		del(&models.Property{}, "id = ?", propertyID),
	)
	if otherHomes == 0 {
		steps = append(steps, func() error { return deleteHousehold(tx, property.HouseholdID) })
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// clearHouseholdLedger hard-deletes a household's settlements (with their
// allocations) and expenses (with their splits).
func clearHouseholdLedger(tx *gorm.DB, householdID uint) error {
	all := tx.Unscoped().Session(&gorm.Session{})
	if err := all.Where("household_id = ?", householdID).Delete(&models.Settlement{}).Error; err != nil {
		return err
	}
	var expenseIDs []uint
	if err := all.Model(&models.Expense{}).Where("household_id = ?", householdID).Pluck("id", &expenseIDs).Error; err != nil {
		return err
	}
	return deleteExpenses(tx, expenseIDs)
}

// deleteHousehold hard-deletes a household that has no homes and no ledger
// left: its settings, join requests and members, and the members' IDs in
// everyone's default split.
func deleteHousehold(tx *gorm.DB, householdID uint) error {
	all := tx.Unscoped().Session(&gorm.Session{})
	var members []uint
	if err := all.Model(&models.HouseholdMember{}).Where("household_id = ?", householdID).Pluck("id", &members).Error; err != nil {
		return err
	}
	if err := cleanupSplitDefaults(tx, members...); err != nil {
		return err
	}
	for _, model := range []any{&models.HouseholdSettings{}, &models.HouseholdJoinRequest{}, &models.HouseholdMember{}} {
		if err := all.Where("household_id = ?", householdID).Delete(model).Error; err != nil {
			return err
		}
	}
	return all.Where("id = ?", householdID).Delete(&models.Household{}).Error
}

// deleteExpenses hard-deletes expenses. Their splits go with them; the
// allocations that settled those splits are money that moved, which the
// schema refuses to drop implicitly, so they are deleted here first. A
// settlement aimed at one of them stays, untargeted.
func deleteExpenses(tx *gorm.DB, expenseIDs []uint) error {
	if len(expenseIDs) == 0 {
		return nil
	}
	all := tx.Unscoped().Session(&gorm.Session{})
	splits := all.Model(&models.ExpenseSplit{}).Select("id").Where("expense_id IN ?", expenseIDs)
	if err := all.Where("expense_split_id IN (?)", splits).Delete(&models.SettlementAllocation{}).Error; err != nil {
		return err
	}
	if err := all.Where("id IN ?", expenseIDs).Delete(&models.Expense{}).Error; err != nil {
		return err
	}
	return deleteFromSearchIndex(tx, "expense", expenseIDs)
}

// deleteProjects hard-deletes projects with their member lists. Expenses and
// expense templates filed under them stay, unfiled.
func deleteProjects(tx *gorm.DB, projectIDs []uint) error {
	if len(projectIDs) == 0 {
		return nil
	}
	if err := tx.Unscoped().Where("id IN ?", projectIDs).Delete(&models.Project{}).Error; err != nil {
		return err
	}
	return deleteFromSearchIndex(tx, "project", projectIDs)
}

func deleteFromSearchIndex(tx *gorm.DB, entityType string, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return tx.Exec("DELETE FROM search_index WHERE entity_type = ? AND entity_id IN ?", entityType, ids).Error
}
