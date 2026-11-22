package handlers

import (
	"errors"
	"math"
	"sort"
	"strconv"

	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// Counterpart is one other member of the household, with the running balance
// between them and the current member (positive: they owe the current member).
type Counterpart struct {
	MemberID  uint    `json:"member_id"`
	Name      string  `json:"name"`
	IsVirtual bool    `json:"is_virtual"`
	Balance   float64 `json:"balance"`
}

// errInvalidOtherMember marks an other_member_id the caller got wrong — either
// unparseable or naming a member of some other household. That is a client
// mistake (400) and must not be flattened into the empty-payload answer that
// "this household has no counterpart yet" legitimately produces.
var errInvalidOtherMember = errors.New("invalid other_member_id")

// errOtherMemberNotFound marks an other_member_id that parses but names no
// member of this household.
var errOtherMemberNotFound = errors.New("other member not found")

// pairBalances returns the running balance between currentMemberID and every
// member it shares an unsettled split with, keyed by that member's id.
// Positive means the other member owes the current one. Long-term debts are
// excluded by design: they are repaid on their own schedule from the Debiti
// tab, and netting them here would swallow every ordinary expense until the
// big debt is gone (see Expense.IsLongTermDebt).
func pairBalances(db *gorm.DB, householdID, currentMemberID uint) (map[uint]float64, error) {
	balances := map[uint]float64{}

	var settings models.HouseholdSettings
	err := db.Where("household_id = ?", householdID).First(&settings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return balances, nil
	}
	if err != nil {
		return nil, err
	}
	if !settings.SplitMode {
		return balances, nil
	}

	var splits []models.ExpenseSplit
	err = db.
		Preload("Expense").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expense_splits.is_settled = ?", false).
		Where("expenses.household_id = ?", householdID).
		Where("expenses.deleted_at IS NULL").
		Where("expenses.is_long_term_debt = ?", false).
		Where("expenses.paid_by_member_id = ? OR expense_splits.member_id = ?", currentMemberID, currentMemberID).
		Find(&splits).Error
	if err != nil {
		return nil, err
	}

	// The remaining amount, not the original share: a split may have been
	// partially paid down via the settlement ledger. Settlements are not
	// counted again — SettlementHandler.Create applies them to SettledAmount.
	for _, split := range splits {
		payer, debtor := split.Expense.PaidByMemberID, split.MemberID
		switch {
		case payer == debtor:
			continue
		case payer == currentMemberID:
			balances[debtor] += remainingOwed(split)
		case debtor == currentMemberID:
			balances[payer] -= remainingOwed(split)
		}
	}
	return balances, nil
}

// CalculateBalance calculates the balance between two members for a household.
// Positive balance means other member owes current member; negative means
// current member owes other member.
func CalculateBalance(currentMemberID, otherMemberID, householdID uint, db *gorm.DB) (float64, error) {
	balances, err := pairBalances(db, householdID, currentMemberID)
	if err != nil {
		return 0, err
	}
	return balances[otherMemberID], nil
}

// householdCounterparts lists every other member of the household with the
// running balance against current, ordered by who matters most to the
// current member: the largest balance either way first, then account members
// before virtual ones, then the oldest member.
func householdCounterparts(db *gorm.DB, householdID uint, current models.HouseholdMember) ([]Counterpart, error) {
	var others []models.HouseholdMember
	if err := db.Where("household_id = ? AND id != ?", householdID, current.ID).
		Order("id").Find(&others).Error; err != nil {
		return nil, err
	}
	balances, err := pairBalances(db, householdID, current.ID)
	if err != nil {
		return nil, err
	}

	counterparts := make([]Counterpart, 0, len(others))
	for _, m := range others {
		counterparts = append(counterparts, Counterpart{
			MemberID:  m.ID,
			Name:      m.Name,
			IsVirtual: m.IsVirtual,
			Balance:   math.Round(balances[m.ID]*100) / 100,
		})
	}
	sort.SliceStable(counterparts, func(i, j int) bool {
		a, b := math.Abs(counterparts[i].Balance), math.Abs(counterparts[j].Balance)
		if a != b {
			return a > b
		}
		return !counterparts[i].IsVirtual && counterparts[j].IsVirtual
	})
	return counterparts, nil
}

// resolveMemberPair finds the current user's member for a household, every
// counterpart, and the one the caller is looking at: the member named by
// other_member_id, or else the first counterpart. Picking "any other member"
// used to show a household of three the balance with whoever had the lowest
// id — a virtual member owed nothing while a real one owed hundreds.
//
// A missing pair comes back as gorm.ErrRecordNotFound (caller may answer with
// an empty ledger, current still set when it exists); a bad other_member_id
// as errInvalidOtherMember (unparseable) or errOtherMemberNotFound;
// anything else is a real database failure.
func resolveMemberPair(db *gorm.DB, userID, householdID uint, otherMemberIDStr string) (current models.HouseholdMember, other Counterpart, counterparts []Counterpart, err error) {
	if err = db.Where("household_id = ? AND user_id = ?", householdID, userID).First(&current).Error; err != nil {
		return current, other, nil, err
	}
	if counterparts, err = householdCounterparts(db, householdID, current); err != nil {
		return current, other, nil, err
	}

	if otherMemberIDStr != "" {
		parsed, parseErr := strconv.ParseUint(otherMemberIDStr, 10, 32)
		if parseErr != nil {
			return current, other, counterparts, errInvalidOtherMember
		}
		for _, cp := range counterparts {
			if cp.MemberID == uint(parsed) {
				return current, cp, counterparts, nil
			}
		}
		return current, other, counterparts, errOtherMemberNotFound
	}
	if len(counterparts) == 0 {
		return current, other, counterparts, gorm.ErrRecordNotFound
	}
	return current, counterparts[0], counterparts, nil
}
