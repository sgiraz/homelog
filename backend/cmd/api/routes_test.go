package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// Every GET route of the real router, called with id 1 on a household that has
// one of everything, must answer without a server error. GORM takes relation
// and column names as strings, so a stale one compiles and only fails here.
func TestEveryGETRouteAnswersWithoutServerError(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "homelog.db"))
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	db := testutil.NewDB(t)
	if err := database.SeedDefaultCategories(db); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	user := models.User{Email: "owner@example.com", PasswordHash: "x", Name: "Owner", Role: "admin", IsActive: true}
	mustCreate(t, db, &user)
	if err := database.SeedDefaultData(db, user.ID); err != nil {
		t.Fatal(err)
	}
	prop := models.Property{UserID: user.ID, Name: "Casa", IsCurrent: true}
	mustCreate(t, db, &prop)
	member := models.HouseholdMember{PropertyID: prop.ID, UserID: &user.ID, Name: user.Name, Role: "admin"}
	partner := models.HouseholdMember{PropertyID: prop.ID, Name: "Partner", Role: "member"}
	mustCreate(t, db, &member, &partner)
	utility := models.Utility{UserID: user.ID, PropertyID: prop.ID, Type: "gas", Provider: "Eon", IsMetered: true}
	mustCreate(t, db, &utility)
	value := 100.0
	reading := models.MeterReading{UtilityID: utility.ID, ReadingDate: now, Value: &value}
	mustCreate(t, db, &reading)
	bill := models.Bill{UtilityID: utility.ID, BillNumber: "1", IssueDate: now, PeriodStart: now.AddDate(0, -1, 0),
		PeriodEnd: now, DueDate: now, AmountTotal: 50, UserReadingID: &reading.ID}
	mustCreate(t, db, &bill)
	mustCreate(t, db, &models.BillInstallment{BillID: bill.ID, Number: 1, DueDate: now, Amount: 50})
	mustCreate(t, db, &models.ServiceCommunication{UtilityID: utility.ID, Type: "info", Title: "Info"})
	project := models.Project{UserID: user.ID, PropertyID: &prop.ID, Name: "Kitchen", StartDate: now, EndDate: now}
	mustCreate(t, db, &project)
	expense := models.Expense{UserID: user.ID, PropertyID: &prop.ID, CategoryID: 1, Amount: 10, Date: now,
		Description: "Coffee", PaidByMemberID: member.ID, ProjectID: &project.ID}
	mustCreate(t, db, &expense)
	mustCreate(t, db, &models.ExpenseSplit{ExpenseID: expense.ID, MemberID: partner.ID, Amount: 5})
	mustCreate(t, db, &models.Settlement{PropertyID: prop.ID, FromMemberID: partner.ID, ToMemberID: member.ID, Amount: 5, Date: now})
	mustCreate(t, db, &models.Notification{UserID: user.ID, Type: "expense_shared", Title: "Shared"})

	router := newRouter(db)
	token := testutil.SignToken(t, &user)
	param := regexp.MustCompile(`:[A-Za-z]+|\*[A-Za-z]+`)
	for _, route := range router.Routes() {
		// /version asks GitHub for the latest release.
		if route.Method != http.MethodGet || route.Path == "/api/v1/version" {
			continue
		}
		path := param.ReplaceAllString(route.Path, "1")
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code >= http.StatusInternalServerError || rec.Code == http.StatusTooManyRequests {
			t.Errorf("GET %s: %d %s", path, rec.Code, rec.Body)
		}
	}
}

func mustCreate(t *testing.T, db *gorm.DB, rows ...any) {
	t.Helper()
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create %T: %v", row, err)
		}
	}
}
