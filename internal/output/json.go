package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONPrinter outputs everything as JSON.
type JSONPrinter struct {
	w io.Writer
}

func (p *JSONPrinter) Writer() io.Writer { return p.w }

func (p *JSONPrinter) Table(headers []string, rows [][]string) {
	// Convert table to array of objects using headers as keys
	records := make([]map[string]string, len(rows))
	for i, row := range rows {
		record := make(map[string]string, len(headers))
		for j, h := range headers {
			if j < len(row) {
				record[h] = row[j]
			}
		}
		records[i] = record
	}
	p.JSON(records)
}

func (p *JSONPrinter) Detail(fields []Field) {
	record := make(map[string]string, len(fields))
	for _, f := range fields {
		record[f.Label] = f.Value
	}
	p.JSON(record)
}

func (p *JSONPrinter) JSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(p.w, string(data))
	return nil
}
