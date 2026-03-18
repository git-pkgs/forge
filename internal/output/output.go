package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Format specifies how to render output.
type Format string

const (
	Table Format = "table"
	JSON  Format = "json"
	Plain Format = "plain"

	tabPadding = 2
)

// ParseFormat converts a string flag value to a Format.
func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return JSON
	case "plain":
		return Plain
	default:
		return Table
	}
}

// Printer handles formatted output.
type Printer struct {
	Format Format
	Out    io.Writer
}

// PrintJSON writes v as indented JSON.
func (p *Printer) PrintJSON(v any) error {
	enc := json.NewEncoder(p.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintTable writes rows as a tab-aligned table. headers are the column names;
// rows is a slice of slices where each inner slice is one row.
func (p *Printer) PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(p.Out, 0, 0, tabPadding, ' ', 0)
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		_, _ = fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
}

// PrintPlain writes each value on its own line.
func (p *Printer) PrintPlain(lines []string) {
	for _, line := range lines {
		_, _ = fmt.Fprintln(p.Out, line)
	}
}
