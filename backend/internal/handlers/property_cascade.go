package handlers

import (
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// cascadeDeleteProperty hard-deletes a property and everything hanging off it:
// utilities (with bills, installments, readings, rates, price changes,
// communications), expenses and splits, projects, settlements and their
// allocations, bill templates, household settings, members, join requests and
// notifications. Run it inside a transaction: it stops at the first failing
// statement so the caller can roll back. Bill attachments left on disk are
// reclaimed by SweepUploads.
//
// Soft-deleted rows are part of the tree: they still hold their foreign keys,
// so every lookup and delete is Unscoped. Rows outside the property that point
// into it (a split's last settlement, another home's expense filed under one of
// these projects) are unlinked, not deleted. TestCascadeDeleteProperty_CoversSchema
// fails when a new foreign key into this tree is not handled here.
//
// search_index rows are removed explicitly: batch deletes hand AfterDelete a
// zero-value struct, so the hooks would only remove entity_id=0.
func cascadeDeleteProperty(tx *gorm.DB, propertyID uint) error {
	// A Session makes the handle safe to reuse: each chain starts clean.
	all := tx.Unscoped().Session(&gorm.Session{})
	var utilityIDs, billIDs, expenseIDs, splitIDs, projectIDs, settlementIDs []uint
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
			return all.Model(&models.ExpenseSplit{}).Where("expense_id IN (?)",
				all.Model(&models.Expense{}).Select("id").Where("property_id = ?", propertyID)).Pluck("id", &splitIDs).Error
		},
		func() error {
			return all.Model(&models.Project{}).Where("property_id = ?", propertyID).Pluck("id", &projectIDs).Error
		},
		func() error {
			return all.Model(&models.Settlement{}).Where("property_id = ?", propertyID).Pluck("id", &settlementIDs).Error
		},
	}
	for _, lookup := range lookups {
		if err := lookup(); err != nil {
			return err
		}
	}

	steps := []func() error{}
	del := func(model any, query string, ids any) {
		steps = append(steps, func() error { return all.Where(query, ids).Delete(model).Error })
	}
	unlink := func(model any, column string, ids []uint) {
		steps = append(steps, func() error {
			return all.Model(model).Where(column+" IN ?", ids).Update(column, nil).Error
		})
	}
	unindex := func(entityType string, ids []uint) {
		steps = append(steps, func() error {
			return tx.Exec("DELETE FROM search_index WHERE entity_type = ? AND entity_id IN ?", entityType, ids).Error
		})
	}

	// Settlements go first: allocations and splits point at them, and they
	// point at expenses (target_expense_id).
	if len(settlementIDs) > 0 {
		del(&models.SettlementAllocation{}, "settlement_id IN ?", settlementIDs)
		unlink(&models.ExpenseSplit{}, "settlement_id", settlementIDs)
		del(&models.Settlement{}, "id IN ?", settlementIDs)
	}

	if len(billIDs) > 0 {
		del(&models.BillInstallment{}, "bill_id IN ?", billIDs)
		del(&models.Bill{}, "id IN ?", billIDs)
		unindex("bill", billIDs)
	}
	if len(utilityIDs) > 0 {
		del(&models.PriceChange{}, "utility_id IN ?", utilityIDs)
		del(&models.ServiceCommunication{}, "utility_id IN ?", utilityIDs)
		del(&models.MeterReading{}, "utility_id IN ?", utilityIDs)
		del(&models.UtilityRate{}, "utility_id IN ?", utilityIDs)
		del(&models.Utility{}, "id IN ?", utilityIDs)
		unindex("utility", utilityIDs)
	}

	if len(splitIDs) > 0 {
		del(&models.SettlementAllocation{}, "expense_split_id IN ?", splitIDs)
	}
	if len(expenseIDs) > 0 {
		unlink(&models.Settlement{}, "target_expense_id", expenseIDs)
		del(&models.ExpenseSplit{}, "expense_id IN ?", expenseIDs)
		del(&models.Expense{}, "id IN ?", expenseIDs)
		unindex("expense", expenseIDs)
	}

	if len(projectIDs) > 0 {
		// Expenses of these projects filed under another property survive, unlinked.
		unlink(&models.Expense{}, "project_id", projectIDs)
		unlink(&models.ExpenseTemplate{}, "project_id", projectIDs)
		del(&models.ProjectMember{}, "project_id IN ?", projectIDs)
		del(&models.Project{}, "id IN ?", projectIDs)
		unindex("project", projectIDs)
	}

	del(&models.BillTemplate{}, "property_id = ?", propertyID)
	del(&models.HouseholdSettings{}, "property_id = ?", propertyID)
	del(&models.PropertyJoinRequest{}, "property_id = ?", propertyID)
	del(&models.Notification{}, "property_id = ?", propertyID)
	del(&models.HouseholdMember{}, "property_id = ?", propertyID)
	del(&models.Property{}, "id = ?", propertyID)

	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}
