package database

import (
	"testing"
	"time"

	"github.com/sgiraz/homelog/internal/models"
)

// Bill templates used to belong to their author. The move gives each one the
// property it serves and must land every row exactly once.
func TestMoveBillTemplatesToProperty(t *testing.T) {
	db := newTestDB(t)
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	mustCreate := func(recs ...any) {
		t.Helper()
		for _, r := range recs {
			if err := db.Create(r).Error; err != nil {
				t.Fatalf("create %T: %v", r, err)
			}
		}
	}

	// The table as the per-user build left it.
	mustExec(`DROP TABLE bill_templates`)
	mustExec("CREATE TABLE `bill_templates` (`id` integer PRIMARY KEY AUTOINCREMENT,`created_at` datetime,`updated_at` datetime," +
		"`user_id` integer NOT NULL,`name` text NOT NULL,`provider` text NOT NULL,`utility_type` text NOT NULL," +
		"`is_default` numeric NOT NULL DEFAULT false,`extraction_rules` text," +
		"CONSTRAINT `fk_bill_templates_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`))")

	now := time.Now()
	author := &models.User{Email: "author@example.com", PasswordHash: "x", Name: "Author", IsActive: true}
	partner := &models.User{Email: "partner@example.com", PasswordHash: "x", Name: "Partner", IsActive: true}
	lodger := &models.User{Email: "lodger@example.com", PasswordHash: "x", Name: "Lodger", IsActive: true}
	loner := &models.User{Email: "loner@example.com", PasswordHash: "x", Name: "Loner", IsActive: true}
	mustCreate(author, partner, lodger, loner)

	home := models.Property{UserID: author.ID, Name: "Home", StartDate: now, IsCurrent: true}
	cabin := models.Property{UserID: author.ID, Name: "Cabin", StartDate: now}
	flat := models.Property{UserID: lodger.ID, Name: "Flat", StartDate: now}
	mustCreate(&cabin, &home, &flat)
	mustCreate(
		&models.HouseholdMember{PropertyID: cabin.ID, UserID: &author.ID, Name: "Author"},
		&models.HouseholdMember{PropertyID: home.ID, UserID: &author.ID, Name: "Author"},
		&models.HouseholdMember{PropertyID: home.ID, UserID: &partner.ID, Name: "Partner"},
		&models.HouseholdMember{PropertyID: flat.ID, UserID: &lodger.ID, Name: "Lodger"},
	)

	insert := func(userID uint, provider, utilityType string, isDefault bool) uint {
		t.Helper()
		mustExec(`INSERT INTO bill_templates (created_at, updated_at, user_id, name, provider, utility_type, is_default, extraction_rules)
			VALUES (?, ?, ?, ?, ?, ?, ?, '[]')`, now, now, userID, provider, provider, utilityType, isDefault)
		var id uint
		db.Raw(`SELECT MAX(id) FROM bill_templates`).Scan(&id)
		return id
	}
	byDefault := insert(author.ID, "Acqua", "water", false)
	byProvider := insert(author.ID, "  ENEL ", "electricity", false)
	unmatched := insert(author.ID, "Nobody", "gas", false)
	firstHousehold := insert(lodger.ID, "Etra", "water", false)
	orphan := insert(loner.ID, "Etra", "water", false)
	shared := insert(author.ID, "Fibra", "internet", false)
	olderDefault := insert(author.ID, "Eni", "gas", true)
	newerDefault := insert(partner.ID, "eni", "gas", true)

	ref := func(id uint) *uint { return &id }
	mustCreate(
		&models.Utility{UserID: author.ID, PropertyID: cabin.ID, Type: "water", Provider: "Other", DefaultBillTemplateID: ref(byDefault)},
		&models.Utility{UserID: author.ID, PropertyID: cabin.ID, Type: "electricity", Provider: "Enel"},
		&models.Utility{UserID: author.ID, PropertyID: home.ID, Type: "internet", Provider: "Fibra", DefaultBillTemplateID: ref(shared)},
		&models.Utility{UserID: author.ID, PropertyID: cabin.ID, Type: "internet", Provider: "Fibra", DefaultBillTemplateID: ref(shared)},
	)

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	propertyOf := func(id uint) uint {
		t.Helper()
		var tpl models.BillTemplate
		if err := db.First(&tpl, id).Error; err != nil {
			t.Fatalf("template %d: %v", id, err)
		}
		return tpl.PropertyID
	}
	for name, c := range map[string]struct{ id, want uint }{
		"default of a service":     {byDefault, cabin.ID},
		"provider match":           {byProvider, cabin.ID},
		"current property":         {unmatched, home.ID},
		"first household":          {firstHousehold, flat.ID},
		"default of first service": {shared, home.ID},
	} {
		if got := propertyOf(c.id); got != c.want {
			t.Errorf("%s: property %d, want %d", name, got, c.want)
		}
	}

	var count int64
	db.Model(&models.BillTemplate{}).Where("id = ?", orphan).Count(&count)
	if count != 0 {
		t.Error("a template no household can reach survived")
	}

	var cabinInternet models.Utility
	db.Where("property_id = ? AND type = ?", cabin.ID, "internet").First(&cabinInternet)
	if cabinInternet.DefaultBillTemplateID == nil || *cabinInternet.DefaultBillTemplateID == shared ||
		propertyOf(*cabinInternet.DefaultBillTemplateID) != cabin.ID {
		t.Errorf("the cabin's internet service should default to its own copy, got %v", cabinInternet.DefaultBillTemplateID)
	}

	var older, newer models.BillTemplate
	db.First(&older, olderDefault)
	db.First(&newer, newerDefault)
	if older.IsDefault || !newer.IsDefault {
		t.Errorf("one default per property+provider+type: older=%v newer=%v", older.IsDefault, newer.IsDefault)
	}

	var before, after int64
	db.Model(&models.BillTemplate{}).Count(&before)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	db.Model(&models.BillTemplate{}).Count(&after)
	if before != after {
		t.Errorf("a second boot changed the template count: %d -> %d", before, after)
	}

	// The column is NOT NULL once backfilled.
	if err := db.Exec(`INSERT INTO bill_templates (name, provider, utility_type, is_default) VALUES ('x', 'x', 'gas', 0)`).Error; err == nil {
		t.Error("a template without a property was accepted")
	}
}
