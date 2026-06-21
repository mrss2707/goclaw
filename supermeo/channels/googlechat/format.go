package googlechat

import (
	"regexp"
	"strings"
)

// Google Chat markdown subset: https://developers.google.com/chat/format-messages
//
// Supported:
//   - *bold* (single asterisk)
//   - _italic_
//   - ~strikethrough~
//   - `monospace`
//   - ```code block```
//   - <url|link text> (custom link format)
//   - Bullet lists (- item)
//
// NOT supported (stripped):
//   - Headings (###) → converted to *bold*
//   - Tables → ASCII in <pre>
//   - Images → URL text
//   - Horizontal rules → removed

var (
	zeroWidthRE = regexp.MustCompile(`[\x{200B}-\x{200D}\x{FEFF}\x{200E}\x{200F}\x{FE00}-\x{FE0F}]`)
	varSelRE    = regexp.MustCompile(`[\x{E0100}-\x{E01EF}]`)
)

// formatGoogleChat converts standard markdown to Google Chat dialect.
func formatGoogleChat(text string) string {
	// Strip zero-width chars and variation selectors
	text = zeroWidthRE.ReplaceAllString(text, "")
	text = varSelRE.ReplaceAllString(text, "")

	text = convertBold(text)
	text = convertHeadings(text)
	text = convertLinks(text)
	text = convertCodeBlocks(text)
	text = convertTables(text)

	return text
}

// convertBold: **text** → *text* (Google Chat uses single asterisk for bold).
func convertBold(text string) string {
	// Simple state machine: track whether we're inside **...**
	var result strings.Builder
	result.Grow(len(text))
	i := 0
	for i < len(text) {
		if i+1 < len(text) && text[i] == '*' && text[i+1] == '*' {
			// Check for closing **
			start := i + 2
			end := strings.Index(text[start:], "**")
			if end >= 0 {
				inner := text[start : start+end]
				result.WriteString("*")
				result.WriteString(inner)
				result.WriteString("*")
				i = start + end + 2
				continue
			}
		}
		result.WriteByte(text[i])
		i++
	}
	return result.String()
}

// convertHeadings: ### heading → *heading* (bold, no heading support in Google Chat).
func convertHeadings(text string) string {
	re := regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	return re.ReplaceAllString(text, "*$1*")
}

// convertLinks: [text](url) → <url|text> (Google Chat inline link format).
func convertLinks(text string) string {
	re := regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	return re.ReplaceAllString(text, "<$2|$1>")
}

// convertCodeBlocks: protects inline code and converts fenced code blocks.
// Google Chat supports ``` blocks, so we keep them as-is.
func convertCodeBlocks(text string) string {
	// Inline code and fenced blocks are already compatible — no conversion needed.
	// This function exists as a hook for future dialect adjustments.
	return text
}

// convertTables: Google Chat doesn't support markdown tables.
// Convert each table to ASCII art inside <pre> blocks.
func convertTables(text string) string {
	lines := strings.Split(text, "\n")

	// Collect all table ranges (start, end) pairs.
	type tableRange struct{ start, end int }
	var tables []tableRange
	inTable := false
	var start int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		isTableLine := strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|")
		if isTableLine && !inTable {
			start = i
			inTable = true
		} else if !isTableLine && inTable {
			tables = append(tables, tableRange{start, i})
			inTable = false
		}
	}
	if inTable {
		tables = append(tables, tableRange{start, len(lines)})
	}

	if len(tables) == 0 {
		return text
	}

	// Process from last to first so indices stay valid.
	var result []string
	prevEnd := len(lines)
	for i := len(tables) - 1; i >= 0; i-- {
		tr := tables[i]
		// Lines after the table
		after := lines[tr.end:prevEnd]
		// Convert table to ASCII
		ascii := tableToASCII(lines[tr.start:tr.end])
		tableBlock := append([]string{"<pre>"}, ascii...)
		tableBlock = append(tableBlock, "</pre>")

		result = append(tableBlock, result...)
		result = append(after, result...)
		prevEnd = tr.start
	}
	// Prepend lines before first table
	result = append(lines[:prevEnd], result...)

	return strings.Join(result, "\n")
}

// tableToASCII converts pipe-delimited table rows to ASCII art.
func tableToASCII(rows []string) []string {
	if len(rows) == 0 {
		return nil
	}
	// Parse columns
	var cols [][]string
	maxCols := 0
	for _, row := range rows {
		cells := splitTableRow(row)
		if len(cells) > maxCols {
			maxCols = len(cells)
		}
		cols = append(cols, cells)
	}

	// Calculate column widths
	widths := make([]int, maxCols)
	for _, cells := range cols {
		for i, cell := range cells {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	// Min 3 char width
	for i := range widths {
		if widths[i] < 3 {
			widths[i] = 3
		}
	}

	var result []string
	separator := buildSeparator(widths)

	// Skip separator rows (---|---|)
	dataRows := make([][]string, 0)
	for _, cells := range cols {
		isSep := len(cells) > 0
		for _, c := range cells {
			if !isAllDash(c) {
				isSep = false
				break
			}
		}
		if !isSep {
			dataRows = append(dataRows, cells)
		}
	}

	// Build top border
	result = append(result, separator)
	for i, cells := range dataRows {
		result = append(result, buildRow(cells, widths))
		if i == 0 {
			result = append(result, separator)
		}
	}
	result = append(result, separator)

	return result
}

func splitTableRow(row string) []string {
	trimmed := strings.TrimSpace(row)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	cells := strings.Split(trimmed, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func isAllDash(s string) bool {
	cleaned := strings.ReplaceAll(s, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned == ""
}

func buildRow(cells []string, widths []int) string {
	var parts []string
	for i, w := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		parts = append(parts, "| "+padRight(cell, w)+" ")
	}
	return strings.Join(parts, "") + "|"
}

func buildSeparator(widths []int) string {
	var parts []string
	for _, w := range widths {
		parts = append(parts, "+"+strings.Repeat("-", w+2))
	}
	return strings.Join(parts, "") + "+"
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
