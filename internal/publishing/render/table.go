package render

import (
	"strings"
	"unicode/utf8"

	"mx_news_bot/internal/publishing/contentmodel"
)

// RenderTable lays a table out in fixed-width columns. maxRows of 0 means every row.
//
// Column widths come from the model but grow to fit the content, so a long rider name pushes the
// column instead of being silently cut.
func RenderTable(table contentmodel.Table, maxRows int) string {
	if len(table.Columns) == 0 {
		return ""
	}

	rows := table.Rows
	if maxRows > 0 && len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	widths := make([]int, len(table.Columns))
	for i, column := range table.Columns {
		widths[i] = max(column.Width, utf8.RuneCountInString(column.Header))
	}
	for _, row := range rows {
		for i := range table.Columns {
			if i < len(row) {
				widths[i] = max(widths[i], utf8.RuneCountInString(row[i]))
			}
		}
	}

	var b strings.Builder
	headers := make([]string, len(table.Columns))
	for i, column := range table.Columns {
		headers[i] = pad(column.Header, widths[i], contentmodel.AlignLeft)
	}
	header := strings.Join(headers, " | ")
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", utf8.RuneCountInString(header)))
	b.WriteString("\n")

	for _, row := range rows {
		cells := make([]string, len(table.Columns))
		for i, column := range table.Columns {
			value := ""
			if i < len(row) {
				value = row[i]
			}
			cells[i] = pad(value, widths[i], column.Align)
		}
		b.WriteString(strings.TrimRight(strings.Join(cells, " | "), " "))
		b.WriteString("\n")
	}

	return b.String()
}

func pad(value string, width int, align contentmodel.Align) string {
	missing := width - utf8.RuneCountInString(value)
	if missing <= 0 {
		return value
	}
	if align == contentmodel.AlignRight {
		return strings.Repeat(" ", missing) + value
	}

	return value + strings.Repeat(" ", missing)
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// RenderItems writes a table as a list of short items, each led by a bold heading: "1. Leclerc —
// Ferrari · 25 pts". A fenced grid wraps badly as soon as it is wider than the screen and Telegram
// gives no way to scroll it; an item is a line that wraps like any other text.
//
// A table whose first column is "#" is a ranking and leads with the position and the name; any
// other table leads with its first cell. The last column is the figure, and a points column is
// said as points.
func RenderItems(table contentmodel.Table) string {
	if len(table.Columns) == 0 {
		return ""
	}

	ranking := strings.TrimSpace(table.Columns[0].Header) == "#"
	unit := ""
	switch strings.ToLower(strings.TrimSpace(table.Columns[len(table.Columns)-1].Header)) {
	case "pts", "points":
		unit = " pts"
	}

	var b strings.Builder
	for _, row := range table.Rows {
		cells := make([]string, 0, len(row))
		for _, cell := range row {
			if trimmed := strings.TrimSpace(cell); trimmed != "" {
				cells = append(cells, trimmed)
			}
		}
		if len(cells) == 0 {
			continue
		}

		heading, rest := cells[0], cells[1:]
		if ranking && len(cells) > 1 {
			heading, rest = cells[0]+". "+cells[1], cells[2:]
		}
		b.WriteString("*" + escapeMarkdown(heading) + "*")
		if len(rest) > 0 {
			if unit != "" && len(row) == len(table.Columns) && strings.TrimSpace(row[len(row)-1]) != "" {
				rest[len(rest)-1] += unit
			}
			b.WriteString(" — " + escapeMarkdown(strings.Join(rest, " · ")))
		}
		b.WriteString("\n")
	}

	return b.String()
}
