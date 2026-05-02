package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  Format
	}{
		{"json", JSON},
		{"JSON", JSON},
		{"plain", Plain},
		{"table", Table},
		{"anything", Table},
		{"", Table},
	}

	for _, tt := range tests {
		got := ParseFormat(tt.input)
		if got != tt.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: JSON, Out: &buf}

	data := map[string]string{"name": "test"}
	if err := p.PrintJSON(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `"name": "test"`) {
		t.Errorf("expected JSON with name field, got %q", got)
	}
}

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: Table, Out: &buf}

	p.PrintTable(
		[]string{"NAME", "LANG"},
		[][]string{
			{"foo/bar", "Go"},
			{"baz/qux", "Rust"},
		},
	)

	got := buf.String()
	if !strings.Contains(got, "NAME") {
		t.Error("expected header in output")
	}
	if !strings.Contains(got, "foo/bar") {
		t.Error("expected row data in output")
	}
}

func TestPrintPlain(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: Plain, Out: &buf}

	p.PrintPlain([]string{"line1", "line2"})

	got := buf.String()
	if !strings.Contains(got, "line1\n") {
		t.Error("expected line1 in output")
	}
	if !strings.Contains(got, "line2\n") {
		t.Error("expected line2 in output")
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"preserves tabs", "col1\tcol2", "col1\tcol2"},
		{"preserves newlines", "line1\nline2", "line1\nline2"},
		{"strips ESC", "normal\x1b[31mred\x1b[0m", "normal[31mred[0m"},
		{"strips BEL", "title\x07", "title"},
		{"strips OSC", "\x1b]0;pwned\x07", "]0;pwned"},
		{"strips null", "a\x00b", "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
