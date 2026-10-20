package handlers

import (
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
)

// TableRule reads a value out of a table: the column under a header, the
// cells between the header and the label right below the table, and one of
// those cells. Only cells in the field's format count, so a record spanning
// several lines or an empty cell does not shift the pick.
type TableRule struct {
	Column   string `json:"column"`              // header text, possibly over two or three lines
	Stop     string `json:"stop,omitempty"`      // label right below the table, telling it from others with the same header
	Cell     string `json:"cell,omitempty"`      // "last" (default), "first" or "row"
	RowLabel string `json:"row_label,omitempty"` // with "row": the label on the value's row

	// Occurrence picks the n-th header from the left when the same text heads
	// several columns of one row (F1/F2/F3 readings); 0 reads them all.
	Occurrence int `json:"occurrence,omitempty"`
}

// textCell is a run of words on one line close enough to read as one value,
// such as a date written out in words or a reading type with its note.
type textCell struct {
	words      []WordInfo
	text       string
	page       int
	x0, x1, y  float64
	lineHeight float64
}

func (c textCell) bottom() float64 { return c.y + c.lineHeight }

// lineHeight is the taller of two words, at least one unit so words without a
// measured height still group by exact position.
func lineHeight(a, b WordInfo) float64 { return math.Max(1, math.Max(a.Height, b.Height)) }

func (c textCell) overlapsX(o textCell) bool { return c.x0 < o.x1 && o.x0 < c.x1 }

// sameRow tells whether two cells sit on one visual row.
func (c textCell) sameRow(o textCell) bool {
	h := math.Max(c.lineHeight, o.lineHeight)
	return c.page == o.page && math.Abs((c.y+c.lineHeight/2)-(o.y+o.lineHeight/2)) < h/2
}

// cellsOf groups words into cells: words share a line when their tops are
// within 0.3 line heights, and a cell when the gap between them is under 0.6.
// Record lines of a table can sit half a line apart, hence the tight band.
// Cells come sorted by page, line and x.
func cellsOf(words []WordInfo) []textCell {
	sorted := make([]WordInfo, len(words))
	copy(sorted, words)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Page != sorted[j].Page {
			return sorted[i].Page < sorted[j].Page
		}
		return sorted[i].Y < sorted[j].Y
	})

	var lines [][]WordInfo
	for _, w := range sorted {
		if n := len(lines); n > 0 {
			first := lines[n-1][0]
			if first.Page == w.Page && w.Y-first.Y < 0.3*lineHeight(first, w) {
				lines[n-1] = append(lines[n-1], w)
				continue
			}
		}
		lines = append(lines, []WordInfo{w})
	}

	var cells []textCell
	for _, line := range lines {
		sort.SliceStable(line, func(i, j int) bool { return line[i].X < line[j].X })
		start := 0
		for i := 1; i <= len(line); i++ {
			if i < len(line) {
				prev := line[i-1]
				if line[i].X-(prev.X+prev.Width) < 0.6*lineHeight(prev, line[i]) {
					continue
				}
			}
			cells = append(cells, newTextCell(line[start:i]))
			start = i
		}
	}
	return cells
}

func newTextCell(words []WordInfo) textCell {
	c := textCell{words: words, page: words[0].Page, x0: words[0].X, y: words[0].Y}
	texts := make([]string, len(words))
	for i, w := range words {
		texts[i] = w.Text
		c.x0 = math.Min(c.x0, w.X)
		c.x1 = math.Max(c.x1, w.X+w.Width)
		c.y = math.Min(c.y, w.Y)
		c.lineHeight = math.Max(c.lineHeight, math.Max(1, w.Height))
	}
	c.text = strings.Join(texts, " ")
	return c
}

// normalizeLabel compares labels ignoring case, spacing and trailing punctuation.
func normalizeLabel(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		words[i] = strings.TrimRight(w, ":.,;")
	}
	return strings.Join(words, " ")
}

// findHeaders returns every cell whose text is the column header, merging a
// header written over up to three stacked lines into one cell. With
// occurrence n, only the n-th of the headers sharing a row is kept.
func findHeaders(column string, occurrence int, cells []textCell) []textCell {
	target := normalizeLabel(column)
	if target == "" {
		return nil
	}
	var headers []textCell
	for i, c := range cells {
		header := c
		text := normalizeLabel(c.text)
		for k := 0; k < 2 && text != target && strings.HasPrefix(target, text+" "); k++ {
			below, ok := cellBelow(header, cells[i+1:])
			if !ok {
				break
			}
			header = mergeCells(header, below)
			text = normalizeLabel(header.text)
		}
		if text == target {
			headers = append(headers, header)
		}
	}
	if occurrence <= 0 {
		return headers
	}
	var picked []textCell
	for i, h := range headers {
		n := 1
		for _, o := range headers[:i] {
			if o.sameRow(h) {
				n++
			}
		}
		if n == occurrence {
			picked = append(picked, h)
		}
	}
	return picked
}

// cellBelow finds the cell on the next line that overlaps c horizontally.
func cellBelow(c textCell, cells []textCell) (textCell, bool) {
	for _, o := range cells {
		if o.page != c.page || o.y <= c.y+c.lineHeight/2 {
			continue
		}
		if o.y-c.bottom() > 0.8*c.lineHeight {
			break
		}
		if c.overlapsX(o) {
			return o, true
		}
	}
	return textCell{}, false
}

// mergeCells joins a header with the line below it. The column spans the
// lower line, the most specific one: a "reading" heading over "final" sits
// next to the same heading over "initial".
func mergeCells(a, b textCell) textCell {
	return textCell{
		words:      append(append([]WordInfo{}, a.words...), b.words...),
		text:       a.text + " " + b.text,
		page:       a.page,
		x0:         b.x0,
		x1:         b.x1,
		y:          a.y,
		lineHeight: b.bottom() - a.y,
	}
}

// extractFromTable applies a table rule on every page. The reading type cell
// is kept whole: its words, not a format, tell an estimate from a reading.
func (h *PDFHandler) extractFromTable(rule BillTemplateRule, words []WordInfo) string {
	var valueRe *regexp.Regexp
	if rule.ValuePattern != "" && rule.Field != "reading_type" {
		re, err := regexp.Compile("(?i)^(?:" + rule.ValuePattern + ")$")
		if err != nil {
			log.Printf("Invalid value pattern for table field %s: %v", rule.Field, err)
			return ""
		}
		valueRe = re
	}

	cells := cellsOf(words)
	type match struct {
		cell  textCell
		value string
	}
	var matches []match
	var rowLabels []AnchorMatch
	if rule.Table.Cell == "row" {
		rowLabels = findAnchorOccurrences(rule.Table.RowLabel, words)
	}

	headers := findHeaders(rule.Table.Column, rule.Table.Occurrence, cells)
	stops := make([]float64, len(headers))
	stopFound := false
	for i, header := range headers {
		stops[i] = math.Inf(1)
		if rule.Table.Stop == "" {
			continue
		}
		for _, s := range findAnchorOccurrences(rule.Table.Stop, words) {
			if s.Page == header.page && s.Words[0].Y > header.bottom() && s.Words[0].Y < stops[i] {
				stops[i] = s.Words[0].Y
				stopFound = true
			}
		}
	}

	for i, header := range headers {
		stopY := stops[i]
		// The stop label tells this table from another one with the same
		// header; a bill that lacks it everywhere still has its table.
		if stopFound && math.IsInf(stopY, 1) {
			continue
		}
		// A blank band of a line and a half also ends the table: the next section
		// often starts before the stop label.
		gap := 1.5 * math.Max(1, header.words[len(header.words)-1].Height)
		prevBottom := header.bottom()
		for _, c := range cells {
			if c.page != header.page || c.y < header.bottom()-1 || c.y >= stopY {
				continue
			}
			if c.y-prevBottom > gap {
				stopY = c.y
				break
			}
			prevBottom = math.Max(prevBottom, c.bottom())
		}

		for _, c := range cells {
			if c.page != header.page || c.y < header.bottom()-1 || c.y >= stopY || !c.overlapsX(header) {
				continue
			}
			if rule.Table.Cell == "row" && !onLabelledRow(c, rowLabels) {
				continue
			}
			value, ok := c.text, true
			if valueRe != nil {
				value, ok = wholeWordsMatch(c, valueRe)
			}
			if ok {
				matches = append(matches, match{c, strings.TrimSpace(value)})
			}
		}
	}

	if len(matches) == 0 {
		return ""
	}
	if rule.Table.Cell == "first" || rule.Table.Cell == "row" {
		return matches[0].value
	}
	return matches[len(matches)-1].value
}

// wholeWordsMatch finds the leftmost, longest run of whole words in a cell
// that the anchored pattern matches, and returns its last capture group: "-"
// after a reading or "Da" before a date stay out, and "2024" does not pass
// for a reading "202".
func wholeWordsMatch(c textCell, re *regexp.Regexp) (string, bool) {
	for i := range c.words {
		for j := len(c.words); j > i; j-- {
			if m := re.FindStringSubmatch(newTextCell(c.words[i:j]).text); m != nil {
				return m[len(m)-1], true
			}
		}
	}
	return "", false
}

func onLabelledRow(c textCell, labels []AnchorMatch) bool {
	for _, l := range labels {
		if c.sameRow(newTextCell(l.Words)) {
			return true
		}
	}
	return false
}
