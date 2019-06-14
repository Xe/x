package entropy

import "math"

// Shannon measures the Shannon entropy of a string.
// See http://bearcave.com/misl/misl_tech/wavelets/compression/shannon.html for the algorithmic explanation.
func Shannon(value string) (bits int) {
	frq := make(map[rune]float64)

	//get frequency of characters
	for _, i := range value {
		frq[i]++
	}

	var sum float64

	for _, v := range frq {
		f := v / float64(len(value))
		sum += f * math.Log2(f)
	}

	bits = int(math.Ceil(sum*-1)) * len(value)
	return
}
