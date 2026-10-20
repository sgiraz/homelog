package database

import (
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

type DeviceCategorySpec struct {
	Slug string
	Name string
	Icon string
}

var DefaultDeviceCategories = []DeviceCategorySpec{
	{Slug: "network", Name: "Netzwerk", Icon: "🌐"},
	{Slug: "computer", Name: "Computer", Icon: "💻"},
	{Slug: "mobile", Name: "Smartphone & Tablet", Icon: "📱"},
	{Slug: "tv_audio", Name: "TV & Audio", Icon: "📺"},
	{Slug: "appliances", Name: "Haushaltsgeräte", Icon: "🏠"},
	{Slug: "smart_home", Name: "Smart Home", Icon: "💡"},
	{Slug: "security", Name: "Sicherheit", Icon: "🔒"},
	{Slug: "other", Name: "Sonstiges", Icon: "📦"},
}

func SeedDefaultDeviceCategories(db *gorm.DB) error {
	for _, spec := range DefaultDeviceCategories {
		var category models.DeviceCategory

		err := db.
			Where("slug = ? AND user_id IS NULL", spec.Slug).
			First(&category).Error

		if err == nil {
			continue
		}

		if err != gorm.ErrRecordNotFound {
			return err
		}

		category = models.DeviceCategory{
			Slug:      spec.Slug,
			Name:      spec.Name,
			Icon:      spec.Icon,
			IsDefault: true,
		}

		if err := db.Create(&category).Error; err != nil {
			return err
		}
	}

	return nil
}
