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
		line(10, 100, "Tipo lettura"), line(100, 100, "Data lettura"), line(200, 100, "Lettura"),
		line(10, 112, "Rilevata"), line(100, 112, "03/01/2026"), line(200, 112, "89"),
		line(10, 124, "Autolettura nota"),
		line(10, 136, "MA - Stimata"), line(100, 136, "16/06/2026"), line(200, 136, "122"),
		line(10, 148, "Consumi del periodo"),
		line(10, 160, "Rilevata"), line(100, 160, "20/06/2026"), line(200, 160, "130"),
	)
	stop := "Consumi del periodo"
	reading := BillTemplateRule{Field: "provider_reading", ValuePattern: `(\d+)`, Table: &TableRule{Column: "Lettura", Stop: stop}}
	if got := tableValue(t, words, reading); got != "122" {
		t.Errorf("last reading = %q, want 122", got)
	}
	reading.Table.Cell = "first"
	if got := tableValue(t, words, reading); got != "89" {
		t.Errorf("first reading = %q, want 89", got)
	}
	kind := BillTemplateRule{Field: "reading_type", ValuePattern: `([A-Za-z]+)`, Table: &TableRule{Column: "Tipo lettura", Stop: stop}}
	if got := tableValue(t, words, kind); got != "MA - Stimata" {
		t.Errorf("reading type cell = %q, want the whole cell", got)
	}
}

// The same header can head several columns of one row, over two lines, and
// another table further down; the stop label tells the tables apart.
func TestTableRuleOccurrenceAndStop(t *testing.T) {
	words := page(
		line(100, 100, "Lettura"), line(200, 100, "Lettura"), line(300, 100, "Lettura"),
		line(100, 110, "finale"), line(200, 110, "iniziale"), line(300, 110, "finale"),
		line(95, 124, "2.322,88 -"), line(195, 124, "2.251,87 ="), line(295, 124, "3.078,26 -"),
		line(10, 136, "Costante"),
		line(100, 200, "Lettura"), line(100, 210, "finale"),
		line(95, 224, "9.999,99"),
	)
	rule := BillTemplateRule{Field: "provider_reading_f3", ValuePattern: `(\d{1,3}(?:[.\s]\d{3})*(?:,\d+)?)`,
		Table: &TableRule{Column: "Lettura finale", Occurrence: 2, Stop: "Costante"}}
	if got := tableValue(t, words, rule); got != "3.078,26" {
		t.Errorf("second 'Lettura finale' = %q, want 3.078,26", got)
	}
	rule.Table.Occurrence = 1
	if got := tableValue(t, words, rule); got != "2.322,88" {
		t.Errorf("first 'Lettura finale' = %q, want 2.322,88 from the table that ends at the stop label", got)
	}
}

func TestTableRuleRowLabel(t *testing.T) {
	words := page(
		line(10, 100, "Descrizione"), line(200, 100, "Data inizio"), line(300, 100, "Data fine"),
		line(10, 112, "Componenti nucleo"), line(200, 112, "19/09/2024"), line(300, 112, "31/07/2026"),
		line(10, 124, "Superficie imponibile"), line(200, 124, "02/06/2024"), line(300, 124, "31/07/2026"),
		line(10, 140, "DETTAGLIO SUPERFICI"),
	)
	rule := BillTemplateRule{Field: "period_start", ValuePattern: `(\d{2}/\d{2}/\d{4})`,
		Table: &TableRule{Column: "Data inizio", Stop: "DETTAGLIO SUPERFICI", Cell: "row", RowLabel: "Superficie imponibile"}}
	if got := tableValue(t, words, rule); got != "02/06/2024" {
		t.Errorf("start on the labelled row = %q, want 02/06/2024", got)
	}
}

// A label wrapped over two lines is one anchor, and a value can span words.
// A word of the label further along the same line is not part of it.
func TestAnchorOnTwoLinesWithAMultiWordValue(t *testing.T) {
	words := page(
		line(100, 100, "Scadenza"), line(400, 100, "PAGAMENTO"),
		line(95, 111, "pagamento"),
		line(90, 125, "12 ottobre 2026"),
		line(400, 112, "01 gennaio 2020"),
	)
	rule := BillTemplateRule{Field: "due_date", AnchorText: "Scadenza pagamento", AnchorDirection: "below",
		ValuePattern: `(\d{1,2}\s+[a-z]+\s+\d{4})`}
	if got := tableValue(t, words, rule); got != "12 ottobre 2026" {
		t.Errorf("due date = %q, want 12 ottobre 2026", got)
	}
}

func TestReadingType(t *testing.T) {
	words := []string{"stimata", "presunta"}
	for text, want := range map[string]string{
		"MA - Stimata": "estimated", "Presunta Stimata E.ON": "estimated", "Effettiva : Rilevata": "actual",
	} {
		if got := readingType(text, words); got != want {
			t.Errorf("%q = %s, want %s", text, got, want)
		}
	}
	if got := readingType("MA - Stimata", nil); got != "actual" {
		t.Errorf("without the template's words nothing is an estimate, got %s", got)
	}
}
