package output

import "io"

// Printer handles formatted output for CLI commands.
type Printer interface {
	// Table prints tabular data with headers and rows.
	Table(headers []string, rows [][]string)
	// Detail prints a single record as key-value pairs.
	Detail(fields []Field)
	// JSON prints the value as JSON (used in --json mode for both Table and Detail).
	JSON(v any) error
	// Writer returns the underlying writer.
	Writer() io.Writer
}

// Field is a labeled value for detail views.
type Field struct {
	Label string
	Value string
}

// New creates a Printer based on whether JSON mode is enabled.
func New(w io.Writer, jsonMode bool) Printer {
	if jsonMode {
		return &JSONPrinter{w: w}
	}
	return &TablePrinter{w: w}
}
