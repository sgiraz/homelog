package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// ── buildMatchQuery (pure, no DB) ──────────────────────────────────────────

func TestBuildMatchQuery_Empty(t *testing.T) {
	if got := buildMatchQuery(""); got != "" {
		t.Fatalf("want \"\", got %q", got)
	}
}

func TestBuildMatchQuery_OnlyOperators(t *testing.T) {
	// All FTS5 operator chars — should produce empty string.
	if got := buildMatchQuery(`"*:(-)^'`); got != "" {
		t.Fatalf("want \"\", got %q", got)
	}
}

func TestBuildMatchQuery_SingleToken(t *testing.T) {
	got := buildMatchQuery("spesa")
	if got != "spesa*" {
		t.Fatalf("want %q, got %q", "spesa*", got)
	}
}

func TestBuildMatchQuery_MultiToken(t *testing.T) {
	got := buildMatchQuery("luce elettrica")
	if got != "luce* elettrica*" {
		t.Fatalf("want %q, got %q", "luce* elettrica*", got)
	}
}

func TestBuildMatchQuery_StripsOperators(t *testing.T) {
	// Operators are replaced with spaces; remaining tokens get prefix *.
	got := buildMatchQuery(`luce* (bolletta)`)
	if got != "luce* bolletta*" {
		t.Fatalf("want %q, got %q", "luce* bolletta*", got)
	}
}

// ── search scoping ─────────────────────────────────────────────────────────

type searchFixture struct {
	router   *gin.Engine
	alice    *models.User
	bob      *models.User
	aliceTok string
	bobTok   string
}

func setupSearchFixture(t *testing.T) *searchFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)

	db := testutil.NewDB(t)

	now := time.Now()
	alice := &models.User{Email: "alice@search.test", PasswordHash: "x", Name: "Alice", Role: "user", IsActive: true}
	bob := &models.User{Email: "bob@search.test", PasswordHash: "x", Name: "Bob", Role: "user", IsActive: true}
	for _, u := range []*models.User{alice, bob} {
		if err := db.Create(u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	// Two separate households so propertyIDs is never empty for either user,
	// avoiding the IN () edge case when scoping the FTS query.
	propA := models.Property{Household: &models.Household{Name: "Casa Alice"}, UserID: alice.ID, Name: "Casa Alice", Type: "owned", StartDate: now, IsCurrent: true, Residents: 1}
	propB := models.Property{Household: &models.Household{Name: "Casa Bob"}, UserID: bob.ID, Name: "Casa Bob", Type: "owned", StartDate: now, IsCurrent: true, Residents: 1}
	for _, p := range []*models.Property{&propA, &propB} {
		if err := db.Create(p).Error; err != nil {
			t.Fatalf("create property: %v", err)
		}
	}
	memberA := models.HouseholdMember{HouseholdID: propA.HouseholdID, UserID: &alice.ID, Name: "Alice", Role: "admin"}
	memberB := models.HouseholdMember{HouseholdID: propB.HouseholdID, UserID: &bob.ID, Name: "Bob", Role: "admin"}
	for _, m := range []*models.HouseholdMember{&memberA, &memberB} {
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("create member: %v", err)
		}
	}

	// Seed the FTS5 index directly:
	//   entity 1 — expense in Alice's home
	//   entity 2 — expense without a home in Alice's household
	//   entity 3 — expense without a home in Bob's household
	//   entity 4 — project without a home, created by Alice
	//   entity 10 — utility in Alice's home
	seeds := []struct {
		kind, title         string
		id, propID, hholdID uint
		userID              uint
	}{
		{"expense", "bolletta luce alice", 1, propA.ID, propA.HouseholdID, alice.ID},
		{"expense", "spesa vecchia alice", 2, 0, propA.HouseholdID, alice.ID},
		{"expense", "spesa vecchia bob", 3, 0, propB.HouseholdID, bob.ID},
		{"project", "progetto vecchia alice", 4, 0, 0, alice.ID},
		{"utility", "enel servizio alice", 10, propA.ID, 0, 0},
	}
	for _, s := range seeds {
		if err := db.Exec(
			`INSERT INTO search_index(entity_type,entity_id,property_id,household_id,user_id,title,body) VALUES (?,?,?,?,?,?,?)`,
			s.kind, s.id, s.propID, s.hholdID, s.userID, s.title, "",
		).Error; err != nil {
			t.Fatalf("seed search_index %s/%d: %v", s.kind, s.id, err)
		}
	}

	h := NewSearchHandler(db)
	r := gin.New()
	r.Use(middleware.AuthRequired())
	r.GET("/search", h.Query)

	return &searchFixture{
		router:   r,
		alice:    alice,
		bob:      bob,
		aliceTok: testutil.SignToken(t, alice),
		bobTok:   testutil.SignToken(t, bob),
	}
}

func doSearch(t *testing.T, f *searchFixture, q, token string) []map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/search?q="+q, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /search?q=%s: status=%d body=%s", q, rec.Code, rec.Body.String())
	}
	var body struct {
		Hits []map[string]any `json:"hits"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.Hits
}

func entityIDs(hits []map[string]any) []uint {
	ids := make([]uint, 0, len(hits))
	for _, h := range hits {
		ids = append(ids, uint(h["entity_id"].(float64)))
	}
	return ids
}

func containsID(hits []map[string]any, id uint) bool {
	for _, h := range hits {
		if uint(h["entity_id"].(float64)) == id {
			return true
		}
	}
	return false
}

// TestSearch_ScopedToAccessibleProperties verifies that Alice sees her own
// property-scoped expense and utility but not Bob's unscoped expense.
func TestSearch_ScopedToAccessibleProperties(t *testing.T) {
	f := setupSearchFixture(t)
	hits := doSearch(t, f, "alice", f.aliceTok)

	if !containsID(hits, 1) {
		t.Error("alice: expected to find property-scoped expense (id=1)")
	}
	if !containsID(hits, 10) {
		t.Error("alice: expected to find utility (id=10)")
	}
	if containsID(hits, 3) {
		t.Error("alice: should not see Bob's unscoped expense (id=3) — cross-user leakage")
	}
}

// TestSearch_WithoutHome_ScopedByHouseholdOrCreator verifies that an expense
// without a home is visible to its household, and a project without a home to
// its creator only.
func TestSearch_WithoutHome_ScopedByHouseholdOrCreator(t *testing.T) {
	f := setupSearchFixture(t)

	aliceHits := doSearch(t, f, "vecchia", f.aliceTok)
	if !containsID(aliceHits, 2) || !containsID(aliceHits, 4) || containsID(aliceHits, 3) {
		t.Errorf("alice: want her household's expense (2) and her project (4), not Bob's (3); got %v", entityIDs(aliceHits))
	}

	bobHits := doSearch(t, f, "vecchia", f.bobTok)
	if !containsID(bobHits, 3) || containsID(bobHits, 2) || containsID(bobHits, 4) {
		t.Errorf("bob: want only his household's expense (3); got %v", entityIDs(bobHits))
	}
}

// TestSearch_NoLeakAcrossUsers verifies that Alice cannot retrieve a
// property_id=0 row whose user_id is Bob, and vice-versa.
func TestSearch_NoLeakAcrossUsers(t *testing.T) {
	f := setupSearchFixture(t)

	aliceHits := doSearch(t, f, "bob", f.aliceTok)
	if containsID(aliceHits, 3) {
		t.Error("alice got Bob's unscoped expense (id=3) — cross-user leakage via property_id=0")
	}

	bobHits := doSearch(t, f, "alice", f.bobTok)
	if containsID(bobHits, 1) {
		t.Error("bob got Alice's property-scoped expense (id=1) — cross-property leakage")
	}
	if containsID(bobHits, 2) {
		t.Error("bob got Alice's unscoped expense (id=2) — cross-user leakage via property_id=0")
	}
}

// TestSearch_EmptyQuery_ReturnsEmpty verifies that an empty query returns 200
// with no hits (not a 500 from an empty FTS5 MATCH expression).
func TestSearch_EmptyQuery_ReturnsEmpty(t *testing.T) {
	f := setupSearchFixture(t)
	hits := doSearch(t, f, "", f.aliceTok)
	if len(hits) != 0 {
		t.Fatalf("empty query: want 0 hits, got %d", len(hits))
	}
}
