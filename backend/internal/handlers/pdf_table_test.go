package handlers

import (
	"strings"
	"testing"
)

// line lays words out from x, 5 units per character and one space apart.
func line(x, y float64, text string) []WordInfo {
	var words []WordInfo
	for _, t := range strings.Fields(text) {
		words = append(words, WordInfo{Text: t, X: x, Y: y, Width: float64(5 * len(t)), Height: 10})
		x += float64(5*len(t)) + 2
	}
	return words
}

func page(lines ...[]WordInfo) []WordInfo {
	var words []WordInfo
	for _, l := range lines {
		words = append(words, l...)
	}
	return words
}

func tableValue(t *testing.T, words []WordInfo, rule BillTemplateRule) string {
	t.Helper()
	return (&PDFHandler{}).extractValueWithRuleAndWords(rule, "", words)
}

// A readings table grows from bill to bill: the value is the last cell of its
// column in the field's format, a record line without one does not count.
func TestTableRuleReadsTheColumnCell(t *testing.T) {
	words := page(
		line(10, 100, "Reading type"), line(100, 100, "Reading date"), line(200, 100, "Reading"),
		line(10, 112, "Actual"), line(100, 112, "05/01/2026"), line(200, 112, "410"),
		line(10, 124, "Customer note"),
		line(10, 136, "R3 - Estimated"), line(100, 136, "15/06/2026"), line(200, 136, "498"),
		line(10, 148, "Period consumption"),
		line(10, 160, "Actual"), line(100, 160, "22/06/2026"), line(200, 160, "512"),
	)
	stop := "Period consumption"
	reading := BillTemplateRule{Field: "provider_reading", ValuePattern: `(\d+)`, Table: &TableRule{Column: "Reading", Stop: stop}}
	if got := tableValue(t, words, reading); got != "498" {
		t.Errorf("last reading = %q, want 498", got)
	}
	reading.Table.Cell = "first"
	if got := tableValue(t, words, reading); got != "410" {
		t.Errorf("first reading = %q, want 410", got)
	}
	kind := BillTemplateRule{Field: "reading_type", ValuePattern: `([A-Za-z]+)`, Table: &TableRule{Column: "Reading type", Stop: stop}}
	if got := tableValue(t, words, kind); got != "R3 - Estimated" {
		t.Errorf("reading type cell = %q, want the whole cell", got)
	}
}

// The same header can head several columns of one row, over two lines, and
// another table further down; the stop label tells the tables apart.
func TestTableRuleOccurrenceAndStop(t *testing.T) {
	words := page(
		line(100, 100, "Reading"), line(200, 100, "Reading"), line(300, 100, "Reading"),
		line(100, 110, "end"), line(200, 110, "start"), line(300, 110, "end"),
		line(95, 124, "1.234,56 -"), line(195, 124, "1.111,11 ="), line(295, 124, "2.468,02 -"),
		line(10, 136, "Multiplier"),
		line(100, 200, "Reading"), line(100, 210, "end"),
		line(95, 224, "9.999,99"),
	)
	rule := BillTemplateRule{Field: "provider_reading_f3", ValuePattern: `(\d{1,3}(?:[.\s]\d{3})*(?:,\d+)?)`,
		Table: &TableRule{Column: "Reading end", Occurrence: 2, Stop: "Multiplier"}}
	if got := tableValue(t, words, rule); got != "2.468,02" {
		t.Errorf("second 'Reading end' = %q, want 2.468,02", got)
	}
	rule.Table.Occurrence = 1
	if got := tableValue(t, words, rule); got != "1.234,56" {
		t.Errorf("first 'Reading end' = %q, want 1.234,56 from the table that ends at the stop label", got)
	}
}

func TestTableRuleRowLabel(t *testing.T) {
	words := page(
		line(10, 100, "Item"), line(200, 100, "Start date"), line(300, 100, "End date"),
		line(10, 112, "Household size"), line(200, 112, "01/03/2024"), line(300, 112, "30/06/2026"),
		line(10, 124, "Taxable area"), line(200, 124, "15/04/2024"), line(300, 124, "30/06/2026"),
		line(10, 140, "SURFACE DETAILS"),
	)
	rule := BillTemplateRule{Field: "period_start", ValuePattern: `(\d{2}/\d{2}/\d{4})`,
		Table: &TableRule{Column: "Start date", Stop: "SURFACE DETAILS", Cell: "row", RowLabel: "Taxable area"}}
	if got := tableValue(t, words, rule); got != "15/04/2024" {
		t.Errorf("start on the labelled row = %q, want 15/04/2024", got)
	}
}

// A label wrapped over two lines is one anchor, and a value can span words.
// A word of the label further along the same line is not part of it.
func TestAnchorOnTwoLinesWithAMultiWordValue(t *testing.T) {
	words := page(
		line(100, 100, "Payment"), line(400, 100, "DEADLINE"),
		line(95, 111, "deadline"),
		line(90, 125, "12 ottobre 2026"),
		line(400, 112, "01 gennaio 2020"),
	)
	rule := BillTemplateRule{Field: "due_date", AnchorText: "Payment deadline", AnchorDirection: "below",
		ValuePattern: `(\d{1,2}\s+[a-z]+\s+\d{4})`}
	if got := tableValue(t, words, rule); got != "12 ottobre 2026" {
		t.Errorf("due date = %q, want 12 ottobre 2026", got)
	}
}

func TestReadingType(t *testing.T) {
	words := []string{"estimated", "provisional"}
	for text, want := range map[string]string{
		"R3 - Estimated": "estimated", "Provisional Estimated": "estimated", "Actual : Measured": "actual",
	} {
		if got := readingType(text, words); got != want {
			t.Errorf("%q = %s, want %s", text, got, want)
		}
	}
	if got := readingType("R3 - Estimated", nil); got != "actual" {
		t.Errorf("without the template's words nothing is an estimate, got %s", got)
	}
}
