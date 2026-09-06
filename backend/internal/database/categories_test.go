package database

import (
	"testing"

	"github.com/sgiraz/homelog/internal/models"
)

func TestSeedDefaultCategories_SetsSlugs(t *testing.T) {
	db := newTestDB(t)

	if err := SeedDefaultCategories(db); err != nil {
		t.Fatalf("SeedDefaultCategories: %v", err)
	}

	var cats []models.Category
	if err := db.Preload("Subcategories").Find(&cats).Error; err != nil {
		t.Fatalf("load categories: %v", err)
	}
	if len(cats) != len(DefaultCategories) {
		t.Fatalf("seeded %d categories, want %d", len(cats), len(DefaultCategories))
	}
	for _, c := range cats {
		if c.Slug == "" {
			t.Errorf("category %q seeded without a slug", c.Name)
		}
		for _, s := range c.Subcategories {
			if s.Slug == "" {
				t.Errorf("subcategory %q of %q seeded without a slug", s.Name, c.Name)
			}
		}
	}

	// Slugs must be unique across categories and subcategories alike: they are
	// i18n keys living in one flat namespace.
	seen := map[string]string{}
	for _, c := range cats {
		if prev, dup := seen[c.Slug]; dup {
			t.Errorf("slug %q used by both %q and %q", c.Slug, prev, c.Name)
		}
		seen[c.Slug] = c.Name
		for _, s := range c.Subcategories {
			if prev, dup := seen[s.Slug]; dup {
				t.Errorf("slug %q used by both %q and %q", s.Slug, prev, s.Name)
			}
			seen[s.Slug] = s.Name
		}
	}
}

// A database seeded before slugs existed must get them backfilled from the
// Italian names the old seed wrote.
func TestMigrateDefaultCategorySlugs_BackfillsLegacyRows(t *testing.T) {
	db := newTestDB(t)

	legacy := models.Category{Name: "Casa", Icon: "🏠", IsDefault: true}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy category: %v", err)
	}
	if err := db.Create(&models.Subcategory{CategoryID: legacy.ID, Name: "Utenze"}).Error; err != nil {
		t.Fatalf("create legacy subcategory: %v", err)
	}

	if err := MigrateDefaultCategorySlugs(db); err != nil {
		t.Fatalf("MigrateDefaultCategorySlugs: %v", err)
	}

	var got models.Category
	if err := db.Preload("Subcategories").First(&got, legacy.ID).Error; err != nil {
		t.Fatalf("reload category: %v", err)
	}
	if got.Slug != SlugHome {
		t.Errorf("category slug = %q, want %q", got.Slug, SlugHome)
	}
	if len(got.Subcategories) != 1 || got.Subcategories[0].Slug != SlugHomeUtilities {
		t.Errorf("subcategory slug = %+v, want %q", got.Subcategories, SlugHomeUtilities)
	}
}

// A default category an admin renamed must keep an empty slug, so the UI shows
// the admin's label instead of a translated one.
func TestMigrateDefaultCategorySlugs_SkipsRenamedCategory(t *testing.T) {
	db := newTestDB(t)

	renamed := models.Category{Name: "Casa al mare", IsDefault: true}
	if err := db.Create(&renamed).Error; err != nil {
		t.Fatalf("create renamed category: %v", err)
	}

	if err := MigrateDefaultCategorySlugs(db); err != nil {
		t.Fatalf("MigrateDefaultCategorySlugs: %v", err)
	}

	var got models.Category
	if err := db.First(&got, renamed.ID).Error; err != nil {
		t.Fatalf("reload category: %v", err)
	}
	if got.Slug != "" {
		t.Errorf("renamed category got slug %q, want none", got.Slug)
	}
}

// The subcategory top-up must match on slug and leave renamed rows alone
// instead of re-creating them under their original name.
func TestMigrateDefaultSubcategories_AddsMissingWithoutDuplicating(t *testing.T) {
	db := newTestDB(t)
	if err := SeedDefaultCategories(db); err != nil {
		t.Fatalf("SeedDefaultCategories: %v", err)
	}

	var home models.Category
	if err := db.Where("slug = ?", SlugHome).Preload("Subcategories").First(&home).Error; err != nil {
		t.Fatalf("load home category: %v", err)
	}
	before := len(home.Subcategories)

	// Drop one, and rename another to simulate user edits.
	if err := db.Where("category_id = ? AND slug = ?", home.ID, "home_furniture").
		Delete(&models.Subcategory{}).Error; err != nil {
		t.Fatalf("delete subcategory: %v", err)
	}
	if err := db.Model(&models.Subcategory{}).
		Where("category_id = ? AND slug = ?", home.ID, SlugHomeUtilities).
		Update("name", "Bollette").Error; err != nil {
		t.Fatalf("rename subcategory: %v", err)
	}

	if err := MigrateDefaultSubcategories(db); err != nil {
		t.Fatalf("MigrateDefaultSubcategories: %v", err)
	}

	var subs []models.Subcategory
	if err := db.Where("category_id = ?", home.ID).Find(&subs).Error; err != nil {
		t.Fatalf("reload subcategories: %v", err)
	}
	if len(subs) != before {
		t.Fatalf("got %d subcategories, want %d (missing one re-added, renamed one untouched)", len(subs), before)
	}
	for _, s := range subs {
		if s.Slug == SlugHomeUtilities && s.Name != "Bollette" {
			t.Errorf("renamed subcategory was overwritten: name = %q", s.Name)
		}
	}
}

func TestDefaultCategoryName_LocalizesBuiltInsOnly(t *testing.T) {
	if got := DefaultCategoryName(SlugHome, "en"); got != "Home" {
		t.Errorf("EN name for %q = %q, want \"Home\"", SlugHome, got)
	}
	if got := DefaultCategoryName(SlugHome, "it"); got != "Casa" {
		t.Errorf("IT name for %q = %q, want \"Casa\"", SlugHome, got)
	}
	if got := DefaultCategoryName("", "en"); got != "" {
		t.Errorf("empty slug resolved to %q, want \"\"", got)
	}
}
