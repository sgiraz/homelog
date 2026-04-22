package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// WordInfo represents a word with its position and bounding box.
type WordInfo struct {
	Text      string  `json:"text"`
	LineIndex int     `json:"lineIndex"`
	WordIndex int     `json:"wordIndex"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Page      int     `json:"page"`
}

// ExtractedTextWithPositions holds full-text + per-word positions for a PDF.
type ExtractedTextWithPositions struct {
	RawText   string     `json:"raw_text"`
	Words     []WordInfo `json:"words"`
	HasBBox   bool       `json:"has_bbox"`
	PageCount int        `json:"page_count"`
}

// win1252ToUTF8 maps Windows-1252 specific bytes (0x80-0x9F range) to their
// correct UTF-8 equivalents. These bytes differ from Latin-1/ISO-8859-1.
var win1252ToUTF8 = map[byte][]byte{
	0x80: []byte("€"),
	0x82: []byte("‚"),
	0x83: []byte("ƒ"),
	0x84: []byte("„"),
	0x85: []byte("…"),
	0x86: []byte("†"),
	0x87: []byte("‡"),
	0x88: []byte("ˆ"),
	0x89: []byte("‰"),
	0x8A: []byte("Š"),
	0x8B: []byte("‹"),
	0x8C: []byte("Œ"),
	0x8E: []byte("Ž"),
	0x91: []byte("\u2018"),
	0x92: []byte("\u2019"),
	0x93: []byte("\u201C"),
	0x94: []byte("\u201D"),
	0x95: []byte("•"),
	0x96: []byte("\u2013"),
	0x97: []byte("\u2014"),
	0x98: []byte("˜"),
	0x99: []byte("™"),
	0x9A: []byte("š"),
	0x9B: []byte("›"),
	0x9C: []byte("œ"),
	0x9E: []byte("ž"),
	0x9F: []byte("Ÿ"),
}

// normalizeLatin1ToUTF8 converts bytes that may contain Latin-1 or Windows-1252
// characters to valid UTF-8. pdftotext on Windows often emits Latin-1
// (° as 0xB0) or Windows-1252 (€ as 0x80) instead of proper UTF-8.
func normalizeLatin1ToUTF8(data []byte) string {
	var result []byte
	for i := 0; i < len(data); i++ {
		b := data[i]
		if b < 0x80 {
			result = append(result, b)
		} else if b >= 0xF0 && b <= 0xF7 && i+3 < len(data) &&
			data[i+1] >= 0x80 && data[i+1] < 0xC0 &&
			data[i+2] >= 0x80 && data[i+2] < 0xC0 &&
			data[i+3] >= 0x80 && data[i+3] < 0xC0 {
			result = append(result, b, data[i+1], data[i+2], data[i+3])
			i += 3
		} else if b >= 0xE0 && b <= 0xEF && i+2 < len(data) &&
			data[i+1] >= 0x80 && data[i+1] < 0xC0 &&
			data[i+2] >= 0x80 && data[i+2] < 0xC0 {
			result = append(result, b, data[i+1], data[i+2])
			i += 2
		} else if b >= 0xC0 && b <= 0xDF && i+1 < len(data) &&
			data[i+1] >= 0x80 && data[i+1] < 0xC0 {
			result = append(result, b, data[i+1])
			i++
		} else if b >= 0x80 && b < 0xC0 {
			if mapped, ok := win1252ToUTF8[b]; ok {
				result = append(result, mapped...)
			} else {
				result = append(result, 0xC2, b)
			}
		} else if b >= 0xC0 {
			result = append(result, 0xC3, b-0x40)
		} else {
			result = append(result, b)
		}
	}
	return string(result)
}

// extractTextFromPDF uses pdftotext (poppler-utils) for flat text extraction.
// Every external invocation is bounded by pdfProcessTimeout so a pathological
// PDF cannot stall the request indefinitely.
func (h *PDFHandler) extractTextFromPDF(pdfPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pdfProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", pdfPath, "-")
	output, err := cmd.Output()
	if err == nil {
		return normalizeLatin1ToUTF8(output), nil
	}

	// Windows fallback: read the PDF directly for very basic extraction.
	file, err := os.Open(pdfPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	text := extractBasicPDFText(string(content))
	if text != "" {
		return text, nil
	}

	return "", fmt.Errorf("could not extract text from PDF")
}

// extractTextWithPositions extracts text + per-word bounding boxes using
// `pdftotext -bbox`, falling back to `-layout` (character positions) if -bbox
// fails. Both invocations are timeout-bounded.
func (h *PDFHandler) extractTextWithPositions(pdfPath string) (*ExtractedTextWithPositions, error) {
	result := &ExtractedTextWithPositions{
		Words:   []WordInfo{},
		HasBBox: false,
	}

	bboxCtx, cancelBbox := context.WithTimeout(context.Background(), pdfProcessTimeout)
	defer cancelBbox()
	cmd := exec.CommandContext(bboxCtx, "pdftotext", "-bbox", pdfPath, "-")
	output, err := cmd.Output()
	if err == nil {
		words, pageCount, parseErr := parsePdftextBboxOutput(normalizeLatin1ToUTF8(output))
		if parseErr == nil && len(words) > 0 {
			result.Words = words
			result.HasBBox = true
			result.PageCount = pageCount
			rawCtx, cancelRaw := context.WithTimeout(context.Background(), pdfProcessTimeout)
			defer cancelRaw()
			rawCmd := exec.CommandContext(rawCtx, "pdftotext", "-layout", pdfPath, "-")
			rawOutput, _ := rawCmd.Output()
			result.RawText = normalizeLatin1ToUTF8(rawOutput)
			return result, nil
		}
		log.Printf("Warning: Could not parse -bbox output, falling back to layout: %v", parseErr)
	}

	layoutCtx, cancelLayout := context.WithTimeout(context.Background(), pdfProcessTimeout)
	defer cancelLayout()
	cmd = exec.CommandContext(layoutCtx, "pdftotext", "-layout", pdfPath, "-")
	output, err = cmd.Output()
	if err != nil {
		file, fileErr := os.Open(pdfPath)
		if fileErr != nil {
			return nil, fileErr
		}
		defer file.Close()

		content, readErr := io.ReadAll(file)
		if readErr != nil {
			return nil, readErr
		}

		text := extractBasicPDFText(string(content))
		if text == "" {
			return nil, fmt.Errorf("could not extract text from PDF")
		}
		result.RawText = text
		result.Words = extractWordsFromLayoutText(text)
		return result, nil
	}

	normalized := normalizeLatin1ToUTF8(output)
	result.RawText = normalized
	result.Words = extractWordsFromLayoutText(normalized)
	return result, nil
}

// parsePdftextBboxOutput parses the HTML/XML output from `pdftotext -bbox`.
func parsePdftextBboxOutput(htmlOutput string) ([]WordInfo, int, error) {
	var words []WordInfo

	pagePattern := regexp.MustCompile(`<page[^>]*>([\s\S]*?)</page>`)
	wordPattern := regexp.MustCompile(`<word\s+xMin="([^"]+)"\s+yMin="([^"]+)"\s+xMax="([^"]+)"\s+yMax="([^"]+)"[^>]*>([^<]+)</word>`)

	pageMatches := pagePattern.FindAllStringSubmatch(htmlOutput, -1)
	pageCount := len(pageMatches)

	if pageCount == 0 {
		return nil, 0, fmt.Errorf("no pages found in bbox output")
	}

	for pageIndex, pageMatch := range pageMatches {
		if len(pageMatch) < 2 {
			continue
		}

		pageContent := pageMatch[1]
		lineIndex := 0
		wordIndexInLine := 0
		lastY := float64(-1)

		wordMatches := wordPattern.FindAllStringSubmatch(pageContent, -1)
		for _, match := range wordMatches {
			if len(match) < 6 {
				continue
			}

			xMin, _ := strconv.ParseFloat(match[1], 64)
			yMin, _ := strconv.ParseFloat(match[2], 64)
			xMax, _ := strconv.ParseFloat(match[3], 64)
			yMax, _ := strconv.ParseFloat(match[4], 64)
			text := strings.TrimSpace(match[5])

			if text == "" {
				continue
			}

			if lastY >= 0 && (yMin-lastY > 5 || yMin < lastY-5) {
				lineIndex++
				wordIndexInLine = 0
			}
			lastY = yMin

			words = append(words, WordInfo{
				Text:      text,
				LineIndex: lineIndex,
				WordIndex: wordIndexInLine,
				X:         xMin,
				Y:         yMin,
				Width:     xMax - xMin,
				Height:    yMax - yMin,
				Page:      pageIndex,
			})
			wordIndexInLine++
		}
	}

	if len(words) == 0 {
		return nil, 0, fmt.Errorf("no words found in bbox output")
	}

	return words, pageCount, nil
}

// extractWordsFromLayoutText extracts words with relative (char-position) coordinates
// from `pdftotext -layout` output. Used as a fallback when -bbox isn't available.
func extractWordsFromLayoutText(text string) []WordInfo {
	var words []WordInfo
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		wordPattern := regexp.MustCompile(`\S+`)
		matches := wordPattern.FindAllStringIndex(line, -1)

		for wordIdx, match := range matches {
			wordText := line[match[0]:match[1]]
			if wordText == "" {
				continue
			}

			words = append(words, WordInfo{
				Text:      wordText,
				LineIndex: lineIdx,
				WordIndex: wordIdx,
				X:         float64(match[0]),
				Y:         float64(lineIdx),
				Width:     float64(match[1] - match[0]),
				Height:    1,
				Page:      0,
			})
		}
	}

	return words
}

// extractBasicPDFText does a minimal BT/ET + Tj parse for simple text PDFs on
// Windows hosts without poppler-utils. Good enough for plain text, not images.
func extractBasicPDFText(content string) string {
	var texts []string

	btPattern := regexp.MustCompile(`BT\s*(.*?)\s*ET`)
	matches := btPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			tjPattern := regexp.MustCompile(`\((.*?)\)\s*Tj`)
			tjMatches := tjPattern.FindAllStringSubmatch(match[1], -1)
			for _, tj := range tjMatches {
				if len(tj) > 1 {
					texts = append(texts, tj[1])
				}
			}
		}
	}

	return strings.Join(texts, " ")
}

// getImageDimensions reads PNG width/height from the IHDR chunk.
func getImageDimensions(path string) (int, int) {
	file, err := os.Open(path)
	if err != nil {
		return 800, 1100
	}
	defer file.Close()

	header := make([]byte, 24)
	if _, err := file.Read(header); err != nil {
		return 800, 1100
	}

	if len(header) >= 24 {
		width := int(header[16])<<24 | int(header[17])<<16 | int(header[18])<<8 | int(header[19])
		height := int(header[20])<<24 | int(header[21])<<16 | int(header[22])<<8 | int(header[23])
		if width > 0 && height > 0 {
			return width, height
		}
	}

	return 800, 1100
}
