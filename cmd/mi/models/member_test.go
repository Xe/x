package models

import "testing"

func TestMemberMatchesName(t *testing.T) {
	tests := []struct {
		name    string
		member  Member
		input   string
		want    bool
	}{
		{
			name:   "exact canonical name",
			member: Member{Name: "Cadey", Aliases: "xe,cadey-ratio"},
			input:  "Cadey",
			want:   true,
		},
		{
			name:   "canonical name case insensitive",
			member: Member{Name: "Cadey", Aliases: "xe,cadey-ratio"},
			input:  "cadey",
			want:   true,
		},
		{
			name:   "alias match",
			member: Member{Name: "Cadey", Aliases: "xe,cadey-ratio"},
			input:  "xe",
			want:   true,
		},
		{
			name:   "alias case insensitive",
			member: Member{Name: "Cadey", Aliases: "xe,cadey-ratio"},
			input:  "Xe",
			want:   true,
		},
		{
			name:   "alias with spaces in list",
			member: Member{Name: "Cadey", Aliases: "xe, cadey-ratio"},
			input:  "cadey-ratio",
			want:   true,
		},
		{
			name:   "no match",
			member: Member{Name: "Cadey", Aliases: "xe,cadey-ratio"},
			input:  "Nicole",
			want:   false,
		},
		{
			name:   "empty aliases",
			member: Member{Name: "Cadey"},
			input:  "Cadey",
			want:   true,
		},
		{
			name:   "empty aliases no match",
			member: Member{Name: "Cadey"},
			input:  "xe",
			want:   false,
		},
		{
			name:   "empty input",
			member: Member{Name: "Cadey", Aliases: "xe"},
			input:  "",
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.member.MatchesName(tc.input)
			if got != tc.want {
				t.Errorf("Member{Name: %q, Aliases: %q}.MatchesName(%q) = %v, want %v",
					tc.member.Name, tc.member.Aliases, tc.input, got, tc.want)
			}
		})
	}
}
