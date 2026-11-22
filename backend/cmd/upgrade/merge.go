package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// Merging households.
//
// The upgrade to households made one household per home, so the same people
// living in two homes ended up with two households and two balances. This step
// merges households whose account members are exactly the same people into the
// oldest of them: homes, expenses, settlements and members move over, and the
// balance between two people becomes the sum of their two balances.
//
// It changes nothing unless asked: without --apply it only prints what it
// would do.

// householdRow is one household as the merge sees it.
type householdRow struct {
	ID   uint
	Name string
}

// memberRow is one household member, deleted ones included: the ledger may
// still point at them.
type memberRow struct {
	ID          uint
	HouseholdID uint
	UserID      *uint
	Name        string
	Role        string
	IsVirtual   bool
	Deleted     bool
}

// mergePlan merges source into target.
type mergePlan struct {
	Target, Source householdRow
	// Mapped source members are replaced by their target counterpart.
	Mapped map[uint]memberRow // source member id -> target member
	// Moved source members join the target household as they are.
	Moved []memberRow
	// Skip explains why this pair cannot be merged automatically.
	Skip string
	// Counts of what moves, for the report.
	Homes, Expenses, Settlements int64
}

// planHouseholdMerges finds every household that has the same account members
// as an older one and plans its merge into the oldest.
func planHouseholdMerges(db *gorm.DB) ([]mergePlan, error) {
	var households []householdRow
	if err := db.Raw(`SELECT id, name FROM households WHERE deleted_at IS NULL ORDER BY id`).Scan(&households).Error; err != nil {
		return nil, err
	}
	var members []memberRow
	if err := db.Raw(`SELECT id, household_id, user_id, name, role, is_virtual, deleted_at IS NOT NULL AS deleted
		FROM household_members ORDER BY id`).Scan(&members).Error; err != nil {
		return nil, err
	}
	byHousehold := map[uint][]memberRow{}
	for _, m := range members {
		byHousehold[m.HouseholdID] = append(byHousehold[m.HouseholdID], m)
	}

	// Group households by the set of people with an account in them.
	groups := map[string][]householdRow{}
	var keys []string
	for _, h := range households {
		var users []string
		for _, m := range byHousehold[h.ID] {
			if m.UserID != nil && !m.IsVirtual && !m.Deleted {
				users = append(users, fmt.Sprint(*m.UserID))
			}
		}
		if len(users) == 0 {
			continue
		}
		sort.Strings(users)
		key := strings.Join(users, ",")
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], h)
	}

	var plans []mergePlan
	for _, key := range keys {
		group := groups[key]
		for _, source := range group[1:] {
			plan := planPair(group[0], source, byHousehold[group[0].ID], byHousehold[source.ID])
			if err := countMoves(db, &plan); err != nil {
				return nil, err
			}
			plans = append(plans, plan)
		}
	}
	return plans, nil
}

// planPair matches every member of source with one of target: account members
// by account, virtual ones by name. A member with no counterpart moves over.
func planPair(target, source householdRow, targetMembers, sourceMembers []memberRow) mergePlan {
	plan := mergePlan{Target: target, Source: source, Mapped: map[uint]memberRow{}}
	used := map[uint]uint{} // target member id -> source member id
	for _, sm := range sourceMembers {
		var matches []memberRow
		for _, tm := range targetMembers {
			if tm.Deleted {
				continue
			}
			switch {
			case sm.UserID != nil && tm.UserID != nil && *sm.UserID == *tm.UserID:
				matches = append(matches, tm)
			case sm.UserID == nil && tm.UserID == nil && sameName(sm.Name, tm.Name):
				matches = append(matches, tm)
			}
		}
		switch len(matches) {
		case 0:
			plan.Moved = append(plan.Moved, sm)
		case 1:
			if prev, taken := used[matches[0].ID]; taken {
				plan.Skip = fmt.Sprintf("members %d and %d would both become %q", prev, sm.ID, matches[0].Name)
				return plan
			}
			used[matches[0].ID] = sm.ID
			plan.Mapped[sm.ID] = matches[0]
		default:
			plan.Skip = fmt.Sprintf("%q matches %d members of %q", sm.Name, len(matches), target.Name)
			return plan
		}
	}
	return plan
}

func sameName(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func countMoves(db *gorm.DB, plan *mergePlan) error {
	for _, c := range []struct {
		table string
		dest  *int64
	}{
		{"properties", &plan.Homes},
		{"expenses", &plan.Expenses},
		{"settlements", &plan.Settlements},
	} {
		if err := db.Table(c.table).Where("household_id = ?", plan.Source.ID).Count(c.dest).Error; err != nil {
			return err
		}
	}
	return nil
}

// printMergePlans reports the plans in plain words.
func printMergePlans(w io.Writer, plans []mergePlan) {
	if len(plans) == 0 {
		fmt.Fprintln(w, "No households to merge: no two households have the same people.")
		return
	}
	for _, p := range plans {
		fmt.Fprintf(w, "%q (%d) into %q (%d)", p.Source.Name, p.Source.ID, p.Target.Name, p.Target.ID)
		if p.Skip != "" {
			fmt.Fprintf(w, ": skipped, %s. Rename the member, then run this again.\n", p.Skip)
			continue
		}
		fmt.Fprintf(w, ": %d homes, %d expenses, %d settlements move over.\n", p.Homes, p.Expenses, p.Settlements)
		ids := make([]uint, 0, len(p.Mapped))
		for id := range p.Mapped {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			fmt.Fprintf(w, "    member %d becomes %q (%d)\n", id, p.Mapped[id].Name, p.Mapped[id].ID)
		}
		for _, m := range p.Moved {
			fmt.Fprintf(w, "    member %q (%d) joins as is\n", m.Name, m.ID)
		}
	}
}

// applyMergePlans carries out every plan that is not skipped, each in its own
// transaction.
func applyMergePlans(db *gorm.DB, plans []mergePlan) error {
	for _, p := range plans {
		if p.Skip != "" {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error { return mergeHousehold(tx, p) }); err != nil {
			return fmt.Errorf("merge %q into %q: %w", p.Source.Name, p.Target.Name, err)
		}
	}
	return nil
}

// mergeHousehold moves everything of plan.Source into plan.Target and deletes
// plan.Source. TestMergeHouseholds_CoversSchema keeps this list complete.
func mergeHousehold(tx *gorm.DB, plan mergePlan) error {
	src, dst := plan.Source.ID, plan.Target.ID

	remap := map[uint]uint{}
	for from, to := range plan.Mapped {
		remap[from] = to.ID
	}
	for from, to := range remap {
		for _, stmt := range []string{
			`UPDATE expenses SET paid_by_member_id = ? WHERE paid_by_member_id = ?`,
			`UPDATE expense_splits SET member_id = ? WHERE member_id = ?`,
			`UPDATE settlements SET from_member_id = ? WHERE from_member_id = ?`,
			`UPDATE settlements SET to_member_id = ? WHERE to_member_id = ?`,
			`UPDATE utilities SET paid_by_member_id = ? WHERE paid_by_member_id = ?`,
		} {
			if err := tx.Exec(stmt, to, from).Error; err != nil {
				return err
			}
		}
	}
	if err := remapMemberLists(tx, "user_settings", "default_split_with_member_ids", remap); err != nil {
		return err
	}
	if err := remapMemberLists(tx, "utilities", "split_member_ids", remap); err != nil {
		return err
	}

	for from, to := range plan.Mapped {
		var role string
		if err := tx.Raw(`SELECT role FROM household_members WHERE id = ?`, from).Scan(&role).Error; err != nil {
			return err
		}
		if role == "admin" && to.Role != "admin" {
			if err := tx.Exec(`UPDATE household_members SET role = 'admin' WHERE id = ?`, to.ID).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec(`DELETE FROM household_members WHERE id = ?`, from).Error; err != nil {
			return err
		}
	}

	for _, stmt := range []string{
		`UPDATE household_members SET household_id = ? WHERE household_id = ?`,
		`UPDATE expenses SET household_id = ? WHERE household_id = ?`,
		`UPDATE settlements SET household_id = ? WHERE household_id = ?`,
	} {
		if err := tx.Exec(stmt, dst, src).Error; err != nil {
			return err
		}
	}
	// The target keeps its primary home.
	if err := tx.Exec(`UPDATE properties SET household_id = ?, is_current = false WHERE household_id = ?`, dst, src).Error; err != nil {
		return err
	}
	// A request from someone already in the target has nothing left to ask.
	if err := tx.Exec(`DELETE FROM household_join_requests WHERE household_id = ? AND user_id IN
		(SELECT user_id FROM household_members WHERE household_id = ? AND user_id IS NOT NULL)`, src, dst).Error; err != nil {
		return err
	}
	if err := tx.Exec(`UPDATE household_join_requests SET household_id = ? WHERE household_id = ?`, dst, src).Error; err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM household_settings WHERE household_id = ?`, src).Error; err != nil {
		return err
	}
	if tx.Migrator().HasTable("search_index") {
		if err := tx.Exec(`UPDATE search_index SET household_id = ? WHERE household_id = ?`, dst, src).Error; err != nil {
			return err
		}
	}
	return tx.Exec(`DELETE FROM households WHERE id = ?`, src).Error
}

// remapMemberLists rewrites the JSON member-id lists in table.column.
func remapMemberLists(tx *gorm.DB, table, column string, remap map[uint]uint) error {
	var rows []struct {
		ID   uint
		List string
	}
	if err := tx.Raw(fmt.Sprintf(`SELECT id, %s AS list FROM %s WHERE %s IS NOT NULL AND %s <> ''`, column, table, column, column)).
		Scan(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		var ids []uint
		if err := json.Unmarshal([]byte(r.List), &ids); err != nil {
			continue // not a list the app could read either
		}
		changed := false
		seen := map[uint]bool{}
		out := make([]uint, 0, len(ids))
		for _, id := range ids {
			if to, ok := remap[id]; ok {
				id, changed = to, true
			}
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
		if !changed {
			continue
		}
		raw, _ := json.Marshal(out)
		if err := tx.Exec(fmt.Sprintf(`UPDATE %s SET %s = ? WHERE id = ?`, table, column), string(raw), r.ID).Error; err != nil {
			return err
		}
	}
	return nil
}
