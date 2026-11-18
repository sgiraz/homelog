package handlers

import (
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// cascadeDeleteProperty hard-deletes a property and everything hanging off it:
// utilities (with bills, installments, readings, rates, price changes,
// communications), expenses and splits, projects, settlements and their
// allocations, bill templates, household settings, members, join requests and
// notifications, and the members' IDs in everyone's default split. Run it
// inside a transaction: it stops at the first failing statement so the caller
// can roll back. Bill attachments left on disk are reclaimed by SweepUploads.
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
	var utilityIDs, billIDs, expenseIDs, projectIDs, settlementIDs, memberIDs []uint
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
		func() error {
			return all.Model(&models.Settlement{}).Where("property_id = ?", propertyID).Pluck("id", &settlementIDs).Error
		},
		func() error {
			return all.Model(&models.HouseholdMember{}).Where("property_id = ?", propertyID).Pluck("id", &memberIDs).Error
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

	steps = append(steps,
		func() error { return deleteExpenses(tx, expenseIDs) },
		// Expenses of these projects filed under another property survive, unlinked.
		func() error { return deleteProjects(tx, projectIDs) },
	)

	del(&models.BillTemplate{}, "property_id = ?", propertyID)
	del(&models.HouseholdSettings{}, "property_id = ?", propertyID)
	del(&models.PropertyJoinRequest{}, "property_id = ?", propertyID)
	del(&models.Notification{}, "property_id = ?", propertyID)
	steps = append(steps, func() error { return cleanupSplitDefaults(tx, memberIDs...) })
	del(&models.HouseholdMember{}, "property_id = ?", propertyID)
	del(&models.Property{}, "id = ?", propertyID)

	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// deleteExpenses hard-deletes expenses with their splits and the allocations
// that settled them. A settlement aimed at one of them stays, untargeted.
func deleteExpenses(tx *gorm.DB, expenseIDs []uint) error {
	if len(expenseIDs) == 0 {
		return nil
	}
	all := tx.Unscoped().Session(&gorm.Session{})
	splits := all.Model(&models.ExpenseSplit{}).Select("id").Where("expense_id IN ?", expenseIDs)
	steps := []func() error{
		func() error {
			return all.Where("expense_split_id IN (?)", splits).Delete(&models.SettlementAllocation{}).Error
		},
		func() error {
			return all.Model(&models.Settlement{}).Where("target_expense_id IN ?", expenseIDs).Update("target_expense_id", nil).Error
		},
		func() error { return all.Where("expense_id IN ?", expenseIDs).Delete(&models.ExpenseSplit{}).Error },
		func() error { return all.Where("id IN ?", expenseIDs).Delete(&models.Expense{}).Error },
		func() error {
			return tx.Exec("DELETE FROM search_index WHERE entity_type = 'expense' AND entity_id IN ?", expenseIDs).Error
		},
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// deleteProjects hard-deletes projects and their member lists. Expenses and
// expense templates filed under them stay, unfiled.
func deleteProjects(tx *gorm.DB, projectIDs []uint) error {
	if len(projectIDs) == 0 {
		return nil
	}
	all := tx.Unscoped().Session(&gorm.Session{})
	steps := []func() error{
		func() error {
			return all.Model(&models.Expense{}).Where("project_id IN ?", projectIDs).Update("project_id", nil).Error
		},
		func() error {
			return all.Model(&models.ExpenseTemplate{}).Where("project_id IN ?", projectIDs).Update("project_id", nil).Error
		},
		func() error { return all.Where("project_id IN ?", projectIDs).Delete(&models.ProjectMember{}).Error },
		func() error { return all.Where("id IN ?", projectIDs).Delete(&models.Project{}).Error },
		func() error {
			return tx.Exec("DELETE FROM search_index WHERE entity_type = 'project' AND entity_id IN ?", projectIDs).Error
		},
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}
