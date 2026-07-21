package awssig

import (
	"strings"
	"testing"
)

// TestCollapseSpacesFieldsEquivalence pins collapseSpaces to its documented
// behavior: the exact result of strings.Join(strings.Fields(v), " "). The
// old byte-loop implementation only folded ASCII spaces and silently dropped
// tabs, newlines, and other whitespace the doc comment promised to fold.
func TestCollapseSpacesFieldsEquivalence(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"single_space", " "},
		{"single_token", "hello"},
		{"two_tokens_single_space", "hello world"},
		{"multiple_consecutive_spaces", "hello    world"},
		{"leading_space", "   hello world"},
		{"trailing_space", "hello world   "},
		{"leading_and_trailing_space", "   hello world   "},
		{"only_spaces", "     "},
		{"single_tab", "\t"},
		{"interior_tab", "hello\tworld"},
		{"mixed_space_tab", "hello \t world\t\t end"},
		{"leading_tab", "\t\thello world"},
		{"trailing_tab", "hello world\t\t"},
		{"newline_only", "\n"},
		{"interior_newline", "hello\nworld"},
		{"carriage_return", "hello\rworld"},
		{"vertical_tab", "hello\vworld"},
		{"form_feed", "hello\fworld"},
		{"mixed_whitespace_runs", "  \t\n hello \r\n world \t  "},
		{"nbsp", "hello\u00a0world"},              // U+00A0 NBSP, unicode.IsSpace = true
		{"unicode_space", "hello\u2000world"},     // U+2000 EN QUAD
		{"unicode_em_space", "hello\u2003world"},  // U+2003 EM SPACE
		{"ideographic_space", "hello\u3000world"}, // U+3000 IDEOGRAPHIC SPACE
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := collapseSpaces(tc.in)
			want := strings.Join(strings.Fields(tc.in), " ")
			if got != want {
				t.Errorf("collapseSpaces(%q) = %q; want %q", tc.in, got, want)
			}
			// Empty input must round-trip to empty.
			if tc.in == "" && got != "" {
				t.Errorf("collapseSpaces(%q) = %q; want empty", tc.in, got)
			}
		})
	}
}
