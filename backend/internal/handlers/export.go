package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/i18n"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

type ExportHandler struct {
	db *gorm.DB
}

func NewExportHandler(db *gorm.DB) *ExportHandler {
	return &ExportHandler{db: db}
}

// writeCSV writes a UTF-8 BOM CSV file to the response with the given headers and rows.
func writeCSV(c *gin.Context, filename string, headers []string, rows [][]string) {
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	// Write UTF-8 BOM for Excel compatibility
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(c.Writer)
	w.Write(headers)
	for _, row := range rows {
		w.Write(row)
	}
	w.Flush()
}

type ExportData struct {
	ExportedAt        time.Time                  `json:"exported_at"`
	Version           string                     `json:"version"`
	User              models.User                `json:"user"`
	Properties        []models.Property          `json:"properties"`
	HouseholdMembers  []models.HouseholdMember   `json:"household_members"`
	Categories        []models.Category          `json:"categories"`
	DeviceCategories  []models.DeviceCategory    `json:"device_categories"`
	Expenses          []models.Expense           `json:"expenses"`
	ExpenseSplits     []models.ExpenseSplit      `json:"expense_splits"`
	Settlements       []models.Settlement        `json:"settlements"`
	Devices           []models.Device            `json:"devices"`
	Utilities         []models.Utility           `json:"utilities"`
	MeterReadings     []models.MeterReading      `json:"meter_readings"`
	Bills             []models.Bill              `json:"bills"`
	Projects          []models.Project           `json:"projects"`
	UserSettings      models.UserSettings        `json:"user_settings"`
	HouseholdSettings []models.HouseholdSettings `json:"household_settings"`
}

// ExportAll exports all user data as a JSON download, or expenses as CSV when format=csv.
// GET /api/v1/export/all
func (h *ExportHandler) ExportAll(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	// CSV format: export expenses as CSV (single-sheet summary)
	if c.Query("format") == "csv" {
		var expenses []models.Expense
		h.db.Where("user_id = ?", userID).
			Preload("Category").
			Preload("Project").
			Preload("Splits").
			Preload("PaidBy").
			Order("date DESC").
			Find(&expenses)

		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("homelog_spese_%s.csv", timestamp)
		headers := []string{"Date", "Description", "Amount", "Category", "Project", "Paid By", "Split", "Notes"}
		lang := i18n.UserLanguage(h.db, userID)
		rows := make([][]string, 0, len(expenses))
		for _, e := range expenses {
			rows = append(rows, expenseToCSVRow(e, lang))
		}
		log.Printf("✅ CSV export (expenses) done: %d rows for user %d", len(rows), userID)
		writeCSV(c, filename, headers, rows)
		return
	}

	log.Printf("📦 Starting full export for user %d", userID)

	data := ExportData{
		ExportedAt: time.Now(),
		Version:    "1.0.0",
	}

	// User
	h.db.First(&data.User, userID)

	// Properties owned by this user
	h.db.Where("user_id = ?", userID).Find(&data.Properties)
	propertyIDs := make([]uint, len(data.Properties))
	for i, p := range data.Properties {
		propertyIDs[i] = p.ID
	}

	// Household members across all user's properties
	if len(propertyIDs) > 0 {
		h.db.Where("property_id IN ?", propertyIDs).Find(&data.HouseholdMembers)
	}

	// Categories: user's custom + system defaults
	h.db.Where("user_id = ? OR user_id IS NULL", userID).Find(&data.Categories)

	// Device categories: user's custom + system defaults
	h.db.Where("user_id = ? OR user_id IS NULL", userID).Find(&data.DeviceCategories)

	// Expenses
	h.db.Where("user_id = ?", userID).Find(&data.Expenses)
	expenseIDs := make([]uint, len(data.Expenses))
	for i, e := range data.Expenses {
		expenseIDs[i] = e.ID
	}

	// Devices
	if len(propertyIDs) > 0 {
		h.db.Where("property_id IN ?", propertyIDs).Find(&data.Devices)
	}

	// Expense splits
	if len(expenseIDs) > 0 {
		h.db.Where("expense_id IN ?", expenseIDs).Find(&data.ExpenseSplits)
	}

	// Settlements
	if len(propertyIDs) > 0 {
		h.db.Where("property_id IN ?", propertyIDs).Find(&data.Settlements)
	}

	// Utilities + readings + bills
	h.db.Where("user_id = ?", userID).Find(&data.Utilities)
	utilityIDs := make([]uint, len(data.Utilities))
	for i, u := range data.Utilities {
		utilityIDs[i] = u.ID
	}
	if len(utilityIDs) > 0 {
		h.db.Where("utility_id IN ?", utilityIDs).Find(&data.MeterReadings)
		h.db.Where("utility_id IN ?", utilityIDs).Find(&data.Bills)
	}

	// Projects
	h.db.Where("user_id = ?", userID).Find(&data.Projects)

	// User settings
	h.db.Where("user_id = ?", userID).First(&data.UserSettings)

	// Household settings for all properties
	if len(propertyIDs) > 0 {
		h.db.Where("property_id IN ?", propertyIDs).Find(&data.HouseholdSettings)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("homelog_backup_%s.json", timestamp)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")

	log.Printf("✅ Full export done: %d expenses, %d utilities, %d projects",
		len(data.Expenses), len(data.Utilities), len(data.Projects))

	c.JSON(http.StatusOK, data)
}

// expenseToCSVRow converts an Expense to a slice of strings for CSV output.
// lang localizes built-in category names; user-created ones export as stored.
func expenseToCSVRow(e models.Expense, lang string) []string {
	date := e.Date.Format("2006-01-02")
	amount := fmt.Sprintf("%.2f", e.Amount)
	category := e.Category.Name
	if localized := database.DefaultCategoryName(e.Category.Slug, lang); localized != "" {
		category = localized
	}
	project := ""
	if e.Project != nil {
		project = e.Project.Name
	}
	paidBy := ""
	if e.PaidBy != nil {
		paidBy = e.PaidBy.Name
	}
	split := ""
	if len(e.Splits) > 0 {
		split = fmt.Sprintf("%d split(s)", len(e.Splits))
	}
	return []string{date, e.Description, amount, category, project, paidBy, split, ""}
}

// ExportExpenses exports only the user's expenses.
// GET /api/v1/export/expenses
func (h *ExportHandler) ExportExpenses(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var expenses []models.Expense
	h.db.Where("user_id = ?", userID).
		Preload("Category").
		Preload("Project").
		Preload("Splits").
		Preload("PaidBy").
		Order("date DESC").
		Find(&expenses)

	timestamp := time.Now().Format("2006-01-02_15-04-05")

	if c.Query("format") == "csv" {
		filename := fmt.Sprintf("homelog_expenses_%s.csv", timestamp)
		headers := []string{"Date", "Description", "Amount", "Category", "Project", "Paid By", "Split", "Notes"}
		lang := i18n.UserLanguage(h.db, userID)
		rows := make([][]string, 0, len(expenses))
		for _, e := range expenses {
			rows = append(rows, expenseToCSVRow(e, lang))
		}
		log.Printf("✅ Exported %d expenses (CSV) for user %d", len(expenses), userID)
		writeCSV(c, filename, headers, rows)
		return
	}

	filename := fmt.Sprintf("homelog_expenses_%s.json", timestamp)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")

	log.Printf("✅ Exported %d expenses for user %d", len(expenses), userID)

	c.JSON(http.StatusOK, gin.H{
		"exported_at": time.Now(),
		"version":     "1.0.0",
		"type":        "expenses",
		"count":       len(expenses),
		"expenses":    expenses,
	})
}

// ExportUtilities exports the user's utilities with readings and bills.
// GET /api/v1/export/utilities
func (h *ExportHandler) ExportUtilities(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var utilities []models.Utility
	h.db.Where("user_id = ?", userID).
		Preload("Readings").
		Preload("Bills").
		Find(&utilities)

	timestamp := time.Now().Format("2006-01-02_15-04-05")

	if c.Query("format") == "csv" {
		filename := fmt.Sprintf("homelog_utilities_%s.csv", timestamp)
		headers := []string{"Provider", "Type", "Bill Number", "Amount", "Period Start", "Period End", "Is Paid"}
		rows := [][]string{}
		for _, u := range utilities {
			for _, b := range u.Bills {
				periodStart := ""
				if !b.PeriodStart.IsZero() {
					periodStart = b.PeriodStart.Format("2006-01-02")
				}
				periodEnd := ""
				if !b.PeriodEnd.IsZero() {
					periodEnd = b.PeriodEnd.Format("2006-01-02")
				}
				isPaid := "false"
				if b.IsPaid {
					isPaid = "true"
				}
				rows = append(rows, []string{
					u.Provider,
					u.Type,
					b.BillNumber,
					fmt.Sprintf("%.2f", b.AmountTotal),
					periodStart,
					periodEnd,
					isPaid,
				})
			}
			// If the utility has no bills, still emit one row with utility info
			if len(u.Bills) == 0 {
				rows = append(rows, []string{u.Provider, u.Type, "", "", "", "", ""})
			}
		}
		log.Printf("✅ Exported %d utilities (CSV, %d bill rows) for user %d", len(utilities), len(rows), userID)
		writeCSV(c, filename, headers, rows)
		return
	}

	filename := fmt.Sprintf("homelog_utilities_%s.json", timestamp)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")

	log.Printf("✅ Exported %d utilities for user %d", len(utilities), userID)

	c.JSON(http.StatusOK, gin.H{
		"exported_at": time.Now(),
		"version":     "1.0.0",
		"type":        "utilities",
		"count":       len(utilities),
		"utilities":   utilities,
	})
}

// ExportProjects exports the user's projects.
// GET /api/v1/export/projects
func (h *ExportHandler) ExportProjects(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var projects []models.Project
	h.db.Where("user_id = ?", userID).Find(&projects)

	timestamp := time.Now().Format("2006-01-02_15-04-05")

	if c.Query("format") == "csv" {
		// Preload expenses for spent calculation
		h.db.Where("user_id = ?", userID).Preload("Expenses").Find(&projects)

		filename := fmt.Sprintf("homelog_projects_%s.csv", timestamp)
		headers := []string{"Name", "Description", "Budget", "Spent", "Status", "Created At"}
		rows := make([][]string, 0, len(projects))
		for _, p := range projects {
			budget := fmt.Sprintf("%.2f", p.Budget)
			var spent float64
			for _, e := range p.Expenses {
				spent += e.Amount
			}
			createdAt := p.CreatedAt.Format("2006-01-02")
			rows = append(rows, []string{p.Name, p.Description, budget, fmt.Sprintf("%.2f", spent), p.Status, createdAt})
		}
		log.Printf("✅ Exported %d projects (CSV) for user %d", len(projects), userID)
		writeCSV(c, filename, headers, rows)
		return
	}

	filename := fmt.Sprintf("homelog_projects_%s.json", timestamp)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")

	log.Printf("✅ Exported %d projects for user %d", len(projects), userID)

	c.JSON(http.StatusOK, gin.H{
		"exported_at": time.Now(),
		"version":     "1.0.0",
		"type":        "projects",
		"count":       len(projects),
		"projects":    projects,
	})
}

// ImportData imports data from a backup JSON file.
// POST /api/v1/import
func (h *ExportHandler) ImportData(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "import_invalid_json", "The file is not valid JSON")
		return
	}

	version, _ := payload["version"].(string)
	if version == "" {
		apierr.Fail(c, http.StatusBadRequest, "import_invalid_version", "The backup is missing a valid 'version' field")
		return
	}

	importType := "full"
	if t, ok := payload["type"].(string); ok && t != "" {
		importType = t
	}

	log.Printf("📥 Starting '%s' import for user %d", importType, userID)

	tx := h.db.Begin()
	if tx.Error != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to start transaction")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("❌ Import panicked: %v", r)
		}
	}()

	counts := map[string]int{}

	switch importType {
	case "expenses":
		if raw, ok := payload["expenses"].([]any); ok {
			n, err := h.importExpenses(tx, userID, raw, map[uint]uint{})
			if err != nil {
				tx.Rollback()
				apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to import expenses: "+err.Error())
				return
			}
			counts["expenses"] = n
		}

	case "utilities":
		if raw, ok := payload["utilities"].([]any); ok {
			n, err := h.importUtilities(tx, userID, raw)
			if err != nil {
				tx.Rollback()
				apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to import utilities: "+err.Error())
				return
			}
			counts["utilities"] = n
		}

	case "projects":
		if raw, ok := payload["projects"].([]any); ok {
			n, err := h.importProjects(tx, userID, raw)
			if err != nil {
				tx.Rollback()
				apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to import projects: "+err.Error())
				return
			}
			counts["projects"] = n
		}

	default: // full backup
		if err := h.importFull(tx, userID, payload, counts); err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Import failed: "+err.Error())
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to commit transaction")
		return
	}

	log.Printf("✅ Import completed for user %d: %+v", userID, counts)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Importazione completata con successo",
		"imported": counts,
	})
}

// importExpenses inserts expense records for the given user.
// BillID is always cleared to avoid dangling FK references.
func (h *ExportHandler) importExpenses(tx *gorm.DB, userID uint, raw []any, deviceIDs map[uint]uint) (int, error) {
	count := 0
	for _, item := range raw {
		b, _ := json.Marshal(item)
		var e models.Expense
		if err := json.Unmarshal(b, &e); err != nil {
			continue
		}

		if e.DeviceID != nil {
			if newDeviceID, ok := deviceIDs[*e.DeviceID]; ok {
				e.DeviceID = &newDeviceID
			} else {
				e.DeviceID = nil
			}
		}

		e.ID = 0          // let DB assign a new ID
		e.UserID = userID // enforce ownership
		e.BillID = nil    // do not carry over bill FK — bills are not re-imported here
		e.Splits = nil    // splits handled separately in full import
		if err := tx.Omit("Splits").Create(&e).Error; err != nil {
			return count, fmt.Errorf("spesa '%s': %w", e.Description, err)
		}
		count++
	}
	return count, nil
}

// importUtilities inserts utility records (without readings/bills).
func (h *ExportHandler) importUtilities(tx *gorm.DB, userID uint, raw []any) (int, error) {
	count := 0
	for _, item := range raw {
		b, _ := json.Marshal(item)
		var u models.Utility
		if err := json.Unmarshal(b, &u); err != nil {
			continue
		}
		u.ID = 0
		u.UserID = userID
		u.Readings = nil
		u.Bills = nil
		if err := tx.Omit("Readings", "Bills").Create(&u).Error; err != nil {
			return count, fmt.Errorf("utenza '%s': %w", u.Provider, err)
		}
		count++
	}
	return count, nil
}

// importProjects inserts project records.
func (h *ExportHandler) importProjects(tx *gorm.DB, userID uint, raw []any) (int, error) {
	count := 0
	for _, item := range raw {
		b, _ := json.Marshal(item)
		var p models.Project
		if err := json.Unmarshal(b, &p); err != nil {
			continue
		}
		p.ID = 0
		p.UserID = userID
		p.SharedWith = nil
		if err := tx.Omit("SharedWith").Create(&p).Error; err != nil {
			return count, fmt.Errorf("progetto '%s': %w", p.Name, err)
		}
		count++
	}
	return count, nil
}

func (h *ExportHandler) importDeviceCategories(tx *gorm.DB, userID uint, raw []any) (map[uint]uint, error) {
	categoryIDs := make(map[uint]uint)

	for _, item := range raw {
		b, _ := json.Marshal(item)

		var category models.DeviceCategory
		if err := json.Unmarshal(b, &category); err != nil {
			continue
		}

		oldID := category.ID

		if category.UserID == nil || category.IsDefault {
			var existing models.DeviceCategory

			if err := tx.
				Where("slug = ? AND user_id IS NULL", category.Slug).
				First(&existing).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return categoryIDs, err
				}

				category.ID = 0
				category.UserID = nil
				category.IsDefault = true

				if err := tx.Create(&category).Error; err != nil {
					return categoryIDs, fmt.Errorf("device category '%s': %w", category.Name, err)
				}

				existing = category
			}

			categoryIDs[oldID] = existing.ID
			continue
		}

		var existing models.DeviceCategory

		if err := tx.
			Where("user_id = ? AND name = ?", userID, category.Name).
			First(&existing).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return categoryIDs, err
			}

			category.ID = 0
			category.UserID = &userID
			category.IsDefault = false

			if err := tx.Create(&category).Error; err != nil {
				return categoryIDs, fmt.Errorf("device category '%s': %w", category.Name, err)
			}

			existing = category
		}

		categoryIDs[oldID] = existing.ID
	}

	return categoryIDs, nil
}

func (h *ExportHandler) importDevices(tx *gorm.DB, userID uint, raw []any, categoryIDs map[uint]uint) (int, map[uint]uint, error) {
	count := 0
	deviceIDs := make(map[uint]uint)

	for _, item := range raw {
		b, _ := json.Marshal(item)

		var device models.Device
		if err := json.Unmarshal(b, &device); err != nil {
			continue
		}
		if !isPropertyMember(tx, userID, device.PropertyID) {
			return count, deviceIDs, fmt.Errorf(
				"device '%s': property %d is not accessible",
				device.Name,
				device.PropertyID,
			)
		}

		oldDeviceID := device.ID

		device.ID = 0
		device.UserID = userID

		if device.CategoryID != nil {
			if newCategoryID, ok := categoryIDs[*device.CategoryID]; ok {
				device.CategoryID = &newCategoryID
			} else {
				device.CategoryID = nil
			}
		}

		device.Property = models.Property{}
		device.Category = nil
		device.Expenses = nil

		if err := tx.Omit("Property", "Category", "Expenses").Create(&device).Error; err != nil {
			return count, deviceIDs, fmt.Errorf("device '%s': %w", device.Name, err)
		}
		deviceIDs[oldDeviceID] = device.ID

		count++
	}

	return count, deviceIDs, nil
}

// importFull handles a complete backup, including devices and device categories.
// Importing is additive: no existing data is overwritten.
func (h *ExportHandler) importFull(tx *gorm.DB, userID uint, payload map[string]any, counts map[string]int) error {
	var deviceCategoryIDs map[uint]uint
	var deviceIDs map[uint]uint

	if raw, ok := payload["device_categories"].([]any); ok {
		var err error
		deviceCategoryIDs, err = h.importDeviceCategories(tx, userID, raw)
		if err != nil {
			return err
		}
		counts["device_categories"] = len(deviceCategoryIDs)
	}

	if raw, ok := payload["devices"].([]any); ok {
		n, ids, err := h.importDevices(tx, userID, raw, deviceCategoryIDs)
		if err != nil {
			return err
		}
		deviceIDs = ids
		counts["devices"] = n
	}

	if raw, ok := payload["expenses"].([]any); ok {
		n, err := h.importExpenses(tx, userID, raw, deviceIDs)
		if err != nil {
			return err
		}
		counts["expenses"] = n
	}
	if raw, ok := payload["utilities"].([]any); ok {
		n, err := h.importUtilities(tx, userID, raw)
		if err != nil {
			return err
		}
		counts["utilities"] = n
	}
	if raw, ok := payload["projects"].([]any); ok {
		n, err := h.importProjects(tx, userID, raw)
		if err != nil {
			return err
		}
		counts["projects"] = n
	}

	return nil
}
