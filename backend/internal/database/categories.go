package database

import (
	"log"

	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

// Slugs of built-in categories/subcategories that backend code needs to look
// up by hand. Never match a default category by its Name — the name is a
// localized, user-renameable label; the slug is the stable identity.
const (
	SlugHome          = "home"
	SlugHomeUtilities = "home_utilities"
)

// SubcategorySpec describes one built-in subcategory.
type SubcategorySpec struct {
	Slug   string
	NameIT string
	NameEN string
}

// CategorySpec describes one built-in category.
type CategorySpec struct {
	Slug   string
	NameIT string
	NameEN string
	Icon   string
	Color  string
	Subs   []SubcategorySpec
}

// DefaultCategories is the catalogue of built-in categories. NameIT is what
// lands in the categories.name column at seed time (and what old clients and
// raw DB queries see); the UI never shows it for a row that has a slug — it
// renders t("categories.<slug>") instead. NameEN exists only so server-side
// output (CSV export) can be localized without a second table.
//
// Keep the slugs in sync with frontend/src/i18n/locales/*/categories.json.
var DefaultCategories = []CategorySpec{
	{
		Slug: SlugHome, NameIT: "Casa", NameEN: "Home", Icon: "🏠", Color: "#3B82F6",
		Subs: []SubcategorySpec{
			{Slug: SlugHomeUtilities, NameIT: "Utenze", NameEN: "Utilities"},
			{Slug: "home_ordinary_maintenance", NameIT: "Manutenzione ordinaria", NameEN: "Routine maintenance"},
			{Slug: "home_major_works", NameIT: "Lavori straordinari", NameEN: "Major works"},
			{Slug: "home_furniture", NameIT: "Arredamento", NameEN: "Furniture"},
			{Slug: "home_appliances", NameIT: "Elettrodomestici", NameEN: "Appliances"},
		},
	},
	{
		Slug: "food_dining", NameIT: "Alimentari e Ristorazione", NameEN: "Food & Dining", Icon: "🍕", Color: "#10B981",
		Subs: []SubcategorySpec{
			{Slug: "food_groceries", NameIT: "Spesa supermercato", NameEN: "Groceries"},
			{Slug: "food_restaurants", NameIT: "Ristoranti/Pizzerie", NameEN: "Restaurants"},
			{Slug: "food_cafes", NameIT: "Bar/Caffè", NameEN: "Cafés"},
			{Slug: "food_delivery", NameIT: "Delivery", NameEN: "Delivery"},
		},
	},
	{
		Slug: "transport", NameIT: "Trasporti", NameEN: "Transport", Icon: "🚗", Color: "#F59E0B",
		Subs: []SubcategorySpec{
			{Slug: "transport_fuel", NameIT: "Carburante", NameEN: "Fuel"},
			{Slug: "transport_car_maintenance", NameIT: "Manutenzione auto", NameEN: "Car maintenance"},
			{Slug: "transport_insurance", NameIT: "Assicurazione", NameEN: "Insurance"},
			{Slug: "transport_parking_tolls", NameIT: "Parcheggi/Pedaggi", NameEN: "Parking & tolls"},
			{Slug: "transport_public", NameIT: "Trasporti pubblici", NameEN: "Public transport"},
		},
	},
	{
		Slug: "entertainment", NameIT: "Intrattenimento", NameEN: "Entertainment", Icon: "🎬", Color: "#EC4899",
		Subs: []SubcategorySpec{
			{Slug: "entertainment_cinema_theatre", NameIT: "Cinema/Teatro", NameEN: "Cinema & theatre"},
			{Slug: "entertainment_streaming", NameIT: "Streaming/Abbonamenti", NameEN: "Streaming & subscriptions"},
			{Slug: "entertainment_books", NameIT: "Libri/Riviste", NameEN: "Books & magazines"},
			{Slug: "entertainment_travel", NameIT: "Viaggi/Vacanze", NameEN: "Travel & holidays"},
		},
	},
	{
		Slug: "family", NameIT: "Famiglia", NameEN: "Family", Icon: "👶", Color: "#14B8A6",
		Subs: []SubcategorySpec{
			{Slug: "family_kids_clothing", NameIT: "Abbigliamento bambini", NameEN: "Kids clothing"},
			{Slug: "family_school", NameIT: "Scuola/Asilo", NameEN: "School & daycare"},
			{Slug: "family_toys", NameIT: "Giochi/Attrezzature", NameEN: "Toys & equipment"},
			{Slug: "family_health", NameIT: "Salute", NameEN: "Health"},
		},
	},
	{
		Slug: "clothing_personal_care", NameIT: "Abbigliamento e Cura Personale", NameEN: "Clothing & Personal Care", Icon: "👕", Color: "#F97316",
		Subs: []SubcategorySpec{
			{Slug: "clothing_adults", NameIT: "Abbigliamento adulti", NameEN: "Adult clothing"},
			{Slug: "clothing_hairdresser", NameIT: "Parrucchiere/Barbiere", NameEN: "Hairdresser & barber"},
			{Slug: "clothing_personal_products", NameIT: "Prodotti cura personale", NameEN: "Personal care products"},
		},
	},
	{
		Slug: "education", NameIT: "Istruzione", NameEN: "Education", Icon: "🎓", Color: "#6366F1",
		Subs: []SubcategorySpec{
			{Slug: "education_courses", NameIT: "Corsi/Formazione", NameEN: "Courses & training"},
			{Slug: "education_textbooks", NameIT: "Libri di testo", NameEN: "Textbooks"},
			{Slug: "education_supplies", NameIT: "Materiale scolastico", NameEN: "School supplies"},
		},
	},
	{
		Slug: "technology", NameIT: "Tecnologia", NameEN: "Technology", Icon: "📱", Color: "#0EA5E9",
		Subs: []SubcategorySpec{
			{Slug: "technology_devices", NameIT: "Dispositivi", NameEN: "Devices"},
			{Slug: "technology_software", NameIT: "Abbonamenti software", NameEN: "Software subscriptions"},
			{Slug: "technology_repairs", NameIT: "Riparazioni", NameEN: "Repairs"},
		},
	},
	{
		Slug: "gifts", NameIT: "Regali", NameEN: "Gifts", Icon: "🎁", Color: "#EF4444",
		Subs: []SubcategorySpec{
			{Slug: "gifts_birthdays", NameIT: "Compleanni", NameEN: "Birthdays"},
			{Slug: "gifts_christmas", NameIT: "Natale", NameEN: "Christmas"},
			{Slug: "gifts_anniversaries", NameIT: "Anniversari", NameEN: "Anniversaries"},
			{Slug: "gifts_valentines", NameIT: "San Valentino", NameEN: "Valentine's Day"},
			{Slug: "gifts_graduations", NameIT: "Lauree/Diplomi", NameEN: "Graduations"},
			{Slug: "gifts_births", NameIT: "Nascite/Battesimi", NameEN: "Births & christenings"},
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

// DefaultCategoryName returns the built-in name for a category slug in the
// given language, or "" if the slug is not a built-in one. Used for
// server-generated output (CSV export) that can't go through vue-i18n.
func DefaultCategoryName(slug, lang string) string {
	spec, ok := categorySpecBySlug[slug]
	if !ok {
		return ""
	}
	if lang == "it" {
		return spec.NameIT
	}
	return spec.NameEN
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
			Name:      spec.NameIT,
			Icon:      spec.Icon,
			Color:     spec.Color,
			IsDefault: true,
		}
		if err := db.Create(&cat).Error; err != nil {
			return err
		}

		subs := make([]models.Subcategory, len(spec.Subs))
		for i, s := range spec.Subs {
			subs[i] = models.Subcategory{CategoryID: cat.ID, Slug: s.Slug, Name: s.NameIT}
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
		nameToSpec[spec.NameIT] = spec
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
			subNameToSlug[s.NameIT] = s.Slug
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
			if existing["slug:"+s.Slug] || existing["name:"+s.NameIT] {
				continue
			}
			sub := models.Subcategory{CategoryID: cat.ID, Slug: s.Slug, Name: s.NameIT}
			if err := db.Create(&sub).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
