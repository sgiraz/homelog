package handlers

import (
	"errors"
	"log"

	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/models"
)

// createHousehold creates a household with its default split settings and the
// creator as its admin member.
func createHousehold(tx *gorm.DB, name string, creator models.User) (models.Household, models.HouseholdMember, error) {
	household := models.Household{Name: name}
	if err := tx.Create(&household).Error; err != nil {
		return household, models.HouseholdMember{}, err
	}
	if err := tx.Create(&models.HouseholdSettings{
		HouseholdID:      household.ID,
		SplitMode:        false,
		DefaultSplitType: "equal",
	}).Error; err != nil {
		return household, models.HouseholdMember{}, err
	}
	admin := models.HouseholdMember{
		HouseholdID: household.ID,
		UserID:      &creator.ID,
		Name:        creator.Name,
		Role:        "admin",
		IsVirtual:   false,
	}
	if err := tx.Create(&admin).Error; err != nil {
		return household, admin, err
	}
	return household, admin, nil
}

// makePrimaryHome makes home the primary home of its household, the default
// home for its Home expenses. There is one per household.
func makePrimaryHome(tx *gorm.DB, home models.Property) error {
	if err := tx.Model(&models.Property{}).
		Where("household_id = ? AND id != ?", home.HouseholdID, home.ID).
		Update("is_current", false).Error; err != nil {
		return err
	}
	return tx.Model(&models.Property{}).Where("id = ?", home.ID).Update("is_current", true).Error
}

// ensurePrimaryHome promotes the household's oldest home in use when none is
// primary.
func ensurePrimaryHome(tx *gorm.DB, householdID uint) error {
	var primaries int64
	if err := tx.Model(&models.Property{}).
		Where("household_id = ? AND is_current = ?", householdID, true).
		Count(&primaries).Error; err != nil || primaries > 0 {
		return err
	}
	var oldest models.Property
	if err := tx.Where("household_id = ?", householdID).Order("end_date IS NOT NULL, created_at ASC, id ASC").
		Limit(1).Find(&oldest).Error; err != nil || oldest.ID == 0 {
		return err
	}
	return makePrimaryHome(tx, oldest)
}

// errHomeNotInHousehold is expenseHome refusing a home of another household.
var errHomeNotInHousehold = errors.New("home not in household")

// errHouseholdHasNoHome is expenseHome finding no home for a Home expense.
var errHouseholdHasNoHome = errors.New("household has no home")

// expenseHome returns the home an expense belongs to. Only the Home category
// carries one: the requested home, which must be the household's, or else
// the household's primary home. Every other category has none.
func expenseHome(db *gorm.DB, category models.Category, householdID uint, requested *uint) (*uint, error) {
	if category.Slug != database.SlugHome {
		return nil, nil
	}
	var home models.Property
	query := db.Select("id").Where("household_id = ?", householdID)
	if requested != nil {
		query = query.Where("id = ?", *requested)
	} else {
		query = query.Order("is_current DESC, end_date IS NOT NULL, created_at ASC")
	}
	if err := query.Limit(1).Find(&home).Error; err != nil {
		return nil, err
	}
	if home.ID == 0 {
		if requested != nil {
			return nil, errHomeNotInHousehold
		}
		return nil, errHouseholdHasNoHome
	}
	return &home.ID, nil
}

// propertyIDsOfMember lists the homes of every household the user is a member
// of. A failed lookup yields none: the caller then finds nothing to open.
func propertyIDsOfMember(db *gorm.DB, userID uint) []uint {
	ids := []uint{}
	if err := db.Model(&models.Property{}).
		Where("household_id IN (?)",
			db.Model(&models.HouseholdMember{}).Select("household_id").Where("user_id = ?", userID)).
		Pluck("id", &ids).Error; err != nil {
		log.Printf("ERROR listing homes of user %d: %v", userID, err)
	}
	return ids
}

// accessiblePropertyIDs lists the homes the user can open: those of every
// household they are a member of, plus the ones they own.
func accessiblePropertyIDs(db *gorm.DB, userID uint) ([]uint, error) {
	ids := []uint{}
	err := db.Model(&models.Property{}).
		Where("user_id = ? OR household_id IN (?)", userID,
			db.Model(&models.HouseholdMember{}).Select("household_id").Where("user_id = ?", userID)).
		Pluck("id", &ids).Error
	return ids, err
}
