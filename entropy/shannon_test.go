package entropy

import "testing"

func isHighEntropy(bits int) bool {
	if bits >= 128 {
		return true
	}

	return false
}

func TestShannon(t *testing.T) {
	var cases = []struct {
		input       string
		highEntropy bool
	}{
		{
			input:       "AAAAAAAAAAA",
			highEntropy: false,
		},
		{
			input:       "0",
			highEntropy: false,
		},
		{
			input:       "false",
			highEntropy: false,
		},
		{
			input:       "668108162888",
			highEntropy: false,
		},
		{
			input:       "0127B6-85D8BD-E21ADE",
			highEntropy: false,
		},
		{
			input:       "ZmYwOTZmNmQyNWFjMWY4ZGY4MDBjNjQ3N2IwOGMxMDY4NTE1ODFjMjhlZmRjZGNmZmE2ZTM2MTQ4NjA2YTFkNDM2MDljZjc1MDFhODgxOTI0NGZmMmNmNmE1NWEyNDEzNmJjMWQxZmVkMmUwZmQ4ZDc5ODdiMjhiNzU4ZWUzYWYK",
			highEntropy: true,
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			bits := Shannon(c.input)
			ent := isHighEntropy(bits)

			t.Logf("entropy is: %d", bits)

			if ent != c.highEntropy {
				t.Errorf("%q was expected to be high entropy: %v got: %v", c.input, c.highEntropy, ent)
			}
		})
	}
}
