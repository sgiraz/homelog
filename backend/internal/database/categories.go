package database

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"strings"

	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

// Built-in category labels, one file per language, keyed by slug. These are
// copies: the canonical strings live in frontend/src/i18n/locales/<lang>/
// categories.json — that is what Weblate edits and what the UI renders — and
// scripts/sync-category-names.mjs brings them here, because go:embed cannot
// reach outside the module. categories_names_test.go fails if they drift.
//
//go:embed locales/categories.*.json
var localeFiles embed.FS

// categoryNames is language -> slug -> label.
var categoryNames = loadCategoryNames()

func loadCategoryNames() map[string]map[string]string {
	entries, err := fs.Glob(localeFiles, "locales/categories.*.json")
	if err != nil {
		panic(fmt.Sprintf("category names: %v", err))
	}
	names := make(map[string]map[string]string, len(entries))
	for _, path := range entries {
		lang := strings.TrimSuffix(strings.TrimPrefix(path, "locales/categories."), ".json")
		raw, err := localeFiles.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("category names %s: %v", path, err))
		}
		var bySlug map[string]string
		if err := json.Unmarshal(raw, &bySlug); err != nil {
			panic(fmt.Sprintf("category names %s: %v", path, err))
		}
		names[lang] = bySlug
	}
	return names
}

// LocalizedCategoryLanguages lists the languages the embedded catalogue covers.
func LocalizedCategoryLanguages() []string {
	langs := make([]string, 0, len(categoryNames))
	for lang := range categoryNames {
		langs = append(langs, lang)
	}
	return langs
}

// Slugs of built-in categories/subcategories that backend code needs to look
// up by hand. Never match a default category by its Name — the name is a
// localized, user-renameable label; the slug is the stable identity.
const (
	SlugHome          = "home"
	SlugHomeUtilities = "home_utilities"
)

// SubcategorySpec describes one built-in subcategory.
type SubcategorySpec struct {
	Slug string
	// SeedName is the row written to subcategories.name. See CategorySpec.SeedName.
	SeedName string
}

// CategorySpec describes one built-in category. It deliberately carries no
// translations: those live in the embedded locale files, keyed by Slug, so
// adding a language never touches this file.
type CategorySpec struct {
	Slug string
	// SeedName is the row written to categories.name at seed time. It stays
	// Italian on purpose: it is what pre-slug databases were seeded with, and
	// therefore the key MigrateDefaultCategorySlugs matches legacy rows on.
	// Nothing displays it for a row that has a slug.
	SeedName string
	Icon     string
	Color    string
	Subs     []SubcategorySpec
}

// DefaultCategories is the catalogue of built-in categories: the slugs, the
// structure and the presentation bits. The labels are NOT here — they live in
// the locale files keyed by slug, so adding a language is a data change and
// never touches this list.
//
// Keep the slugs in sync with frontend/src/i18n/locales/*/categories.json.
var DefaultCategories = []CategorySpec{
	{
		Slug: SlugHome, SeedName: "Casa", Icon: "🏠", Color: "#3B82F6",
		Subs: []SubcategorySpec{
			{Slug: SlugHomeUtilities, SeedName: "Utenze"},
			{Slug: "home_ordinary_maintenance", SeedName: "Manutenzione ordinaria"},
			{Slug: "home_major_works", SeedName: "Lavori straordinari"},
			{Slug: "home_furniture", SeedName: "Arredamento"},
			{Slug: "home_appliances", SeedName: "Elettrodomestici"},
		},
	},
	{
		Slug: "food_dining", SeedName: "Alimentari e Ristorazione", Icon: "🍕", Color: "#10B981",
		Subs: []SubcategorySpec{
			{Slug: "food_groceries", SeedName: "Spesa supermercato"},
			{Slug: "food_restaurants", SeedName: "Ristoranti/Pizzerie"},
			{Slug: "food_cafes", SeedName: "Bar/Caffè"},
			{Slug: "food_delivery", SeedName: "Delivery"},
		},
	},
	{
		Slug: "transport", SeedName: "Trasporti", Icon: "🚗", Color: "#F59E0B",
		Subs: []SubcategorySpec{
			{Slug: "transport_fuel", SeedName: "Carburante"},
			{Slug: "transport_car_maintenance", SeedName: "Manutenzione auto"},
			{Slug: "transport_insurance", SeedName: "Assicurazione"},
			{Slug: "transport_parking_tolls", SeedName: "Parcheggi/Pedaggi"},
			{Slug: "transport_public", SeedName: "Trasporti pubblici"},
		},
	},
	{
		Slug: "entertainment", SeedName: "Intrattenimento", Icon: "🎬", Color: "#EC4899",
		Subs: []SubcategorySpec{
			{Slug: "entertainment_cinema_theatre", SeedName: "Cinema/Teatro"},
			{Slug: "entertainment_streaming", SeedName: "Streaming/Abbonamenti"},
			{Slug: "entertainment_books", SeedName: "Libri/Riviste"},
			{Slug: "entertainment_travel", SeedName: "Viaggi/Vacanze"},
		},
	},
	{
		Slug: "family", SeedName: "Famiglia", Icon: "👶", Color: "#14B8A6",
		Subs: []SubcategorySpec{
			{Slug: "family_kids_clothing", SeedName: "Abbigliamento bambini"},
			{Slug: "family_school", SeedName: "Scuola/Asilo"},
			{Slug: "family_toys", SeedName: "Giochi/Attrezzature"},
			{Slug: "family_health", SeedName: "Salute"},
		},
	},
	{
		Slug: "clothing_personal_care", SeedName: "Abbigliamento e Cura Personale", Icon: "👕", Color: "#F97316",
		Subs: []SubcategorySpec{
			{Slug: "clothing_adults", SeedName: "Abbigliamento adulti"},
			{Slug: "clothing_hairdresser", SeedName: "Parrucchiere/Barbiere"},
			{Slug: "clothing_personal_products", SeedName: "Prodotti cura personale"},
		},
	},
	{
		Slug: "education", SeedName: "Istruzione", Icon: "🎓", Color: "#6366F1",
		Subs: []SubcategorySpec{
			{Slug: "education_courses", SeedName: "Corsi/Formazione"},
			{Slug: "education_textbooks", SeedName: "Libri di testo"},
			{Slug: "education_supplies", SeedName: "Materiale scolastico"},
		},
	},
	{
		Slug: "technology", SeedName: "Tecnologia", Icon: "📱", Color: "#0EA5E9",
		Subs: []SubcategorySpec{
			{Slug: "technology_devices", SeedName: "Dispositivi"},
			{Slug: "technology_software", SeedName: "Abbonamenti software"},
			{Slug: "technology_repairs", SeedName: "Riparazioni"},
		},
	},
	{
		Slug: "gifts", SeedName: "Regali", Icon: "🎁", Color: "#EF4444",
		Subs: []SubcategorySpec{
			{Slug: "gifts_birthdays", SeedName: "Compleanni"},
			{Slug: "gifts_christmas", SeedName: "Natale"},
			{Slug: "gifts_anniversaries", SeedName: "Anniversari"},
			{Slug: "gifts_valentines", SeedName: "San Valentino"},
			{Slug: "gifts_graduations", SeedName: "Lauree/Diplomi"},
			{Slug: "gifts_births", SeedName: "Nascite/Battesimi"},
		},
	},
}

// categorySpecBySlug indexes DefaultCategories for O(1) lookups.
var categorySpecBySlug = func() map[string]CategorySpec {
	m := make(map[string]CategorySpec, len(DefaultCategories))
	for _, c := range DefaultCategories {
		m[c.Slug] = c
	}
	return m
}()

// DefaultCategoryName returns the built-in label for a category slug in the
// given language, or "" if the slug is not a built-in one. Used by
// server-generated output (the CSV export) that cannot go through vue-i18n.
// A language with no text for the slug falls back to English, matching the
// frontend's fallbackLocale.
func DefaultCategoryName(slug, lang string) string {
	if _, ok := categorySpecBySlug[slug]; !ok {
		return ""
	}
	if name := categoryNames[lang][slug]; name != "" {
		return name
	}
	return categoryNames["en"][slug]
}

// SeedDefaultCategories seeds global default categories (called once for first user)
func SeedDefaultCategories(db *gorm.DB) error {
	// Check if categories already exist
	var count int64
	db.Model(&models.Category{}).Count(&count)
	if count > 0 {
		return nil // Categories already exist
	}

	log.Println("🔄 Seeding default categories...")

	for _, spec := range DefaultCategories {
		cat := models.Category{
			Slug:      spec.Slug,
			Name:      spec.SeedName,
			Icon:      spec.Icon,
			Color:     spec.Color,
			IsDefault: true,
		}
		if err := db.Create(&cat).Error; err != nil {
			return err
		}

		subs := make([]models.Subcategory, len(spec.Subs))
		for i, s := range spec.Subs {
			subs[i] = models.Subcategory{CategoryID: cat.ID, Slug: s.Slug, Name: s.SeedName}
		}
		if len(subs) > 0 {
			if err := db.Create(&subs).Error; err != nil {
				return err
			}
		}
	}

	log.Println("✅ Default categories seeded")
	return nil
}

// MigrateDefaultCategorySlugs backfills the slug column on databases seeded
// before slugs existed, matching on the Italian name the seed wrote. A default
// category an admin has since renamed finds no match and keeps an empty slug,
// which is the intended outcome: the rename wins over the translation.
// Safe to call at every startup — it only fills rows whose slug is empty.
func MigrateDefaultCategorySlugs(db *gorm.DB) error {
	var categories []models.Category
	if err := db.Where("is_default = true AND (slug IS NULL OR slug = '')").
		Find(&categories).Error; err != nil {
		return err
	}

	nameToSpec := make(map[string]CategorySpec, len(DefaultCategories))
	for _, spec := range DefaultCategories {
		nameToSpec[spec.SeedName] = spec
	}

	for _, cat := range categories {
		spec, ok := nameToSpec[cat.Name]
		if !ok {
			continue
		}
		if err := db.Model(&models.Category{}).Where("id = ?", cat.ID).
			Update("slug", spec.Slug).Error; err != nil {
			return err
		}
	}

	// Backfill subcategory slugs the same way, scoped to the parent's slug so
	// two categories can't cross-match on a shared subcategory name.
	var parents []models.Category
	if err := db.Where("is_default = true AND slug != ''").Preload("Subcategories").
		Find(&parents).Error; err != nil {
		return err
	}
	for _, cat := range parents {
		spec, ok := categorySpecBySlug[cat.Slug]
		if !ok {
			continue
		}
		subNameToSlug := make(map[string]string, len(spec.Subs))
		for _, s := range spec.Subs {
			subNameToSlug[s.SeedName] = s.Slug
		}
		for _, sub := range cat.Subcategories {
			if sub.Slug != "" {
				continue
			}
			slug, ok := subNameToSlug[sub.Name]
			if !ok {
				continue
			}
			if err := db.Model(&models.Subcategory{}).Where("id = ?", sub.ID).
				Update("slug", slug).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// MigrateDefaultSubcategories adds missing subcategories to existing default
// categories. Matching is by slug, so a subcategory the user renamed is left
// alone instead of being re-created under its original name.
// Safe to call at every startup — it only inserts what's missing.
func MigrateDefaultSubcategories(db *gorm.DB) error {
	var categories []models.Category
	if err := db.Where("is_default = true AND slug != ''").Preload("Subcategories").
		Find(&categories).Error; err != nil {
		return err
	}

	for _, cat := range categories {
		spec, ok := categorySpecBySlug[cat.Slug]
		if !ok {
			continue
		}

		existing := make(map[string]bool, len(cat.Subcategories))
		for _, s := range cat.Subcategories {
			if s.Slug != "" {
				existing["slug:"+s.Slug] = true
			}
			existing["name:"+s.Name] = true
		}

		for _, s := range spec.Subs {
			if existing["slug:"+s.Slug] || existing["name:"+s.SeedName] {
				continue
			}
			sub := models.Subcategory{CategoryID: cat.ID, Slug: s.Slug, Name: s.SeedName}
			if err := db.Create(&sub).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
