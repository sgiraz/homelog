package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a system user
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"not null" json:"-"`
	Name         string `gorm:"not null" json:"name"`
	Role         string `gorm:"not null;default:'user'" json:"role"` // admin, user
	IsActive     bool   `gorm:"not null;default:true" json:"is_active"`

	PasswordResetToken   string     `gorm:"index" json:"-"`
	PasswordResetExpires *time.Time `json:"-"`
}

// HouseholdMember represents a member of a household (can be a registered user or virtual)
type HouseholdMember struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	PropertyID uint   `gorm:"not null;index" json:"property_id"`
	UserID     *uint  `gorm:"index" json:"user_id,omitempty"` // NULL if virtual member
	Name       string `gorm:"not null" json:"name"`
	Role       string `json:"role,omitempty"` // partner, figlio, coinquilino, etc.
	IsVirtual  bool   `gorm:"not null;default:false" json:"is_virtual"`

	// Relations
	Property Property `json:"property,omitempty"`
	User     *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Property represents a house/apartment
type Property struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID    uint       `gorm:"not null;index" json:"user_id"`
	Name      string     `gorm:"not null" json:"name"`
	Address   string     `json:"address"`
	Type      string     `json:"type"` // owned, rented
	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	IsCurrent bool       `gorm:"not null;default:false" json:"is_current"`
	Residents int        `gorm:"default:1" json:"residents"`
}

// Category represents an expense category
type Category struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID    *uint  `gorm:"index" json:"user_id,omitempty"`
	User      *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Name      string `gorm:"not null" json:"name"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	IsDefault bool   `gorm:"not null;default:false" json:"is_default"`

	Subcategories []Subcategory `gorm:"foreignKey:CategoryID" json:"subcategories,omitempty"`
}

// Subcategory represents an expense subcategory
type Subcategory struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	CategoryID uint   `gorm:"not null;index" json:"category_id"`
	Name       string `gorm:"not null" json:"name"`
}

// Expense represents a single expense
type Expense struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID        uint  `gorm:"not null;index" json:"user_id"`
	PropertyID    *uint `gorm:"index" json:"property_id,omitempty"`
	CategoryID    uint  `gorm:"not null;index" json:"category_id"`
	SubcategoryID *uint `gorm:"index" json:"subcategory_id,omitempty"`
	ProjectID     *uint `gorm:"index" json:"project_id,omitempty"`
	BillID        *uint `gorm:"index" json:"bill_id,omitempty"` // auto-created from bill payment

	Amount        float64   `gorm:"not null" json:"amount"`
	Date          time.Time `gorm:"not null;index" json:"date"`
	Description   string    `json:"description"`
	AttachmentURL string    `json:"attachment_url,omitempty"`

	// Split fields
	PaidByMemberID uint `gorm:"not null;index;default:0" json:"paid_by_member_id"` // HouseholdMember ID
	IsSplit        bool `gorm:"not null;default:false" json:"is_split"`

	// Relations
	Property    *Property        `json:"property,omitempty"`
	Category    Category         `json:"category"`
	Subcategory *Subcategory     `json:"subcategory,omitempty"`
	Project     *Project         `json:"project,omitempty"`
	PaidBy      *HouseholdMember `gorm:"foreignKey:PaidByMemberID" json:"paid_by,omitempty"`
	Splits      []ExpenseSplit   `gorm:"foreignKey:ExpenseID" json:"splits,omitempty"`
}

// Utility represents a utility service (electricity, gas, water, waste)
type Utility struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID       uint       `gorm:"not null;index" json:"user_id"`
	PropertyID   uint       `gorm:"not null;index" json:"property_id"`
	Type         string     `gorm:"not null" json:"type"` // electricity, gas, water, waste
	Provider     string     `gorm:"not null" json:"provider"`
	CustomerCode string     `json:"customer_code"`
	ServiceCode  string     `json:"service_code"` // POD, PDR, etc.
	Address      string     `json:"address"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	IsActive     bool       `gorm:"not null;default:true" json:"is_active"`

	// Additional fields
	PowerCapacity       float64 `json:"power_capacity,omitempty"` // For electricity (kW)
	CustomerPortal      string  `json:"customer_portal,omitempty"`
	Notes               string  `json:"notes,omitempty"`
	AllowsSelfReading   *bool   `gorm:"default:true" json:"allows_self_reading"`  // Se il fornitore accetta autolettura
	ComparisonThreshold float64 `gorm:"default:2.0" json:"comparison_threshold"` // Soglia base per letture stesso giorno (kWh/mc)
	ThresholdPerDay     float64 `gorm:"default:1.0" json:"threshold_per_day"`    // Tolleranza aggiuntiva per ogni giorno di differenza

	// Relations
	Property Property       `json:"property"`
	Readings []MeterReading `gorm:"foreignKey:UtilityID" json:"readings,omitempty"`
	Bills    []Bill         `gorm:"foreignKey:UtilityID" json:"bills,omitempty"`
	Rates    []UtilityRate  `gorm:"foreignKey:UtilityID" json:"rates,omitempty"`
}

// MeterReading represents a USER's manual meter reading (autolettura)
// This is used to compare against provider readings in bills to detect anomalies
type MeterReading struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UtilityID   uint      `gorm:"not null;index" json:"utility_id"`
	ReadingDate time.Time `gorm:"not null;index" json:"reading_date"`

	// For electricity (multi-band meters)
	ValueF1 *float64 `json:"value_f1,omitempty"` // Fascia F1 (peak hours)
	ValueF2 *float64 `json:"value_f2,omitempty"` // Fascia F2 (mid hours)
	ValueF3 *float64 `json:"value_f3,omitempty"` // Fascia F3 (off-peak hours)

	// For gas/water (single value meters)
	Value *float64 `json:"value,omitempty"` // Lettura singola (mc per acqua, Smc per gas)

	// Source of reading
	Source   string `gorm:"default:'manual'" json:"source"` // manual, submitted (inviata al fornitore)
	PhotoURL string `json:"photo_url,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

// Bill represents a utility bill
type Bill struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UtilityID   uint      `gorm:"not null;index" json:"utility_id"`
	BillNumber  string    `gorm:"not null;index" json:"bill_number"`

	// Explicit association to the user's self-reading for this bill period.
	// When set, this reading is used as the "Consumo Effettivo" anchor.
	// When nil, the provider_reading value is used as fallback.
	UserReadingID *uint        `json:"user_reading_id,omitempty"`
	UserReading   *MeterReading `gorm:"foreignKey:UserReadingID" json:"user_reading,omitempty"`
	IssueDate   time.Time `gorm:"not null" json:"issue_date"`
	PeriodStart time.Time `gorm:"not null;index" json:"period_start"`
	PeriodEnd   time.Time `gorm:"not null;index" json:"period_end"`
	DueDate     time.Time `gorm:"not null;index" json:"due_date"`

	// Provider Readings (letture rilevate dal fornitore nella bolletta)
	// These are the readings reported by the utility provider
	ProviderReadingDate *time.Time `json:"provider_reading_date,omitempty"` // Data lettura fornitore
	ProviderReadingF1   *float64   `json:"provider_reading_f1,omitempty"`   // Lettura F1 (electricity peak)
	ProviderReadingF2   *float64   `json:"provider_reading_f2,omitempty"`   // Lettura F2 (electricity mid)
	ProviderReadingF3   *float64   `json:"provider_reading_f3,omitempty"`   // Lettura F3 (electricity off-peak)
	ProviderReading     *float64   `json:"provider_reading,omitempty"`      // Lettura singola (gas/water)
	ReadingType         string     `json:"reading_type,omitempty"`          // actual (rilevata), estimated (stimata)

	// Legacy reading fields (for backwards compatibility)
	ReadingStartDate  *time.Time `json:"reading_start_date,omitempty"`
	ReadingStartValue *float64   `json:"reading_start_value,omitempty"`
	ReadingEndDate    *time.Time `json:"reading_end_date,omitempty"`
	ReadingEndValue   *float64   `json:"reading_end_value,omitempty"`

	// Consumption (from bill)
	ConsumptionTotal float64  `gorm:"not null" json:"consumption_total"`
	ConsumptionF1    *float64 `json:"consumption_f1,omitempty"`
	ConsumptionF2    *float64 `json:"consumption_f2,omitempty"`
	ConsumptionF3    *float64 `json:"consumption_f3,omitempty"`

	// Gas conversion coefficient (Coefficiente di conversione C)
	// True billed consumption = Smc × ConversionCoefficient
	ConversionCoefficient *float64 `json:"conversion_coefficient,omitempty"`

	// Estimated reading/consumption (quando la bolletta contiene una stima)
	EstimatedDate        *time.Time `json:"estimated_date,omitempty"`
	EstimatedReading     *float64   `json:"estimated_reading,omitempty"`     // Lettura stimata (mc)
	EstimatedConsumption *float64   `json:"estimated_consumption,omitempty"` // Consumo stimato (Smc), calcolato

	// Amounts
	AmountTotal  float64  `gorm:"not null" json:"amount_total"`
	AmountEnergy *float64 `json:"amount_energy,omitempty"`
	AmountFixed  *float64 `json:"amount_fixed,omitempty"`
	AmountTaxes  *float64 `json:"amount_taxes,omitempty"`
	AmountVAT    *float64 `json:"amount_vat,omitempty"`

	// Payment
	IsPaid   bool       `gorm:"not null;default:false" json:"is_paid"`
	PaidDate *time.Time `json:"paid_date,omitempty"`

	// Attachments
	PDFURL string `json:"pdf_url,omitempty"`

	// Flexible data for custom fields (JSON)
	ParsedData string `gorm:"type:text" json:"parsed_data,omitempty"` // JSONB-like

	// Relations
	Utility Utility `json:"utility"`
}

// UtilityRate represents historical rates for a utility
type UtilityRate struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UtilityID     uint      `gorm:"not null;index" json:"utility_id"`
	EffectiveDate time.Time `gorm:"not null;index" json:"effective_date"`

	// Rates
	RateF1    *float64 `json:"rate_f1,omitempty"`
	RateF2    *float64 `json:"rate_f2,omitempty"`
	RateF3    *float64 `json:"rate_f3,omitempty"`
	RateFixed *float64 `json:"rate_fixed,omitempty"` // Monthly fixed cost
	RateUnit  *float64 `json:"rate_unit,omitempty"`  // For gas/water (per Smc/mc)

	Notes string `json:"notes,omitempty"`
}

// Project represents a spending project (renovation, trip, etc.)
type Project struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID     uint  `gorm:"not null;index" json:"user_id"`
	PropertyID *uint `gorm:"index" json:"property_id,omitempty"`

	Name        string    `gorm:"not null" json:"name"`
	Icon        string    `json:"icon"`
	Budget      float64   `gorm:"not null" json:"budget"`
	StartDate   time.Time `gorm:"not null" json:"start_date"`
	EndDate     time.Time `gorm:"not null" json:"end_date"`
	Description string    `json:"description,omitempty"`
	Status      string    `gorm:"not null;default:'active'" json:"status"` // active, completed, cancelled

	// Relations
	Property   *Property `json:"property,omitempty"`
	Expenses   []Expense `gorm:"foreignKey:ProjectID" json:"expenses,omitempty"`
	SharedWith []User    `gorm:"many2many:project_members;" json:"shared_with,omitempty"`
}

// UserSettings represents user preferences
type UserSettings struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID            uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	DefaultPropertyID *uint  `json:"default_property_id,omitempty"`
	Language          string `gorm:"not null;default:'it'" json:"language"`
	Currency          string `gorm:"not null;default:'EUR'" json:"currency"`
	Theme             string `gorm:"not null;default:'auto'" json:"theme"` // auto, light, dark
	DateFormat        string `gorm:"not null;default:'DD/MM/YYYY'" json:"date_format"`

	// Split preferences
	DefaultSplitWithMemberIDs string `json:"default_split_with_member_ids,omitempty"` // JSON array as string e.g. "[2,3]"

	// Template preferences
	DefaultTemplates string `json:"default_templates,omitempty"` // JSON object as string e.g. {"electricity": 1, "gas": 2}

	// Notifications
	EmailNotifications  bool `gorm:"not null;default:true" json:"email_notifications"`
	InAppNotifications  bool `gorm:"not null;default:true" json:"in_app_notifications"`
	BillDueAlertDays    int  `gorm:"not null;default:3" json:"bill_due_alert_days"`
	ReadingReminderDays int  `gorm:"not null;default:7" json:"reading_reminder_days"`

	// Thresholds
	AnomalyThreshold float64 `gorm:"not null;default:5.0" json:"anomaly_threshold"` // Percentage
}

// HouseholdSettings represents split settings for a property
type HouseholdSettings struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	PropertyID       uint   `gorm:"uniqueIndex;not null" json:"property_id"`
	SplitMode        bool   `gorm:"not null;default:false" json:"split_mode"`
	DefaultSplitType string `gorm:"not null;default:'equal'" json:"default_split_type"` // equal, custom, income_based

	// Relations
	Property Property `json:"property"`
}

// ExpenseSplit represents a member's share of an expense
type ExpenseSplit struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ExpenseID    uint       `gorm:"not null;index" json:"expense_id"`
	MemberID     uint       `gorm:"not null;index" json:"member_id"` // HouseholdMember ID
	Amount       float64    `gorm:"not null" json:"amount"`
	IsSettled    bool       `gorm:"not null;default:false" json:"is_settled"`
	SettledAt    *time.Time `json:"settled_at,omitempty"`
	SettlementID *uint      `gorm:"index" json:"settlement_id,omitempty"`

	// Relations
	Expense    Expense         `json:"expense"`
	Member     HouseholdMember `gorm:"foreignKey:MemberID" json:"member"`
	Settlement *Settlement     `json:"settlement,omitempty"`
}

// Settlement represents a payment between members to settle debts
type Settlement struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	PropertyID    uint      `gorm:"not null;index" json:"property_id"`
	FromMemberID  uint      `gorm:"not null;index" json:"from_member_id"` // HouseholdMember ID
	ToMemberID    uint      `gorm:"not null;index" json:"to_member_id"`   // HouseholdMember ID
	Amount        float64   `gorm:"not null" json:"amount"`
	Date          time.Time `gorm:"not null;index" json:"date"`
	PaymentMethod string    `json:"payment_method,omitempty"` // bank_transfer, cash, satispay, paypal
	Note          string    `json:"note,omitempty"`

	// Relations
	Property      Property        `json:"property"`
	FromMember    HouseholdMember `gorm:"foreignKey:FromMemberID" json:"from_member"`
	ToMember      HouseholdMember `gorm:"foreignKey:ToMemberID" json:"to_member"`
	ExpenseSplits []ExpenseSplit  `gorm:"foreignKey:SettlementID" json:"expense_splits,omitempty"`
}

// BillTemplate represents extraction rules for a utility provider's bill format
type BillTemplate struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID          uint   `gorm:"not null;index" json:"user_id"`
	Name            string `gorm:"not null" json:"name"`                     // Template display name
	Provider        string `gorm:"not null;index" json:"provider"`           // Provider name (user-defined)
	UtilityType     string `gorm:"not null" json:"utility_type"`             // electricity, gas, water, waste
	IsDefault       bool   `gorm:"not null;default:false" json:"is_default"` // Default template for this provider+type
	ExtractionRules string `gorm:"type:text" json:"extraction_rules"`        // JSON with regex patterns for each field

	// Relations
	User User `json:"user,omitempty"`
}

// ContractTemplate represents extraction rules for utility contracts
type ContractTemplate struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID          uint   `gorm:"not null;index" json:"user_id"`
	Name            string `gorm:"not null" json:"name"`
	Provider        string `gorm:"not null;index" json:"provider"`
	UtilityType     string `gorm:"not null" json:"utility_type"`
	IsDefault       bool   `gorm:"not null;default:false" json:"is_default"`
	ExtractionRules string `gorm:"type:text" json:"extraction_rules"` // JSON with patterns for provider, POD/PDR, customer_code, address

	// Relations
	User User `json:"user,omitempty"`
}
