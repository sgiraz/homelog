package handlers

import (
	"math"
	"regexp"
	"strings"
	"unicode"

	"github.com/sgiraz/homelog/internal/models"
)

const (
	numberPattern = `(\d{1,3}(?:[.\s]\d{3})*(?:,\d+)?)`
	monthPattern  = `(?:gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)`
	datePattern   = `(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4}|\d{1,2}\s+` + monthPattern + `\s+\d{4})`
)

var monthWord = regexp.MustCompile(`(?i)^` + monthPattern + `$`)

// valuePatterns is the format each field's value is read in. Go RE2.
var valuePatterns = map[string]string{
	"amount_total":           `(\d{1,3}(?:[.\s]\d{3})*,\d{2})`,
	"consumption_total":      numberPattern,
	"provider_reading":       numberPattern,
	"provider_reading_f1":    numberPattern,
	"provider_reading_f2":    numberPattern,
	"provider_reading_f3":    numberPattern,
	"conversion_coefficient": `(\d+,\d+)`,
	"bill_number":            `([A-Za-z0-9][A-Za-z0-9/-]*\d[A-Za-z0-9/-]*)`,
	"meter_serial":           `([A-Za-z0-9][A-Za-z0-9/-]{3,})`,
	"reading_type":           `(.+)`,
	"issue_date":             datePattern,
	"due_date":               datePattern,
	"period_start":           datePattern,
	"period_end":             datePattern,
	"provider_reading_date":  datePattern,
}

// RuleSuggestion is a rule for the value the user picked on a sample bill,
// already run on that same bill: Value is what it extracts there and always
// equals the picked value.
type RuleSuggestion struct {
	Rule  BillTemplateRule `json:"rule"`
	Value string           `json:"value"`
}

// suggestRules proposes, best first, the rules that read the value under the
// picked word back from the bill: by table column when the value sits in a
// column of like values, by the label left of it or above it otherwise. A
// rule that does not reproduce the value is never proposed. cell is the whole
// text the picked word belongs to, for the reading type words.
func (h *PDFHandler) suggestRules(field string, picked WordInfo, words []WordInfo) (suggestions []RuleSuggestion, cell string) {
	cells := cellsOf(words)
	ci, wi := locateWord(cells, picked)
	if ci < 0 {
		return nil, ""
	}
	c := cells[ci]
	cell = c.text

	if isMultiLineField(field) {
		for _, label := range labelsAbove(c, cells, false) {
			rule := BillTemplateRule{Field: field, AnchorText: label, AnchorDirection: "below", MultiLine: true}
			if value := h.extractValueWithRuleAndWords(rule, "", words); value != "" {
				return []RuleSuggestion{{rule, value}}, cell
			}
		}
		return nil, cell
	}

	pattern, ok := valuePatterns[field]
	if !ok {
		return nil, cell
	}
	value, patterns := valueInCell(c, wi, pattern)
	if value == "" {
		return nil, cell
	}

	// Like values aligned above the value: a table reads it, even with a
	// label beside it ("A 30/09/2026" closes a record). Otherwise a label does.
	var candidates []BillTemplateRule
	tables := tableCandidates(field, c, cells)
	anchors := anchorCandidates(field, c, wi, cells)
	if !inColumn(c, cells, pattern) {
		candidates = append(anchors, tables...)
	} else {
		candidates = append(tables, anchors...)
	}
	same := func(got string) bool { return got == value }
	if field == "reading_type" {
		same = func(got string) bool { return normalizeLabel(got) == normalizeLabel(c.text) }
	}

	seen := map[string]bool{}
	for _, rule := range candidates {
		for _, p := range patterns {
			rule.ValuePattern = p
			got := h.extractValueWithRuleAndWords(rule, "", words)
			if !same(got) {
				continue
			}
			key := describeRule(rule)
			if !seen[key] {
				seen[key] = true
				suggestions = append(suggestions, RuleSuggestion{rule, got})
			}
			break
		}
	}
	return suggestions, cell
}

func isMultiLineField(field string) bool {
	for _, set := range models.BillFieldCatalogue {
		for _, f := range set.Fields {
			if f.Key == field {
				return f.MultiLine
			}
		}
	}
	return false
}

// locateWord finds the cell and word index of the picked word. The wizard
// shows a date written out as one box, so the pick may span several words: it
// starts at the first one.
func locateWord(cells []textCell, picked WordInfo) (int, int) {
	for ci, c := range cells {
		if c.page != picked.Page {
			continue
		}
		for wi, w := range c.words {
			if strings.HasPrefix(picked.Text, w.Text) && math.Abs(w.X-picked.X) < 0.5 && math.Abs(w.Y-picked.Y) < 2 {
				return ci, wi
			}
		}
	}
	return -1, -1
}

// valueInCell reads the field's value from the words around the picked one
// and returns it with the value patterns to try: the bare format first, then
// the format inside the cell's other words ("2026 / 249743", "Da 31/07/2026")
// with numbers generalized, for a value that only its neighbours single out.
func valueInCell(c textCell, wi int, pattern string) (string, []string) {
	re := regexp.MustCompile("(?i)^(?:" + pattern + ")$")
	for i := 0; i <= wi; i++ {
		for j := len(c.words); j > wi; j-- {
			m := re.FindStringSubmatch(newTextCell(c.words[i:j]).text)
			if m == nil {
				continue
			}
			patterns := []string{pattern}
			if i > 0 || j < len(c.words) {
				patterns = append(patterns, contextPattern(c.words[:i], pattern, c.words[j:]))
			}
			return m[len(m)-1], patterns
		}
	}
	// The value can start inside a word, a preposition glued to a date.
	text := c.text
	loose := regexp.MustCompile("(?i)" + pattern)
	if loc := loose.FindStringSubmatchIndex(text); loc != nil {
		before, after := text[:loc[0]], text[loc[1]:]
		value := text[loc[len(loc)-2]:loc[len(loc)-1]]
		return value, []string{generalize(before) + pattern + generalize(after)}
	}
	return "", nil
}

func contextPattern(before []WordInfo, pattern string, after []WordInfo) string {
	var parts []string
	for _, w := range before {
		parts = append(parts, generalize(w.Text))
	}
	parts = append(parts, pattern)
	for _, w := range after {
		parts = append(parts, generalize(w.Text))
	}
	return strings.Join(parts, `\s+`)
}

var digitRun = regexp.MustCompile(`\d+`)

// generalize quotes text for a pattern, digits standing for any number and a
// month name for any month: the context words of a sample bill must not carry
// its values, a period's other date included.
func generalize(text string) string {
	if monthWord.MatchString(text) {
		return monthPattern
	}
	var b strings.Builder
	last := 0
	for _, loc := range digitRun.FindAllStringIndex(text, -1) {
		b.WriteString(regexp.QuoteMeta(text[last:loc[0]]))
		b.WriteString(`\d+`)
		last = loc[1]
	}
	b.WriteString(regexp.QuoteMeta(text[last:]))
	return strings.ReplaceAll(b.String(), " ", `\s+`)
}

var valueLike = regexp.MustCompile(`\d{3,}|\d[.,/:]\d`)

// isLabel tells a label from a value: a few words with letters and no
// number or date in them. A short code such as "F1" is still a label.
func isLabel(text string) bool {
	if len(strings.Fields(text)) > 6 || valueLike.MatchString(text) {
		return false
	}
	for _, r := range text {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// inColumn tells whether other cells in the same format sit above the picked
// one, aligned with it on the left or the right as table columns are.
func inColumn(c textCell, cells []textCell, pattern string) bool {
	re := regexp.MustCompile("(?i)^(?:" + pattern + ")$")
	for _, o := range cells {
		if o.page != c.page || o.y >= c.y || c.y-o.y > 15*c.lineHeight ||
			math.Abs(o.x0-c.x0) > 1.5 && math.Abs(o.x1-c.x1) > 1.5 {
			continue
		}
		if _, ok := wholeWordsMatch(o, re); ok {
			return true
		}
	}
	return false
}

// labelsAbove lists the label cells right above c, nearest first, each also
// joined with the label line over it (a header or label wrapped on two lines).
// For table headers only rows of labels count, past the records above c.
func labelsAbove(c textCell, cells []textCell, headers bool) []string {
	reach := 15.0
	if headers {
		reach = 25
	}
	var above []textCell
	for _, o := range cells {
		if o.page == c.page && o.y < c.y && c.y-o.y < reach*c.lineHeight && o.overlapsX(c) && isLabel(o.text) &&
			(!headers || headerRow(o, cells)) {
			above = append(above, o)
		}
	}
	var labels []string
	for i := len(above) - 1; i >= 0 && len(labels) < 6; i-- {
		o := above[i]
		if i > 0 && above[i-1].overlapsX(o) && o.y-above[i-1].bottom() < 0.8*o.lineHeight &&
			isLabel(above[i-1].text+" "+o.text) {
			labels = append(labels, above[i-1].text+" "+o.text)
		}
		labels = append(labels, o.text)
	}
	return labels
}

// leftCell is the nearest cell to the left of c on its row.
func leftCell(c textCell, cells []textCell) *textCell {
	var left *textCell
	for i := range cells {
		o := &cells[i]
		if o.sameRow(c) && o.x1 <= c.x0 && (left == nil || o.x1 > left.x1) {
			left = o
		}
	}
	return left
}

func anchorCandidates(field string, c textCell, wi int, cells []textCell) []BillTemplateRule {
	var rules []BillTemplateRule
	// Label words before the value in its own cell ("n° 98142292").
	var inCell []string
	for i := wi - 1; i >= 0 && len(inCell) < 4; i-- {
		if !isLabel(c.words[i].Text) {
			break
		}
		inCell = append([]string{c.words[i].Text}, inCell...)
	}
	if len(inCell) > 0 {
		rules = append(rules, BillTemplateRule{Field: field, AnchorText: strings.Join(inCell, " "), AnchorDirection: "right"})
	}
	if left := leftCell(c, cells); left != nil && isLabel(left.text) {
		rules = append(rules, BillTemplateRule{Field: field, AnchorText: left.text, AnchorDirection: "right"})
	}
	for _, label := range labelsAbove(c, cells, false) {
		rules = append(rules, BillTemplateRule{Field: field, AnchorText: label, AnchorDirection: "below"})
	}
	return rules
}

// tableCandidates builds table rules for each header above the picked cell,
// ending at the label that closes its records, and reading the last cell, the
// one on the row's label, or the first.
func tableCandidates(field string, c textCell, cells []textCell) []BillTemplateRule {
	var rules []BillTemplateRule
	for _, header := range labelsAbove(c, cells, true) {
		headers := findHeaders(header, 0, cells)
		var h *textCell
		for i := range headers {
			if headers[i].page == c.page && headers[i].y < c.y && headers[i].overlapsX(c) {
				if h == nil || headers[i].y > h.y {
					h = &headers[i]
				}
			}
		}
		if h == nil {
			continue
		}
		occurrence := 0
		n := 0
		for _, o := range headers {
			if o.sameRow(*h) {
				n++
				if o.x0 == h.x0 {
					occurrence = n
				}
			}
		}
		if n < 2 {
			occurrence = 0
		}
		table := TableRule{Column: header, Occurrence: occurrence, Stop: stopLabel(*h, c, cells)}

		last := table
		last.Cell = "last"
		rules = append(rules, BillTemplateRule{Field: field, Table: &last})
		if label := rowLabel(c, cells); label != "" {
			row := table
			row.Cell, row.RowLabel = "row", label
			rules = append(rules, BillTemplateRule{Field: field, Table: &row})
		}
		first := table
		first.Cell = "first"
		rules = append(rules, BillTemplateRule{Field: field, Table: &first})
	}
	return rules
}

// headerRow tells whether a header cell heads columns: it shares its row
// with other labels and no number or date. A row holding a value is a record;
// a lone label is a title.
func headerRow(header textCell, cells []textCell) bool {
	n := 0
	for _, o := range cells {
		if !o.sameRow(header) {
			continue
		}
		if valueLike.MatchString(o.text) {
			return false
		}
		n++
	}
	return n > 1
}

// stopLabel is the label opening the first row below the picked cell that is
// not one more record: its label starts no row between the header and the
// pick, as a totals row closes the rows of actual and estimated readings.
func stopLabel(header, picked textCell, cells []textCell) string {
	starts := map[string]bool{}
	var candidates []textCell
	for _, o := range cells {
		if o.page != header.page || o.y < header.bottom()-1 || o.x0 > picked.x0 {
			continue
		}
		if first := rowStart(o, cells); first.x0 != o.x0 {
			continue
		}
		if o.y <= picked.y {
			starts[normalizeLabel(o.text)] = true
		} else {
			candidates = append(candidates, o)
		}
	}
	for _, o := range candidates {
		if isLabel(o.text) && !starts[normalizeLabel(o.text)] {
			fields := strings.Fields(o.text)
			if len(fields) > 4 {
				fields = fields[:4]
			}
			return strings.Join(fields, " ")
		}
	}
	return ""
}

func rowStart(c textCell, cells []textCell) textCell {
	first := c
	for _, o := range cells {
		if o.sameRow(c) && o.x0 < first.x0 {
			first = o
		}
	}
	return first
}

// rowLabel is the label at the start of the picked cell's row, if any.
func rowLabel(c textCell, cells []textCell) string {
	first := rowStart(c, cells)
	if first.x0 >= c.x0 || !isLabel(first.text) {
		return ""
	}
	return first.text
}

func describeRule(r BillTemplateRule) string {
	if r.Table != nil {
		return strings.Join([]string{"table", r.Table.Column, r.Table.Stop, r.Table.Cell, r.Table.RowLabel}, "|")
	}
	return strings.Join([]string{"anchor", r.AnchorText, r.AnchorDirection, r.ValuePattern}, "|")
}
