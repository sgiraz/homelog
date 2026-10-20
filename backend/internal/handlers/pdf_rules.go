package handlers

import (
	"encoding/json"
	"log"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/sgiraz/homelog/internal/models"
)

// AnchorMatch represents a found anchor text location in the document.
type AnchorMatch struct {
	Words    []WordInfo
	LastWord WordInfo
	Page     int
}

// extractBillDataWithTemplateAndWords extracts bill data using template rules with word-position support.
func (h *PDFHandler) extractBillDataWithTemplateAndWords(text string, template models.BillTemplate, pdfURL string, words []WordInfo) ExtractedBillData {
	var rules []BillTemplateRule
	json.Unmarshal([]byte(template.ExtractionRules), &rules)

	extracted := ExtractedBillData{
		PDFURL:   pdfURL,
		RawText:  text,
		Provider: template.Provider,
	}

	for _, rule := range rules {
		value := h.extractValueWithRuleAndWords(rule, text, words)
		if value == "" {
			log.Printf("Template extraction FAILED for field '%s' (anchor='%s', pattern='%s')",
				rule.Field, rule.AnchorText, rule.Pattern)
			continue
		}
		log.Printf("Template extraction OK for field '%s': '%s'", rule.Field, value)

		switch rule.Field {
		case "bill_number":
			extracted.BillNumber = value
		case "issue_date":
			extracted.IssueDate = h.parseDate(value, rule.Format)
		case "period_start":
			extracted.PeriodStart = h.parseDate(value, rule.Format)
		case "period_end":
			extracted.PeriodEnd = h.parseDate(value, rule.Format)
		case "due_date":
			extracted.DueDate = h.parseDate(value, rule.Format)
		case "amount_total":
			extracted.AmountTotal = h.parseAmount(value)
		case "consumption_total":
			extracted.ConsumptionTotal = h.parseAmount(value)
		case "conversion_coefficient":
			coeff := h.parseAmount(value)
			extracted.ConversionCoefficient = &coeff
		case "provider_reading":
			reading := h.parseAmount(value)
			extracted.ProviderReading = &reading
		case "provider_reading_f1":
			reading := h.parseAmount(value)
			extracted.ProviderReadingF1 = &reading
		case "provider_reading_f2":
			reading := h.parseAmount(value)
			extracted.ProviderReadingF2 = &reading
		case "provider_reading_f3":
			reading := h.parseAmount(value)
			extracted.ProviderReadingF3 = &reading
		case "provider_reading_date":
			extracted.ProviderReadingDate = h.parseDate(value, rule.Format)
		case "reading_type":
			extracted.ReadingType = readingType(value, rule.EstimatedWords)
		case "meter_serial":
			extracted.MeterSerial = value
		case "communication_text":
			extracted.CommunicationText = value
		}
	}

	return extracted
}

// readingType reads the reading type text: estimated when it holds one of the
// template's words for an estimate, actual otherwise.
func readingType(text string, estimatedWords []string) string {
	lower := strings.ToLower(text)
	for _, w := range estimatedWords {
		if w = strings.ToLower(strings.TrimSpace(w)); w != "" && strings.Contains(lower, w) {
			return "estimated"
		}
	}
	return "actual"
}

// extractValueWithRuleAndWords applies the rule's own strategy: a table or an
// anchor rule that finds nothing leaves the field empty rather than falling
// back to a guess. Rules with neither keep the legacy position → pattern path.
func (h *PDFHandler) extractValueWithRuleAndWords(rule BillTemplateRule, text string, words []WordInfo) string {
	if rule.Table != nil {
		return h.extractFromTable(rule, words)
	}
	if rule.MultiLine && rule.AnchorText != "" {
		return h.extractMultiLineByAnchor(rule, words)
	}
	if rule.AnchorText != "" {
		return h.extractByAnchor(rule, words)
	}

	if rule.X > 0 && rule.Y > 0 && len(words) > 0 {
		value := h.extractByPosition(rule, words)
		if value != "" {
			log.Printf("Field %s matched by position: %s", rule.Field, value)
			return value
		}
		log.Printf("Field %s: position-based match failed, falling back to pattern", rule.Field)
	}

	return h.extractByPattern(rule, text)
}

// extractByPosition finds the word closest to the stored (x,y) and validates its context.
func (h *PDFHandler) extractByPosition(rule BillTemplateRule, words []WordInfo) string {
	pageWords := make([]WordInfo, 0)
	for _, w := range words {
		if w.Page == rule.Page {
			pageWords = append(pageWords, w)
		}
	}

	if len(pageWords) == 0 {
		return ""
	}

	var bestMatch WordInfo
	bestDistance := float64(999999)
	tolerance := 20.0

	for _, w := range pageWords {
		dx := w.X - rule.X
		dy := w.Y - rule.Y
		distance := dx*dx + dy*dy

		if distance < bestDistance && distance < tolerance*tolerance {
			bestDistance = distance
			bestMatch = w
		}
	}

	if bestMatch.Text != "" {
		if rule.ContextLeft != "" {
			leftContext := h.getContextLeft(bestMatch, pageWords)
			if !strings.Contains(strings.ToLower(leftContext), strings.ToLower(rule.ContextLeft)) {
				log.Printf("Context validation failed for %s: expected '%s' on left, got '%s'",
					rule.Field, rule.ContextLeft, leftContext)
			}
		}

		if rule.ContextAbove != "" {
			aboveContext := h.getContextAbove(bestMatch, pageWords)
			if !strings.Contains(strings.ToLower(aboveContext), strings.ToLower(rule.ContextAbove)) {
				log.Printf("Context validation failed for %s: expected '%s' above, got '%s'",
					rule.Field, rule.ContextAbove, aboveContext)
			}
		}

		return bestMatch.Text
	}

	return ""
}

// getContextLeft returns up to three words immediately left of the target, same line.
func (h *PDFHandler) getContextLeft(target WordInfo, words []WordInfo) string {
	var leftWords []string
	for _, w := range words {
		if abs(w.Y-target.Y) < 5 && w.X < target.X {
			leftWords = append(leftWords, w.Text)
		}
	}
	sort.Slice(leftWords, func(i, j int) bool {
		return i < j
	})
	if len(leftWords) > 3 {
		leftWords = leftWords[len(leftWords)-3:]
	}
	return strings.Join(leftWords, " ")
}

// getContextAbove returns words directly above target within the same column.
func (h *PDFHandler) getContextAbove(target WordInfo, words []WordInfo) string {
	var aboveWords []string
	for _, w := range words {
		if w.Y < target.Y && w.Y > target.Y-120 {
			if w.X < target.X+target.Width && w.X+w.Width > target.X {
				aboveWords = append(aboveWords, w.Text)
			}
		}
	}
	return strings.Join(aboveWords, " ")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// extractByAnchor searches for the rule's anchor text and reads the value in
// the region of interest of its first occurrence that has one, nearest to the
// anchor. A value may span words (a date written out); where the user clicked
// on the sample bill plays no part, since tables move from bill to bill.
func (h *PDFHandler) extractByAnchor(rule BillTemplateRule, words []WordInfo) string {
	anchors := findAnchorOccurrences(rule.AnchorText, words)
	if len(anchors) == 0 {
		return ""
	}

	if rule.ValuePattern == "" {
		return ""
	}
	valueRe, err := regexp.Compile("(?i)^(?:" + rule.ValuePattern + ")$")
	if err != nil {
		log.Printf("Invalid value pattern for anchor search field %s: %v", rule.Field, err)
		return ""
	}

	for _, anchor := range anchors {
		best, bestDist := "", math.Inf(1)
		for _, c := range cellsOf(searchROI(anchor, rule.AnchorDirection, words)) {
			for i := range c.words {
				dx := c.words[i].X - anchor.LastWord.X
				dy := c.words[i].Y - anchor.LastWord.Y
				dist := dx*dx + dy*dy
				for j := len(c.words); j > i; j-- {
					value := newTextCell(c.words[i:j]).text
					if (dist < bestDist || dist == bestDist && len(value) > len(best)) && valueRe.MatchString(value) {
						best, bestDist = value, dist
					}
				}
			}
		}
		if best != "" {
			if m := valueRe.FindStringSubmatch(best); len(m) > 1 {
				return strings.TrimSpace(m[len(m)-1])
			}
			return best
		}
	}
	return ""
}

// extractMultiLineByAnchor collects text lines below an anchor heading until a
// large vertical gap (new section) — used for communication / notes blocks.
func (h *PDFHandler) extractMultiLineByAnchor(rule BillTemplateRule, words []WordInfo) string {
	anchors := findAnchorOccurrences(rule.AnchorText, words)
	if len(anchors) == 0 {
		return ""
	}

	anchor := anchors[0]
	anchorPage := anchor.Page
	anchorY := anchor.LastWord.Y
	anchorX := anchor.Words[0].X

	type lineGroup struct {
		y     float64
		words []WordInfo
	}

	var lines []lineGroup
	maxY := anchorY + 400

	for _, w := range words {
		if w.Page != anchorPage {
			continue
		}
		if w.Y <= anchorY+5 {
			continue
		}
		if w.Y > maxY {
			continue
		}
		if w.X < anchorX-50 {
			continue
		}

		found := false
		for i := range lines {
			if abs(lines[i].y-w.Y) < 8 {
				lines[i].words = append(lines[i].words, w)
				found = true
				break
			}
		}
		if !found {
			lines = append(lines, lineGroup{y: w.Y, words: []WordInfo{w}})
		}
	}

	if len(lines) == 0 {
		return ""
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i].y < lines[j].y
	})

	var resultLines []string
	prevY := anchorY
	for _, line := range lines {
		if line.y-prevY > 40 && len(resultLines) > 0 {
			break
		}

		sort.Slice(line.words, func(i, j int) bool {
			return line.words[i].X < line.words[j].X
		})

		var lineText []string
		for _, w := range line.words {
			lineText = append(lineText, w.Text)
		}
		text := strings.Join(lineText, " ")
		if text != "" {
			resultLines = append(resultLines, text)
		}
		prevY = line.y
	}

	return strings.Join(resultLines, "\n")
}

// findAnchorOccurrences finds every case-insensitive occurrence of a (possibly
// multi-word) anchor across all pages, in reading order. Each next word is the
// nearest one close to the right on the same line or, for a label wrapped over two
// lines, right below it.
func findAnchorOccurrences(anchorText string, words []WordInfo) []AnchorMatch {
	anchorWords := strings.Fields(normalizeLabel(anchorText))
	if len(anchorWords) == 0 {
		return nil
	}

	var matches []AnchorMatch
	for _, start := range words {
		if normalizeLabel(start.Text) != anchorWords[0] {
			continue
		}
		matched := []WordInfo{start}
		for _, aw := range anchorWords[1:] {
			next, ok := nextAnchorWord(matched, aw, words)
			if !ok {
				break
			}
			matched = append(matched, next)
		}
		if len(matched) == len(anchorWords) {
			matches = append(matches, AnchorMatch{Words: matched, LastWord: matched[len(matched)-1], Page: start.Page})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i].Words[0], matches[j].Words[0]
		if a.Page != b.Page {
			return a.Page < b.Page
		}
		if math.Abs(a.Y-b.Y) > 0.3*lineHeight(a, b) {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
	return matches
}

// nextAnchorWord looks for text after the last matched word; a word on the
// next line must sit under the matched words, not necessarily the last one.
func nextAnchorWord(matched []WordInfo, text string, words []WordInfo) (WordInfo, bool) {
	prev := matched[len(matched)-1]
	left := matched[0].X
	for _, m := range matched {
		left = math.Min(left, m.X)
	}
	var right, below *WordInfo
	for i := range words {
		w := &words[i]
		if w.Page != prev.Page || normalizeLabel(w.Text) != text {
			continue
		}
		switch {
		case math.Abs(w.Y-prev.Y) < 0.3*lineHeight(*w, prev) && w.X > prev.X &&
			w.X-(prev.X+prev.Width) < 1.5*lineHeight(*w, prev):
			if right == nil || w.X < right.X {
				right = w
			}
		case w.Y > prev.Y && w.Y-(prev.Y+prev.Height) < 0.8*lineHeight(prev, prev) &&
			w.X < prev.X+prev.Width && left < w.X+w.Width:
			if below == nil || w.Y < below.Y {
				below = w
			}
		}
	}
	if right != nil {
		return *right, true
	}
	if below != nil {
		return *below, true
	}
	return WordInfo{}, false
}

// searchROI returns words in the Region of Interest around an anchor.
// direction: "right", "below", or "right_or_below" (default).
func searchROI(anchor AnchorMatch, direction string, allWords []WordInfo) []WordInfo {
	ref := anchor.LastWord
	var results []WordInfo

	if direction == "" {
		direction = "right_or_below"
	}

	if direction == "right" || direction == "right_or_below" {
		// The band the anchor covers, all its lines: a value printed larger
		// than a two-line label sits beside both.
		top, bottom, right := ref.Y, ref.Y, ref.X+ref.Width
		for _, a := range anchor.Words {
			top = math.Min(top, a.Y)
			bottom = math.Max(bottom, a.Y+math.Max(1, a.Height))
			right = math.Max(right, a.X+a.Width)
		}
		for _, w := range allWords {
			if w.Page != ref.Page {
				continue
			}
			if w.Y < bottom && w.Y+math.Max(1, w.Height) > top && w.X > right-5 {
				results = append(results, w)
			}
		}
	}

	if direction == "below" || direction == "right_or_below" {
		anchorXMin := anchor.Words[0].X
		anchorXMax := ref.X + ref.Width
		for _, w := range allWords {
			if w.Page != ref.Page {
				continue
			}
			if w.Y > ref.Y+5 && w.Y < ref.Y+150 {
				if w.X < anchorXMax+100 && w.X+w.Width > anchorXMin-50 {
					results = append(results, w)
				}
			}
		}
	}

	return results
}

// extractByPattern is the regex-on-full-text fallback strategy.
func (h *PDFHandler) extractByPattern(rule BillTemplateRule, text string) string {
	hasContext := rule.Pattern != "" && rule.Pattern != rule.ValuePattern

	if hasContext {
		re, err := regexp.Compile("(?i)" + rule.Pattern)
		if err != nil {
			log.Printf("Warning: Invalid regex pattern for field %s: %v", rule.Field, err)
		} else {
			match := re.FindStringSubmatch(text)
			if len(match) > 1 {
				return strings.TrimSpace(match[1])
			}
		}
	}

	if rule.ValuePattern != "" && rule.Prefix != "" {
		contextPrefix := extractLabelContext(rule.Prefix)
		if contextPrefix != "" {
			contextPattern := contextPrefix + `[\s:]*` + rule.ValuePattern
			re, err := regexp.Compile("(?i)" + contextPattern)
			if err == nil {
				match := re.FindStringSubmatch(text)
				if len(match) > 1 {
					log.Printf("Field %s matched with auto-context '%s': %s", rule.Field, contextPrefix, match[1])
					return strings.TrimSpace(match[1])
				}
			}
			log.Printf("Field %s: auto-context '%s' did not match", rule.Field, contextPrefix)
		}
	}

	return ""
}

// extractLabelContext returns the trailing run of alphabetic words from a prefix
// fragment — structural labels that stay constant across bills.
func extractLabelContext(prefixText string) string {
	words := strings.Fields(prefixText)
	var labels []string
	for i := len(words) - 1; i >= 0; i-- {
		w := words[i]
		isLabel := true
		letterCount := 0
		for _, r := range w {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= 0x00C0 && r <= 0x024F) || r == '(' || r == ')' {
				letterCount++
			}
		}
		if letterCount == 0 || float64(letterCount)/float64(len([]rune(w))) < 0.5 {
			isLabel = false
		}
		if isLabel {
			labels = append([]string{regexp.QuoteMeta(w)}, labels...)
		} else if len(labels) > 0 {
			break
		}
	}
	if len(labels) == 0 {
		return ""
	}
	if len(labels) > 3 {
		labels = labels[len(labels)-3:]
	}
	return strings.Join(labels, `\s+`)
}

// notFoundFields lists, sorted, the fields the template maps that came back
// empty: the bill form flags them instead of filling in a guess.
func notFoundFields(extracted ExtractedBillData, template models.BillTemplate) []string {
	var rules []BillTemplateRule
	json.Unmarshal([]byte(template.ExtractionRules), &rules)
	raw, _ := json.Marshal(extracted)
	var filled map[string]any
	json.Unmarshal(raw, &filled)

	var missing []string
	for _, rule := range rules {
		if _, ok := filled[rule.Field]; !ok && !slices.Contains(missing, rule.Field) {
			missing = append(missing, rule.Field)
		}
	}
	slices.Sort(missing)
	return missing
}
