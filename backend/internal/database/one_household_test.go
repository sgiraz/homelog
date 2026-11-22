package database

import (
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/sgiraz/homelog/internal/models"
)

// An account belongs to one household at most, and the schema says so: the
// handlers check it, a unique index makes sure. Virtual members have no
// account, and a removed member no longer counts.
func TestHouseholdMember_OneHouseholdPerAccount(t *testing.T) {
	db, err := Open(":memory:", &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	user := models.User{Email: "alice@example.com", PasswordHash: "x", Name: "Alice", IsActive: true}
	first, second := models.Household{Name: "Flat"}, models.Household{Name: "Lake house"}
	for _, row := range []any{&user, &first, &second} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	member := func(household uint, userID *uint, deleted bool) error {
		m := models.HouseholdMember{HouseholdID: household, UserID: userID, Name: "Alice", IsVirtual: userID == nil}
		if deleted {
			m.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
		}
		return db.Create(&m).Error
	}

	if err := member(first.ID, &user.ID, false); err != nil {
		t.Fatalf("first membership: %v", err)
	}
	if err := member(second.ID, &user.ID, false); err == nil {
		t.Error("the same account joined a second household")
	}
	if err := member(first.ID, &user.ID, false); err == nil {
		t.Error("the same account joined its household twice")
	}
	if err := member(second.ID, &user.ID, true); err != nil {
		t.Errorf("a removed membership elsewhere: %v", err)
	}
	for _, h := range []uint{first.ID, second.ID, second.ID} {
		if err := member(h, nil, false); err != nil {
			t.Errorf("virtual member in household %d: %v", h, err)
		}
	}
}
