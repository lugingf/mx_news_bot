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
