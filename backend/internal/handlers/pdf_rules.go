package handlers

import (
	"encoding/json"
	"log"
	"regexp"
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
		case "service_code":
			extracted.ServiceCode = value
		case "customer_code":
			extracted.CustomerCode = value
		case "conversion_coefficient":
			coeff := h.parseAmount(value)
			extracted.ConversionCoefficient = &coeff
		case "provider_reading":
			reading := h.parseAmount(value)
			extracted.ProviderReading = &reading
		case "reading_type":
			lower := strings.ToLower(value)
			if strings.Contains(lower, "stim") || strings.Contains(lower, "presunt") {
				extracted.ReadingType = "estimated"
			} else {
				extracted.ReadingType = "actual"
			}
		case "estimated_date":
			parsed := h.parseDate(value, rule.Format)
			extracted.EstimatedDate = &parsed
		case "estimated_reading":
			val := h.parseAmount(value)
			extracted.EstimatedReading = &val
		case "estimated_consumption":
			val := h.parseAmount(value)
			extracted.EstimatedConsumption = &val
		case "previous_estimated_consumption":
			val := h.parseAmount(value)
			extracted.PreviousEstimatedConsumption = &val
		case "communication_text":
			extracted.CommunicationText = value
		}
	}

	return extracted
}

// extractValueWithRuleAndWords picks the best extraction strategy per rule.
// Priority: multi-line anchor → global search → anchor-based → position → pattern.
func (h *PDFHandler) extractValueWithRuleAndWords(rule BillTemplateRule, text string, words []WordInfo) string {
	if rule.MultiLine && rule.AnchorText != "" && len(words) > 0 {
		value := h.extractMultiLineByAnchor(rule, words)
		if value != "" {
			log.Printf("Field %s matched by multi-line anchor '%s': %s", rule.Field, rule.AnchorText, value[:min(len(value), 80)])
			return value
		}
		log.Printf("Field %s: multi-line anchor match failed, trying other strategies", rule.Field)
	}

	if rule.GlobalSearch && rule.ValuePattern != "" && len(words) > 0 {
		value := h.extractByGlobalSearch(rule, words)
		if value != "" {
			log.Printf("Field %s matched by global search: %s", rule.Field, value)
			return value
		}
		log.Printf("Field %s: global search failed, trying next strategy", rule.Field)
	}

	if rule.AnchorText != "" && len(words) > 0 {
		value := h.extractByAnchor(rule, words)
		if value != "" {
			log.Printf("Field %s matched by anchor '%s': %s", rule.Field, rule.AnchorText, value)
			return value
		}
		log.Printf("Field %s: anchor-based match failed, trying position", rule.Field)
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

// extractByAnchor searches for the rule's anchor text, then picks the best value
// in the region-of-interest relative to each occurrence. A graduated historical-
// position bonus prefers the value nearest the user's original click when the
// same anchor text appears multiple times (e.g., summary table vs pie legend).
func (h *PDFHandler) extractByAnchor(rule BillTemplateRule, words []WordInfo) string {
	anchors := findAnchorOccurrences(rule.AnchorText, words)
	if len(anchors) == 0 {
		return ""
	}

	if rule.ValuePattern == "" {
		return ""
	}
	valueRe, err := regexp.Compile("(?i)^" + rule.ValuePattern + "$")
	if err != nil {
		log.Printf("Invalid value pattern for anchor search field %s: %v", rule.Field, err)
		return ""
	}

	type candidate struct {
		text     string
		distance float64
		word     WordInfo
	}

	var bestCandidate *candidate

	for _, anchor := range anchors {
		roiWords := searchROI(anchor, rule.AnchorDirection, words)

		for _, w := range roiWords {
			if valueRe.MatchString(w.Text) {
				dx := w.X - anchor.LastWord.X
				dy := w.Y - anchor.LastWord.Y
				dist := dx*dx + dy*dy

				if rule.X > 0 && rule.Y > 0 {
					histDx := w.X - rule.X
					histDy := w.Y - rule.Y
					histDist := histDx*histDx + histDy*histDy
					if histDist < 30*30 {
						dist *= 0.01
					} else if histDist < 80*80 {
						dist *= 0.05
					} else if histDist < 150*150 {
						dist *= 0.2
					}
					log.Printf("  Field %s: candidate '%s' at (%.1f,%.1f) histDist=%.0f anchorDist=%.0f finalScore=%.1f",
						rule.Field, w.Text, w.X, w.Y, histDist, dx*dx+dy*dy, dist)
				}

				if bestCandidate == nil || dist < bestCandidate.distance {
					bestCandidate = &candidate{text: w.Text, distance: dist, word: w}
				}
			}
		}
	}

	if bestCandidate != nil {
		return bestCandidate.text
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
// multi-word) anchor across all pages.
func findAnchorOccurrences(anchorText string, words []WordInfo) []AnchorMatch {
	anchorWords := strings.Fields(strings.ToLower(anchorText))
	if len(anchorWords) == 0 {
		return nil
	}

	var matches []AnchorMatch

	for i := 0; i <= len(words)-len(anchorWords); i++ {
		matched := true
		var matchedWords []WordInfo

		for j, aw := range anchorWords {
			w := words[i+j]
			wText := strings.ToLower(w.Text)

			if j > 0 && w.Page != words[i].Page {
				matched = false
				break
			}

			if j > 0 && abs(w.Y-words[i+j-1].Y) > 10 {
				matched = false
				break
			}

			wClean := strings.TrimRight(wText, ":.,;")
			awClean := strings.TrimRight(aw, ":.,;")
			if wClean != awClean {
				matched = false
				break
			}

			matchedWords = append(matchedWords, w)
		}

		if matched && len(matchedWords) == len(anchorWords) {
			matches = append(matches, AnchorMatch{
				Words:    matchedWords,
				LastWord: matchedWords[len(matchedWords)-1],
				Page:     matchedWords[0].Page,
			})
		}
	}

	return matches
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
		for _, w := range allWords {
			if w.Page != ref.Page {
				continue
			}
			if abs(w.Y-ref.Y) < 10 && w.X > ref.X+ref.Width-5 && w.X < ref.X+300 {
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

// extractByGlobalSearch scans every word for a value matching the pattern — used
// for unique identifiers like POD/PDR codes.
func (h *PDFHandler) extractByGlobalSearch(rule BillTemplateRule, words []WordInfo) string {
	if rule.ValuePattern == "" {
		return ""
	}

	valueRe, err := regexp.Compile("(?i)^" + rule.ValuePattern + "$")
	if err != nil {
		log.Printf("Invalid value pattern for global search field %s: %v", rule.Field, err)
		return ""
	}

	for _, w := range words {
		if valueRe.MatchString(w.Text) {
			return w.Text
		}
	}

	return ""
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

	if !hasContext && rule.Pattern != "" {
		re, err := regexp.Compile("(?i)" + rule.Pattern)
		if err != nil {
			log.Printf("Warning: Invalid regex pattern for field %s: %v", rule.Field, err)
		} else {
			match := re.FindStringSubmatch(text)
			if len(match) > 1 {
				log.Printf("Field %s matched with bare pattern (no context): %s", rule.Field, match[1])
				return strings.TrimSpace(match[1])
			}
		}
	}

	if rule.ValuePattern != "" && rule.ValuePattern != rule.Pattern {
		re, err := regexp.Compile("(?i)" + rule.ValuePattern)
		if err != nil {
			log.Printf("Warning: Invalid value pattern for field %s: %v", rule.Field, err)
		} else {
			match := re.FindStringSubmatch(text)
			if len(match) > 1 {
				log.Printf("Field %s matched with fallback value_pattern", rule.Field)
				return strings.TrimSpace(match[1])
			} else if len(match) > 0 {
				return strings.TrimSpace(match[0])
			}
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

// getTemplateFieldSet returns the set of field names a template defines rules for.
func (h *PDFHandler) getTemplateFieldSet(template models.BillTemplate) map[string]bool {
	fields := make(map[string]bool)
	var rules []BillTemplateRule
	json.Unmarshal([]byte(template.ExtractionRules), &rules)
	for _, rule := range rules {
		fields[rule.Field] = true
	}
	fieldNames := make([]string, 0, len(fields))
	for f := range fields {
		fieldNames = append(fieldNames, f)
	}
	log.Printf("Template '%s' defines rules for %d fields: %v", template.Name, len(fields), fieldNames)
	return fields
}

// mergeExtracted fills primary's missing fields from fallback, but skips fields
// the template defines rules for — wrong values are worse than missing values.
func (h *PDFHandler) mergeExtracted(primary, fallback ExtractedBillData, templateFields map[string]bool) ExtractedBillData {
	if primary.BillNumber == "" && !templateFields["bill_number"] {
		primary.BillNumber = fallback.BillNumber
	}
	if primary.IssueDate == "" && !templateFields["issue_date"] {
		primary.IssueDate = fallback.IssueDate
	}
	if primary.PeriodStart == "" && !templateFields["period_start"] {
		primary.PeriodStart = fallback.PeriodStart
	}
	if primary.PeriodEnd == "" && !templateFields["period_end"] {
		primary.PeriodEnd = fallback.PeriodEnd
	}
	if primary.DueDate == "" && !templateFields["due_date"] {
		primary.DueDate = fallback.DueDate
	}
	if primary.AmountTotal == 0 && !templateFields["amount_total"] {
		primary.AmountTotal = fallback.AmountTotal
	}
	if primary.ConsumptionTotal == 0 && !templateFields["consumption_total"] {
		primary.ConsumptionTotal = fallback.ConsumptionTotal
	}
	if primary.Provider == "" {
		primary.Provider = fallback.Provider
	}
	if primary.ServiceCode == "" && !templateFields["service_code"] {
		primary.ServiceCode = fallback.ServiceCode
	}
	if primary.CustomerCode == "" && !templateFields["customer_code"] {
		primary.CustomerCode = fallback.CustomerCode
	}
	if primary.ProviderReadingDate == "" && !templateFields["provider_reading_date"] {
		primary.ProviderReadingDate = fallback.ProviderReadingDate
	}
	if primary.ReadingType == "" && !templateFields["reading_type"] {
		primary.ReadingType = fallback.ReadingType
	}
	if primary.ProviderReadingF1 == nil && !templateFields["provider_reading_f1"] {
		primary.ProviderReadingF1 = fallback.ProviderReadingF1
	}
	if primary.ProviderReadingF2 == nil && !templateFields["provider_reading_f2"] {
		primary.ProviderReadingF2 = fallback.ProviderReadingF2
	}
	if primary.ProviderReadingF3 == nil && !templateFields["provider_reading_f3"] {
		primary.ProviderReadingF3 = fallback.ProviderReadingF3
	}
	if primary.ProviderReading == nil && !templateFields["provider_reading"] {
		primary.ProviderReading = fallback.ProviderReading
	}
	if primary.ConversionCoefficient == nil && !templateFields["conversion_coefficient"] {
		primary.ConversionCoefficient = fallback.ConversionCoefficient
	}
	if primary.EstimatedDate == nil && !templateFields["estimated_date"] {
		primary.EstimatedDate = fallback.EstimatedDate
	}
	if primary.EstimatedReading == nil && !templateFields["estimated_reading"] {
		primary.EstimatedReading = fallback.EstimatedReading
	}
	if primary.EstimatedConsumption == nil && !templateFields["estimated_consumption"] {
		primary.EstimatedConsumption = fallback.EstimatedConsumption
	}
	if primary.PreviousEstimatedConsumption == nil && !templateFields["previous_estimated_consumption"] {
		primary.PreviousEstimatedConsumption = fallback.PreviousEstimatedConsumption
	}
	return primary
}
