package h

import "testing"

func TestParse(t *testing.T) {
	if _, err := Parse("h h h"); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := Parse("h h h"); err != nil {
			b.Fatal(err)
		}
	}
}
