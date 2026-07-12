package sigv4a

import "testing"

func TestRegionSetMatches(t *testing.T) {
	tests := []struct {
		name      string
		regionSet string
		region    string
		want      bool
	}{
		{name: "exact match", regionSet: "us-east-1", region: "us-east-1", want: true},
		{name: "exact mismatch", regionSet: "us-east-1", region: "eu-west-1", want: false},
		{name: "list match", regionSet: "us-east-1,eu-west-1", region: "eu-west-1", want: true},
		{name: "list with spaces", regionSet: "us-east-1, eu-west-1", region: "eu-west-1", want: true},
		{name: "global wildcard", regionSet: "*", region: "anything", want: true},
		{name: "prefix wildcard match", regionSet: "us-west-*", region: "us-west-2", want: true},
		{name: "prefix wildcard mismatch", regionSet: "us-west-*", region: "us-east-1", want: false},
		{name: "empty set", regionSet: "", region: "us-east-1", want: false},
		{name: "empty region", regionSet: "*", region: "", want: false},
		{name: "empty entries skipped", regionSet: ",,us-east-1", region: "us-east-1", want: true},
		{name: "wildcard is prefix only", regionSet: "us-*-1", region: "us-east-1", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := regionSetMatches(tt.regionSet, tt.region); got != tt.want {
				t.Errorf("regionSetMatches(%q, %q) = %v, want %v", tt.regionSet, tt.region, got, tt.want)
			}
		})
	}
}
