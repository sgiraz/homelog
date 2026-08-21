package database

import (
	"log"
	"os"
	"time"

	"github.com/sgiraz/homelog/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Demo account credentials. The demo environment exposes a single shared
// account; the password is intentionally trivial because the whole dataset is
// wiped on a schedule (see ResetDemoData). NEVER enable DEMO_MODE in a real
// deployment: it seeds this shared account with a published password and
// erases every row on each reset.
const (
	DemoEmail    = "demo@homelog.app"
	DemoPassword = "demo"
)

// IsDemoMode reports whether the server runs as a public demo instance.
// Controlled by the DEMO_MODE env var ("true" enables it).
func IsDemoMode() bool {
	return os.Getenv("DEMO_MODE") == "true"
}

// demoTables lists every application table cleared during a reset, ordered so
// that children are deleted before their parents while foreign keys are off.
// search_index (FTS5) is included because raw DELETEs bypass the GORM hooks
// that normally keep it in sync — re-seeding via GORM .Create() repopulates it.
var demoTables = []string{
	"search_index",
	"expense_splits",
	"settlements",
	"bill_installments",
	"bills",
	"meter_readings",
	"price_changes",
	"service_communications",
	"utility_rates",
	"utilities",
	"expenses",
	"expense_templates",
	"bill_templates",
	"contract_templates",
	"project_members",
	"projects",
	"notifications",
	"property_join_requests",
	"household_members",
	"household_settings",
	"user_settings",
	"subcategories",
	"categories",
	"properties",
	"users",
}

// ResetDemoData wipes every row and rebuilds the demo dataset from scratch.
// It mirrors AutoMigrate's approach of toggling foreign keys off around a bulk
// mutation rather than wrapping in a transaction, because SQLite cannot change
// the foreign_keys pragma inside an open transaction. The window of
// inconsistency is sub-100ms for the small demo dataset and acceptable for a
// public demo. Deleting all rows resets each table's implicit rowid counter, so
// the demo user is recreated with ID 1 and existing demo JWTs stay valid.
func ResetDemoData(db *gorm.DB) error {
	log.Println("🎭 Resetting demo data...")

	db.Exec("PRAGMA foreign_keys=OFF;")
	defer db.Exec("PRAGMA foreign_keys=ON;")

	for _, t := range demoTables {
		if err := db.Exec("DELETE FROM " + t).Error; err != nil {
			// search_index may not exist on a brand-new DB before initSearchIndex;
			// log and continue rather than aborting the whole reset.
			log.Printf("⚠️  demo reset: failed to clear %s: %v", t, err)
		}
	}

	// GORM declares integer primary keys as AUTOINCREMENT in SQLite, so the
	// per-table counters live in sqlite_sequence and survive DELETE. Clearing
	// them restarts IDs at 1 — crucial so the demo user is recreated as ID 1
	// and previously issued demo JWTs (which embed user_id=1) keep working.
	db.Exec("DELETE FROM sqlite_sequence")

	if err := seedDemoData(db); err != nil {
		return err
	}

	log.Println("✅ Demo data reset complete")
	return nil
}

// SeedDemoIfNeeded seeds the demo dataset at startup when DEMO_MODE is on and
// the demo account is missing (fresh container). Existing data is left intact
// so a restart mid-hour doesn't wipe a visitor's session.
func SeedDemoIfNeeded(db *gorm.DB) error {
	if !IsDemoMode() {
		return nil
	}
	var count int64
	db.Model(&models.User{}).Where("email = ?", DemoEmail).Count(&count)
	if count > 0 {
		return nil
	}
	return ResetDemoData(db)
}

// seedDemoData builds a realistic Italian household: one property, three
// members, a mix of metered and fixed utilities with bills/readings, a spread
// of expenses (some split) across the last few months, a settlement, and two
// projects. It is written against the GORM models so it can never drift from
// the schema. Created via .Create() so the FTS AfterSave hooks repopulate
// search_index automatically.
func seedDemoData(db *gorm.DB) error {
	now := time.Now()
	daysAgo := func(d int) time.Time { return now.AddDate(0, 0, -d) }
	f := func(v float64) *float64 { return &v }

	// Global categories (no user_id) — expenses reference these by ID.
	if err := SeedDefaultCategories(db); err != nil {
		return err
	}
	catID := func(name string) uint {
		var c models.Category
		db.Where("name = ? AND user_id IS NULL", name).First(&c)
		return c.ID
	}

	// User (first insert → ID 1).
	hash, _ := bcrypt.GenerateFromPassword([]byte(DemoPassword), bcrypt.DefaultCost)
	user := models.User{
		Email:        DemoEmail,
		PasswordHash: string(hash),
		Name:         "Mario Rossi",
		Role:         "admin",
		IsActive:     true,
	}
	if err := db.Create(&user).Error; err != nil {
		return err
	}

	settings := models.UserSettings{
		UserID:              user.ID,
		Language:            "it",
		Currency:            "EUR",
		Theme:               "auto",
		DateFormat:          "DD/MM/YYYY",
		EmailNotifications:  true,
		InAppNotifications:  true,
		BillDueAlertDays:    3,
		ReadingReminderDays: 7,
		AnomalyThreshold:    5.0,
		OnboardingCompleted: true,
	}
	if err := db.Create(&settings).Error; err != nil {
		return err
	}

	property := models.Property{
		UserID:    user.ID,
		Name:      "Casa Milano",
		Address:   "Via Roma 12, Milano",
		Type:      "owned",
		StartDate: now.AddDate(-3, 0, 0),
		IsCurrent: true,
		Residents: 3,
	}
	if err := db.Create(&property).Error; err != nil {
		return err
	}

	if err := db.Create(&models.HouseholdSettings{
		PropertyID:       property.ID,
		SplitMode:        true,
		DefaultSplitType: "equal",
	}).Error; err != nil {
		return err
	}

	// Members: Mario (the account), Giulia (partner), Luca (figlio).
	mario := models.HouseholdMember{PropertyID: property.ID, UserID: &user.ID, Name: "Mario", Role: "admin", IsVirtual: false}
	giulia := models.HouseholdMember{PropertyID: property.ID, Name: "Giulia", Role: "partner", IsVirtual: true}
	luca := models.HouseholdMember{PropertyID: property.ID, Name: "Luca", Role: "figlio", IsVirtual: true}
	for _, m := range []*models.HouseholdMember{&mario, &giulia, &luca} {
		if err := db.Create(m).Error; err != nil {
			return err
		}
	}

	// ---- Utilities -------------------------------------------------------
	elec := models.Utility{
		UserID: user.ID, PropertyID: property.ID, Type: "electricity", Provider: "Enel Energia",
		CustomerCode: "IT001E12345678", ServiceCode: "IT001E12345678", Address: property.Address,
		StartDate: now.AddDate(-3, 0, 0), IsActive: true, IsMetered: true, PowerCapacity: 3.0,
		DefaultCategoryID: f0(catID("Casa")), PaidByMemberID: &mario.ID,
		BillingInterval: 2, BillingUnit: "month",
	}
	gas := models.Utility{
		UserID: user.ID, PropertyID: property.ID, Type: "gas", Provider: "Eni Plenitude",
		CustomerCode: "123456789", ServiceCode: "00123456789012", Address: property.Address,
		StartDate: now.AddDate(-3, 0, 0), IsActive: true, IsMetered: true,
		DefaultCategoryID: f0(catID("Casa")), PaidByMemberID: &mario.ID,
		BillingInterval: 2, BillingUnit: "month", IsInstallmentBased: true,
	}
	water := models.Utility{
		UserID: user.ID, PropertyID: property.ID, Type: "water", Provider: "MM Servizi Idrici",
		CustomerCode: "W-998877", Address: property.Address,
		StartDate: now.AddDate(-3, 0, 0), IsActive: true, IsMetered: true, IsDomiciled: true,
		DefaultCategoryID: f0(catID("Casa")), PaidByMemberID: &mario.ID,
		BillingInterval: 3, BillingUnit: "month",
	}
	internet := models.Utility{
		UserID: user.ID, PropertyID: property.ID, Type: "internet", Provider: "Fastweb",
		CustomerCode: "FW-456123", Address: property.Address,
		StartDate: now.AddDate(-2, 0, 0), IsActive: true, IsMetered: false,
		RecurringAmount: f(29.95), BillingInterval: 1, BillingUnit: "month",
		DefaultCategoryID: f0(catID("Tecnologia")), PaidByMemberID: &mario.ID, IsDomiciled: true,
	}
	for _, u := range []*models.Utility{&elec, &gas, &water, &internet} {
		if err := db.Create(u).Error; err != nil {
			return err
		}
	}

	// ---- Meter readings (autoletture) -----------------------------------
	readings := []models.MeterReading{
		{UtilityID: elec.ID, ReadingDate: daysAgo(120), ValueF1: f(1840), ValueF2: f(1210), ValueF3: f(980), Source: "manual"},
		{UtilityID: elec.ID, ReadingDate: daysAgo(60), ValueF1: f(1995), ValueF2: f(1300), ValueF3: f(1055), Source: "manual"},
		{UtilityID: elec.ID, ReadingDate: daysAgo(5), ValueF1: f(2140), ValueF2: f(1388), ValueF3: f(1120), Source: "manual"},
		{UtilityID: gas.ID, ReadingDate: daysAgo(120), Value: f(3420), Source: "manual"},
		{UtilityID: gas.ID, ReadingDate: daysAgo(60), Value: f(3610), Source: "submitted"},
		{UtilityID: gas.ID, ReadingDate: daysAgo(5), Value: f(3755), Source: "manual"},
		{UtilityID: water.ID, ReadingDate: daysAgo(95), Value: f(412), Source: "manual"},
		{UtilityID: water.ID, ReadingDate: daysAgo(5), Value: f(448), Source: "manual"},
	}
	for i := range readings {
		if err := db.Create(&readings[i]).Error; err != nil {
			return err
		}
	}

	// ---- Bills (with one installment each, mirroring the app's invariant) -
	// Helper to create a bill plus a single installment matching it.
	makeBill := func(b models.Bill) (models.Bill, error) {
		if err := db.Create(&b).Error; err != nil {
			return b, err
		}
		inst := models.BillInstallment{
			BillID: b.ID, Number: 1, DueDate: b.DueDate, Amount: b.AmountTotal,
			IsPaid: b.IsPaid, PaidAt: b.PaidDate,
		}
		return b, db.Create(&inst).Error
	}

	bills := []models.Bill{
		{UtilityID: elec.ID, BillNumber: "EE-2025-0012", IssueDate: daysAgo(115), PeriodStart: daysAgo(180), PeriodEnd: daysAgo(120),
			DueDate: daysAgo(95), ProviderReading: f(4030), ConsumptionTotal: 310, AmountTotal: 92.40, IsPaid: true, PaidDate: tptr(daysAgo(96))},
		{UtilityID: elec.ID, BillNumber: "EE-2025-0048", IssueDate: daysAgo(55), PeriodStart: daysAgo(120), PeriodEnd: daysAgo(60),
			DueDate: daysAgo(35), ProviderReading: f(4350), ConsumptionTotal: 320, AmountTotal: 98.10, IsPaid: true, PaidDate: tptr(daysAgo(36))},
		{UtilityID: gas.ID, BillNumber: "GAS-771-2025", IssueDate: daysAgo(50), PeriodStart: daysAgo(120), PeriodEnd: daysAgo(60),
			DueDate: daysAgo(30), ProviderReading: f(3610), ConsumptionTotal: 190, ConversionCoefficient: f(1.04), AmountTotal: 145.80, IsPaid: false},
		{UtilityID: water.ID, BillNumber: "ACQ-2025-334", IssueDate: daysAgo(20), PeriodStart: daysAgo(95), PeriodEnd: daysAgo(5),
			DueDate: daysAgo(-5), ProviderReading: f(448), ConsumptionTotal: 36, AmountTotal: 58.20, IsPaid: false},
	}
	for _, b := range bills {
		if _, err := makeBill(b); err != nil {
			return err
		}
	}

	// A bill-derived communication (price change announcement).
	if err := db.Create(&models.ServiceCommunication{
		UtilityID: elec.ID, Type: "price_change", Title: "Aggiornamento condizioni economiche",
		Content:        "A partire dal prossimo periodo la componente energia sarà aggiornata secondo le nuove condizioni di mercato.",
		ActionDeadline: tptr(now.AddDate(0, 1, 0)), IsImportant: true, IsRead: false,
	}).Error; err != nil {
		return err
	}

	// ---- Projects --------------------------------------------------------
	bagno := models.Project{
		UserID: user.ID, PropertyID: &property.ID, Name: "Ristrutturazione Bagno", Icon: "🛁",
		Budget: 6000, StartDate: daysAgo(90), EndDate: now.AddDate(0, 2, 0), Status: "active",
		Description: "Rifacimento completo del bagno padronale.",
	}
	vacanza := models.Project{
		UserID: user.ID, PropertyID: &property.ID, Name: "Vacanza Estate", Icon: "🏖️",
		Budget: 2500, StartDate: daysAgo(40), EndDate: now.AddDate(0, 3, 0), Status: "active",
		Description: "Settimana al mare in agosto.",
	}
	for _, p := range []*models.Project{&bagno, &vacanza} {
		if err := db.Create(p).Error; err != nil {
			return err
		}
	}

	// ---- Expenses --------------------------------------------------------
	// addExpense creates an expense and, when split, its splits. The payer's
	// own split is always settled (see expense.go gotcha) — the payer owes
	// nothing to themselves.
	addExpense := func(e models.Expense, splitAmong []uint) error {
		if err := db.Create(&e).Error; err != nil {
			return err
		}
		if !e.IsSplit || len(splitAmong) == 0 {
			return nil
		}
		share := round2(e.Amount / float64(len(splitAmong)))
		for _, mid := range splitAmong {
			split := models.ExpenseSplit{ExpenseID: e.ID, MemberID: mid, Amount: share}
			if mid == e.PaidByMemberID {
				split.SettledAmount = share
				split.IsSettled = true
				split.SettledAt = tptr(e.Date)
			}
			if err := db.Create(&split).Error; err != nil {
				return err
			}
		}
		return nil
	}

	all := []uint{mario.ID, giulia.ID, luca.ID}
	couple := []uint{mario.ID, giulia.ID}

	expenses := []struct {
		e     models.Expense
		among []uint
	}{
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Alimentari e Ristorazione"), Amount: 84.30, Date: daysAgo(3), Description: "Spesa settimanale", PaidByMemberID: mario.ID, IsSplit: true}, all},
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Alimentari e Ristorazione"), Amount: 42.00, Date: daysAgo(10), Description: "Cena fuori", PaidByMemberID: giulia.ID, IsSplit: true}, couple},
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Trasporti"), Amount: 60.00, Date: daysAgo(7), Description: "Carburante", PaidByMemberID: mario.ID, IsSplit: false}, nil},
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Intrattenimento"), Amount: 13.99, Date: daysAgo(15), Description: "Abbonamento streaming", PaidByMemberID: mario.ID, IsSplit: true}, all},
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Famiglia"), Amount: 120.00, Date: daysAgo(22), Description: "Materiale scolastico Luca", PaidByMemberID: giulia.ID, IsSplit: false}, nil},
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Casa"), Amount: 220.00, Date: daysAgo(35), Description: "Idraulico", PaidByMemberID: mario.ID, IsSplit: true}, couple},
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Alimentari e Ristorazione"), Amount: 79.10, Date: daysAgo(45), Description: "Spesa supermercato", PaidByMemberID: mario.ID, IsSplit: true}, all},
		// Project-linked expenses
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Casa"), ProjectID: &bagno.ID, Amount: 1850.00, Date: daysAgo(60), Description: "Piastrelle e sanitari", PaidByMemberID: mario.ID, IsSplit: true}, couple},
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Casa"), ProjectID: &bagno.ID, Amount: 900.00, Date: daysAgo(20), Description: "Acconto manodopera", PaidByMemberID: mario.ID, IsSplit: false}, nil},
		{models.Expense{UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Intrattenimento"), ProjectID: &vacanza.ID, Amount: 450.00, Date: daysAgo(12), Description: "Caparra hotel", PaidByMemberID: giulia.ID, IsSplit: true}, all},
	}
	for _, x := range expenses {
		if err := addExpense(x.e, x.among); err != nil {
			return err
		}
	}

	// ---- A settlement (Giulia repays Mario part of a shared expense) -----
	if err := db.Create(&models.Settlement{
		PropertyID: property.ID, FromMemberID: giulia.ID, ToMemberID: mario.ID,
		Amount: 28.10, Date: daysAgo(2), PaymentMethod: "satispay", Note: "Quota spesa settimanale",
	}).Error; err != nil {
		return err
	}

	// ---- A long-term debt, partly repaid ---------------------------------
	// Giulia fronted the deposit on the new house. Left in the running balance
	// it would swallow every ordinary expense between the two of them, so it
	// lives in its own ledger (Expense.IsLongTermDebt) and Mario pays it back
	// at his own pace — the Debts tab shows the remainder and the progress.
	deposit := models.Expense{
		UserID: user.ID, PropertyID: &property.ID, CategoryID: catID("Casa"),
		Amount: 12000, Date: daysAgo(75), Description: "Acconto acquisto casa",
		PaidByMemberID: giulia.ID, IsSplit: true, IsLongTermDebt: true,
	}
	if err := db.Create(&deposit).Error; err != nil {
		return err
	}
	depositShare := round2(deposit.Amount / 2)
	debtSplit := models.ExpenseSplit{ExpenseID: deposit.ID, MemberID: mario.ID, Amount: depositShare}
	payerSplit := models.ExpenseSplit{
		ExpenseID: deposit.ID, MemberID: giulia.ID, Amount: depositShare,
		SettledAmount: depositShare, IsSettled: true, SettledAt: tptr(deposit.Date),
	}
	for _, sp := range []*models.ExpenseSplit{&debtSplit, &payerSplit} {
		if err := db.Create(sp).Error; err != nil {
			return err
		}
	}

	// Two repayments of different sizes, so the progress bar is not at 0.
	for _, r := range []struct {
		amount float64
		day    int
		method string
	}{{800, 50, "bonifico"}, {500, 20, "bonifico"}} {
		repayment := models.Settlement{
			PropertyID: property.ID, FromMemberID: mario.ID, ToMemberID: giulia.ID,
			Amount: r.amount, Date: daysAgo(r.day), PaymentMethod: r.method,
			Note: "Rata acconto casa", TargetExpenseID: &deposit.ID,
		}
		if err := db.Create(&repayment).Error; err != nil {
			return err
		}
		if err := db.Create(&models.SettlementAllocation{
			SettlementID: repayment.ID, ExpenseSplitID: debtSplit.ID,
			Amount: r.amount, Kind: "payment", CreatedAt: repayment.Date,
		}).Error; err != nil {
			return err
		}
		debtSplit.SettledAmount += r.amount
	}
	if err := db.Model(&debtSplit).Update("settled_amount", debtSplit.SettledAmount).Error; err != nil {
		return err
	}

	log.Printf("✅ Demo dataset seeded (user ID=%d)", user.ID)
	return nil
}

// f0 returns a *uint for a non-zero id, or nil when the id is 0 (category not found).
func f0(v uint) *uint {
	if v == 0 {
		return nil
	}
	return &v
}

func tptr(t time.Time) *time.Time { return &t }

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
