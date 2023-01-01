package h

import "testing"

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := Parse("h h h"); err != nil {
			b.Fatal(err)
		}
	}
}
