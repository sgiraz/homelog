package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// SearchHandler owns GET /search. It queries the unified FTS5 index, scoped to
// the properties the current user has access to (owned or member).
type SearchHandler struct {
	db *gorm.DB
}

func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

// SearchHit is one row in the response. entity_type + entity_id let the
// frontend dispatch to the right destination; snippet is FTS5's highlighted
// excerpt (delimited by ASCII SOH/STX, rendered as <mark> by the frontend).
// ParentID is currently only set for bills (→ owning utility) so the UI can
// deep-link to the utility detail page instead of the utilities index.
type SearchHit struct {
	EntityType string `json:"entity_type"`
	EntityID   uint   `json:"entity_id"`
	PropertyID uint   `json:"property_id"`
	ParentID   uint   `json:"parent_id,omitempty"`
	Title      string `json:"title"`
	Snippet    string `json:"snippet"`
}

// Query handles GET /api/v1/search?q=...
//
// Scoping: we collect every property_id the user can read (owned + member) and
// restrict the FTS5 lookup to those. Users with no property memberships do not
// search the global index, but may still match legacy unscoped records tied to
// their user_id (rows where property_id = 0).
//
// Query shape: we sanitise the user input into a prefix MATCH (each token gets
// a trailing *) so partial words work for live-search as the user types on
// mobile. Diacritics are folded by the tokenizer, so "spesa" ≈ "spèsa".
func (h *SearchHandler) Query(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	raw := strings.TrimSpace(c.Query("q"))
	if raw == "" {
		c.JSON(http.StatusOK, gin.H{"hits": []SearchHit{}})
		return
	}

	// Collect accessible property IDs: owned + member-of.
	var ownedIDs []uint
	h.db.Model(&models.Property{}).Where("user_id = ?", userID).Pluck("id", &ownedIDs)

	var memberIDs []uint
	h.db.Model(&models.HouseholdMember{}).Where("user_id = ?", userID).Pluck("property_id", &memberIDs)

	// Merge + dedupe accessible property IDs (no sentinel 0 — unscoped entities
	// are handled separately in the query via the user_id column).
	idSet := map[uint]struct{}{}
	for _, id := range ownedIDs {
		idSet[id] = struct{}{}
	}
	for _, id := range memberIDs {
		idSet[id] = struct{}{}
	}
	propertyIDs := make([]uint, 0, len(idSet))
	for id := range idSet {
		propertyIDs = append(propertyIDs, id)
	}

	match := buildMatchQuery(raw)
	if match == "" {
		c.JSON(http.StatusOK, gin.H{"hits": []SearchHit{}})
		return
	}

	type row struct {
		EntityType string
		EntityID   uint
		PropertyID uint
		Title      string
		Snippet    string
	}

	var rows []row
	// snippet(table, colIndex, open, close, ellipsis, tokens). Column 5 is body
	// (0=entity_type,1=entity_id,2=property_id,3=user_id,4=title,5=body).
	// Delimiters are ASCII control chars (SOH/STX) — cannot appear in user text,
	// so frontend replacement is unambiguous.
	// Two access paths:
	//   1. Rows belonging to a real property the user owns or is a member of.
	//   2. Rows with property_id=0 (unscoped expenses/projects) whose user_id
	//      matches the current user — prevents cross-user leakage of legacy data.
	err := h.db.Raw(`
		SELECT entity_type AS entity_type,
		       entity_id   AS entity_id,
		       property_id AS property_id,
		       title       AS title,
		       snippet(search_index, 5, x'01', x'02', '…', 12) AS snippet
		FROM search_index
		WHERE search_index MATCH ?
		  AND (property_id IN ? OR (property_id = 0 AND user_id = ?))
		ORDER BY bm25(search_index)
		LIMIT 50
	`, match, propertyIDs, userID).Scan(&rows).Error

	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, apierr.CodeServerError, "Search failed")
		return
	}

	// Collect bill IDs so we can resolve each bill → utility_id in one query
	// and surface it as parent_id on the hit. Without this, the UI can only
	// land on the utilities index when the user taps a bill result.
	var billIDs []uint
	for _, r := range rows {
		if r.EntityType == "bill" {
			billIDs = append(billIDs, r.EntityID)
		}
	}
	billToUtility := map[uint]uint{}
	if len(billIDs) > 0 {
		type billRow struct {
			ID        uint
			UtilityID uint
		}
		var brs []billRow
		h.db.Model(&models.Bill{}).
			Select("id, utility_id").
			Where("id IN ?", billIDs).
			Scan(&brs)
		for _, br := range brs {
			billToUtility[br.ID] = br.UtilityID
		}
	}

	hits := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		hit := SearchHit{
			EntityType: r.EntityType,
			EntityID:   r.EntityID,
			PropertyID: r.PropertyID,
			Title:      r.Title,
			Snippet:    r.Snippet,
		}
		if r.EntityType == "bill" {
			hit.ParentID = billToUtility[r.EntityID]
		}
		hits = append(hits, hit)
	}
	c.JSON(http.StatusOK, gin.H{"hits": hits})
}

// buildMatchQuery turns arbitrary user text into a safe FTS5 MATCH expression.
// We strip characters FTS5 treats as operators (", *, :, (, ), -, ^), split on
// whitespace, and append * to each token for prefix matching. An empty input
// returns "" so the caller can short-circuit.
func buildMatchQuery(input string) string {
	// Drop FTS5 operator characters. We quote nothing — we control the tokens.
	replacer := strings.NewReplacer(
		`"`, " ",
		`*`, " ",
		`:`, " ",
		`(`, " ",
		`)`, " ",
		`-`, " ",
		`^`, " ",
		`'`, " ",
	)
	cleaned := replacer.Replace(input)
	fields := strings.Fields(cleaned)
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) == 0 {
			continue
		}
		parts = append(parts, f+"*")
	}
	return strings.Join(parts, " ")
}
