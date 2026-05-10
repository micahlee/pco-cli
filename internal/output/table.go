package output

import (
	"fmt"
	"io"
	"strings"
)

// TablePrinter outputs human-readable tables and detail views.
type TablePrinter struct {
	w io.Writer
}

func (p *TablePrinter) Writer() io.Writer { return p.w }

func (p *TablePrinter) Table(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	fmtParts := make([]string, len(headers))
	for i, w := range widths {
		if i == len(headers)-1 {
			fmtParts[i] = "%s" // last column: no padding
		} else {
			fmtParts[i] = fmt.Sprintf("%%-%ds", w)
		}
	}

	headerVals := make([]any, len(headers))
	for i, h := range headers {
		headerVals[i] = h
	}
	fmt.Fprintf(p.w, strings.Join(fmtParts, "  ")+"\n", headerVals...)

	// Print separator
	sepParts := make([]string, len(widths))
	for i, w := range widths {
		sepParts[i] = strings.Repeat("-", w)
	}
	fmt.Fprintln(p.w, strings.Join(sepParts, "  "))

	// Print rows
	for _, row := range rows {
		vals := make([]any, len(headers))
		for i := range headers {
			if i < len(row) {
				vals[i] = row[i]
			} else {
				vals[i] = ""
			}
		}
		fmt.Fprintf(p.w, strings.Join(fmtParts, "  ")+"\n", vals...)
	}
}

func (p *TablePrinter) Detail(fields []Field) {
	maxLabel := 0
	for _, f := range fields {
		if len(f.Label) > maxLabel {
			maxLabel = len(f.Label)
		}
	}
	for _, f := range fields {
		fmt.Fprintf(p.w, "%-*s  %s\n", maxLabel+1, f.Label+":", f.Value)
	}
}

func (p *TablePrinter) JSON(v any) error {
	// Table printer ignores JSON calls — this shouldn't be called
	return nil
}
