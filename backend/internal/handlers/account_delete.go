package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// Deleting an account removes what belongs to that person and leaves the
// homes they share as the others know them. A home nobody else lives in goes
// with the account. In a shared home the account's member turns virtual and
// keeps its expenses, shares and settlements, so nobody else's balance or
// history moves. That is only possible once the home has another admin and
// the member owes and is owed nothing there: accountDeleteCheck reports both.

// accountBlockedError is deleteAccount refusing: the check did not pass.
type accountBlockedError struct{ check DeleteAccountCheckResponse }

func (e *accountBlockedError) Error() string { return "account deletion is blocked" }

// accountHome is a home the account lives in or owns.
type accountHome struct {
	property models.Property
	// member is the account's member row there, nil when it only owns the home.
	member *models.HouseholdMember
	// others are the other registered, current members.
	others []models.HouseholdMember
}

func (h accountHome) shared() bool { return len(h.others) > 0 }

// successor is the admin who takes over the home, or nil if there is none.
func (h accountHome) successor() *models.HouseholdMember {
	for i := range h.others {
		if h.others[i].Role == "admin" {
			return &h.others[i]
		}
	}
	return nil
}

func accountHomes(db *gorm.DB, userID uint) ([]accountHome, error) {
	var members []models.HouseholdMember
	if err := db.Where("user_id = ?", userID).Order("id").Find(&members).Error; err != nil {
		return nil, err
	}
	memberOf := make(map[uint]*models.HouseholdMember, len(members))
	for i := range members {
		memberOf[members[i].PropertyID] = &members[i]
	}

	var properties []models.Property
	if err := db.Where("user_id = ? OR id IN (?)", userID,
		db.Model(&models.HouseholdMember{}).Select("property_id").Where("user_id = ?", userID)).
		Order("id").Find(&properties).Error; err != nil {
		return nil, err
	}

	homes := make([]accountHome, 0, len(properties))
	for _, p := range properties {
		home := accountHome{property: p, member: memberOf[p.ID]}
		if err := db.Where("property_id = ? AND is_virtual = false AND user_id IS NOT NULL AND user_id <> ?", p.ID, userID).
			Order("id").Find(&home.others).Error; err != nil {
			return nil, err
		}
		homes = append(homes, home)
	}
	return homes, nil
}

// hasOpenBalance reports whether the member still owes, or is owed, any share
// in the property, long-term debts included.
func hasOpenBalance(db *gorm.DB, propertyID, memberID uint) (bool, error) {
	var n int64
	err := db.Model(&models.ExpenseSplit{}).
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expenses.property_id = ? AND expenses.deleted_at IS NULL", propertyID).
		Where("expense_splits.is_settled = ?", false).
		Where("(expense_splits.member_id = ? AND expenses.paid_by_member_id <> ?) OR (expenses.paid_by_member_id = ? AND expense_splits.member_id <> ?)",
			memberID, memberID, memberID, memberID).
		Count(&n).Error
	return n > 0, err
}

// accountDeleteCheck tells what deleting the account would do, and what has
// to happen first.
func accountDeleteCheck(db *gorm.DB, userID uint) (DeleteAccountCheckResponse, error) {
	resp := DeleteAccountCheckResponse{CanDelete: true}
	homes, err := accountHomes(db, userID)
	if err != nil {
		return resp, err
	}
	for _, home := range homes {
		if !home.shared() {
			resp.DataLossProperties = append(resp.DataLossProperties, home.property.Name)
			continue
		}
		leadsHome := home.property.UserID == userID || (home.member != nil && home.member.Role == "admin")
		if leadsHome && home.successor() == nil {
			resp.CanDelete = false
			members := make([]MemberOption, len(home.others))
			for i, m := range home.others {
				members[i] = MemberOption{MemberID: m.ID, UserID: *m.UserID, Name: m.Name}
			}
			resp.BlockingProperties = append(resp.BlockingProperties, BlockingPropertyInfo{
				PropertyID:   home.property.ID,
				PropertyName: home.property.Name,
				Members:      members,
			})
		}
		if home.member != nil {
			open, err := hasOpenBalance(db, home.property.ID, home.member.ID)
			if err != nil {
				return resp, err
			}
			if open {
				resp.CanDelete = false
				resp.OpenBalanceProperties = append(resp.OpenBalanceProperties, home.property.Name)
			}
		}
	}
	return resp, nil
}

// refuseAccountDelete answers 409 for a check that does not pass. The body
// carries the homes involved, so the UI can name them. The codes read the same
// whether the account is the caller's or one an admin is deleting.
func refuseAccountDelete(c *gin.Context, check DeleteAccountCheckResponse) {
	if len(check.BlockingProperties) > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":               "Every shared home needs another admin first",
			"error_code":          "admin_successor_required",
			"blocking_properties": check.BlockingProperties,
		})
		return
	}
	c.JSON(http.StatusConflict, gin.H{
		"error":                   "Balances in shared homes must be settled first",
		"error_code":              "open_balances_remaining",
		"open_balance_properties": check.OpenBalanceProperties,
	})
}

// deleteAccount deletes the user and everything that is theirs alone. Run it
// inside a transaction: it checks first, on the same transaction, and returns
// an *accountBlockedError if the check does not pass. Otherwise it stops at
// the first failing statement so the caller can roll back.
//
// TestDeleteAccount_CoversSchema fails when a foreign key into users is not
// handled here. Files on disk (the avatar) are the caller's, once committed.
func deleteAccount(tx *gorm.DB, userID uint) error {
	check, err := accountDeleteCheck(tx, userID)
	if err != nil {
		return err
	}
	if !check.CanDelete {
		return &accountBlockedError{check}
	}

	all := tx.Unscoped().Session(&gorm.Session{})
	homes, err := accountHomes(tx, userID)
	if err != nil {
		return err
	}
	for _, home := range homes {
		if !home.shared() {
			if err := cascadeDeleteProperty(tx, home.property.ID); err != nil {
				return fmt.Errorf("delete home %d: %w", home.property.ID, err)
			}
			continue
		}
		if err := leaveHome(all, userID, home); err != nil {
			return fmt.Errorf("leave home %d: %w", home.property.ID, err)
		}
	}
	// A home the account owns that was deleted before (soft) is nobody's to
	// keep, but still holds its whole tree.
	var trashed []uint
	if err := all.Model(&models.Property{}).Where("user_id = ? AND deleted_at IS NOT NULL", userID).
		Pluck("id", &trashed).Error; err != nil {
		return err
	}
	for _, id := range trashed {
		if err := cascadeDeleteProperty(tx, id); err != nil {
			return fmt.Errorf("delete removed home %d: %w", id, err)
		}
	}

	steps := []func() error{
		// Personal expenses go first: they must not keep a member row alive.
		func() error { return deletePersonalExpenses(tx, userID) },
		func() error { return detachMembers(tx, userID) },
		func() error { return all.Where("user_id = ?", userID).Delete(&models.ExpenseTemplate{}).Error },
		func() error { return handOverProjects(tx, userID) },
		func() error { return all.Where("user_id = ?", userID).Delete(&models.ProjectMember{}).Error },
		func() error { return deleteCategories(tx, userID) },
		func() error {
			return all.Model(&models.BillTemplate{}).Where("user_id = ?", userID).Update("user_id", nil).Error
		},
		func() error { return all.Where("user_id = ?", userID).Delete(&models.ContractTemplate{}).Error },
		func() error { return all.Where("user_id = ?", userID).Delete(&models.Notification{}).Error },
		func() error { return all.Where("user_id = ?", userID).Delete(&models.PropertyJoinRequest{}).Error },
		func() error { return all.Where("user_id = ?", userID).Delete(&models.UserSettings{}).Error },
		func() error { return all.Delete(&models.User{}, userID).Error },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// leaveHome hands a shared home the account owns to its successor, whom the
// check guarantees, and frees the place the account took in it. The member
// row itself is detachMembers'.
func leaveHome(all *gorm.DB, userID uint, home accountHome) error {
	if next := home.successor(); home.property.UserID == userID && next != nil {
		// is_current marks the owner's home of choice, not the successor's.
		if err := all.Model(&models.Property{}).Where("id = ?", home.property.ID).
			Updates(map[string]any{"user_id": *next.UserID, "is_current": false}).Error; err != nil {
			return err
		}
	}
	if home.member == nil {
		return nil
	}
	return all.Model(&models.Property{}).Where("id = ?", home.property.ID).
		UpdateColumn("residents", gorm.Expr("CASE WHEN residents > 1 THEN residents - 1 ELSE 1 END")).Error
}

// detachMembers unlinks every member row the account still has, removed ones
// included. A member the ledger refers to stays as a virtual member under the
// same name, so the household keeps its history. The rest are deleted.
func detachMembers(tx *gorm.DB, userID uint) error {
	all := tx.Unscoped().Session(&gorm.Session{})
	var memberIDs []uint
	if err := all.Model(&models.HouseholdMember{}).Where("user_id = ?", userID).Pluck("id", &memberIDs).Error; err != nil {
		return err
	}
	if len(memberIDs) == 0 {
		return nil
	}

	var kept []uint
	referenced := all.Model(&models.HouseholdMember{}).Where("id IN ?", memberIDs).Where(
		"id IN (SELECT member_id FROM expense_splits) OR id IN (SELECT paid_by_member_id FROM expenses) OR " +
			"id IN (SELECT from_member_id FROM settlements) OR id IN (SELECT to_member_id FROM settlements)")
	if err := referenced.Pluck("id", &kept).Error; err != nil {
		return err
	}

	if err := cleanupSplitDefaults(tx, memberIDs...); err != nil {
		return err
	}
	// Bills keep being paid: by whoever records them, as for a new service.
	if err := all.Model(&models.Utility{}).Where("paid_by_member_id IN ?", memberIDs).
		Update("paid_by_member_id", nil).Error; err != nil {
		return err
	}
	if len(kept) > 0 {
		if err := all.Model(&models.HouseholdMember{}).Where("id IN ?", kept).Updates(map[string]any{
			"user_id":    nil,
			"is_virtual": true,
			"role":       gorm.Expr("CASE WHEN role = 'admin' THEN 'member' ELSE role END"),
		}).Error; err != nil {
			return err
		}
	}
	return all.Where("id IN ? AND user_id = ?", memberIDs, userID).Delete(&models.HouseholdMember{}).Error
}

// deletePersonalExpenses deletes the expenses only the account could see: not
// split and not paying a bill. Shared ones are the household's ledger.
func deletePersonalExpenses(tx *gorm.DB, userID uint) error {
	var ids []uint
	if err := tx.Unscoped().Model(&models.Expense{}).
		Where("user_id = ? AND is_split = ? AND bill_id IS NULL AND bill_installment_id IS NULL", userID, false).
		Pluck("id", &ids).Error; err != nil {
		return err
	}
	return deleteExpenses(tx, ids)
}

// handOverProjects gives each project the account owns to another of its
// members, who becomes its owner. Projects nobody else is in are deleted.
func handOverProjects(tx *gorm.DB, userID uint) error {
	all := tx.Unscoped().Session(&gorm.Session{})
	var projects []models.Project
	if err := all.Where("user_id = ?", userID).Find(&projects).Error; err != nil {
		return err
	}
	var orphaned []uint
	for _, p := range projects {
		var next []models.ProjectMember
		if !p.DeletedAt.Valid {
			if err := all.Where("project_id = ? AND user_id <> ?", p.ID, userID).
				Order("role = 'owner' DESC, user_id").Limit(1).Find(&next).Error; err != nil {
				return err
			}
		}
		if len(next) == 0 {
			orphaned = append(orphaned, p.ID)
			continue
		}
		if err := all.Model(&models.ProjectMember{}).Where("project_id = ? AND user_id = ?", p.ID, next[0].UserID).
			Update("role", "owner").Error; err != nil {
			return err
		}
		// Save, not Update: the search index records the owner.
		p.UserID = next[0].UserID
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
	}
	return deleteProjects(tx, orphaned)
}

// deleteCategories deletes the account's own categories. One the ledger still
// files expenses under stays, without an owner, like an orphaned bill template:
// nobody can pick it again, but the expenses keep their label.
func deleteCategories(tx *gorm.DB, userID uint) error {
	all := tx.Unscoped().Session(&gorm.Session{})
	inUse := "id IN (SELECT category_id FROM expenses) OR id IN (SELECT category_id FROM expense_templates) OR " +
		"id IN (SELECT default_category_id FROM utilities WHERE default_category_id IS NOT NULL) OR " +
		"id IN (SELECT category_id FROM subcategories WHERE id IN (SELECT subcategory_id FROM expenses WHERE subcategory_id IS NOT NULL) " +
		"OR id IN (SELECT subcategory_id FROM expense_templates WHERE subcategory_id IS NOT NULL))"

	if err := all.Model(&models.Category{}).Where("user_id = ?", userID).Where(inUse).
		Update("user_id", nil).Error; err != nil {
		return err
	}
	var unused []uint
	if err := all.Model(&models.Category{}).Where("user_id = ?", userID).Pluck("id", &unused).Error; err != nil {
		return err
	}
	if len(unused) == 0 {
		return nil
	}
	if err := all.Where("category_id IN ?", unused).Delete(&models.Subcategory{}).Error; err != nil {
		return err
	}
	return all.Where("id IN ?", unused).Delete(&models.Category{}).Error
}
