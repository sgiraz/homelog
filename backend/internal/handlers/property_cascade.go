package handlers

import (
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// cascadeDeleteProperty hard-deletes a property and everything hanging off it:
// utilities (with bills, readings, rates, price changes, communications),
// expenses and splits, projects, settlements, bill templates, household
// settings, members, join requests and notifications. Run it inside a
// transaction: it stops at the first failing statement so the caller can roll
// back. Bill attachments left on disk are reclaimed by SweepUploads.
//
// search_index rows are removed explicitly: batch deletes hand AfterDelete a
// zero-value struct, so the hooks would only remove entity_id=0.
func cascadeDeleteProperty(tx *gorm.DB, propertyID uint) error {
	var utilityIDs, billIDs, expenseIDs, projectIDs []uint
	if err := tx.Model(&models.Utility{}).Where("property_id = ?", propertyID).Pluck("id", &utilityIDs).Error; err != nil {
		return err
	}
	if len(utilityIDs) > 0 {
		if err := tx.Model(&models.Bill{}).Where("utility_id IN ?", utilityIDs).Pluck("id", &billIDs).Error; err != nil {
			return err
		}
	}
	if err := tx.Model(&models.Expense{}).Where("property_id = ?", propertyID).Pluck("id", &expenseIDs).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Project{}).Where("property_id = ?", propertyID).Pluck("id", &projectIDs).Error; err != nil {
		return err
	}

	steps := []func() error{}
	del := func(model any, query string, args ...any) {
		steps = append(steps, func() error { return tx.Unscoped().Where(query, args...).Delete(model).Error })
	}
	exec := func(sql string, args ...any) {
		steps = append(steps, func() error { return tx.Exec(sql, args...).Error })
	}

	if len(utilityIDs) > 0 {
		del(&models.PriceChange{}, "utility_id IN ?", utilityIDs)
		del(&models.ServiceCommunication{}, "utility_id IN ?", utilityIDs)
		del(&models.Bill{}, "utility_id IN ?", utilityIDs)
		del(&models.MeterReading{}, "utility_id IN ?", utilityIDs)
		del(&models.UtilityRate{}, "utility_id IN ?", utilityIDs)
		del(&models.Utility{}, "id IN ?", utilityIDs)
		exec("DELETE FROM search_index WHERE entity_type = 'utility' AND entity_id IN ?", utilityIDs)
	}
	if len(billIDs) > 0 {
		exec("DELETE FROM search_index WHERE entity_type = 'bill' AND entity_id IN ?", billIDs)
	}
	if len(expenseIDs) > 0 {
		del(&models.ExpenseSplit{}, "expense_id IN ?", expenseIDs)
		del(&models.Expense{}, "id IN ?", expenseIDs)
		exec("DELETE FROM search_index WHERE entity_type = 'expense' AND entity_id IN ?", expenseIDs)
	}
	if len(projectIDs) > 0 {
		// Expenses of these projects filed under another property survive, unlinked.
		steps = append(steps,
			func() error {
				return tx.Model(&models.Expense{}).Where("project_id IN ?", projectIDs).Update("project_id", nil).Error
			},
			func() error {
				return tx.Model(&models.ExpenseTemplate{}).Where("project_id IN ?", projectIDs).Update("project_id", nil).Error
			},
		)
		del(&models.ProjectMember{}, "project_id IN ?", projectIDs)
		del(&models.Project{}, "id IN ?", projectIDs)
		exec("DELETE FROM search_index WHERE entity_type = 'project' AND entity_id IN ?", projectIDs)
	}
	del(&models.BillTemplate{}, "property_id = ?", propertyID)
	del(&models.Settlement{}, "property_id = ?", propertyID)
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
