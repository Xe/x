package namcu

// Digits
const (
	Zero  = "no"
	One   = "pa"
	Two   = "re"
	Three = "ci"
	Four  = "vo"
	Five  = "mu"
	Six   = "xa"
	Seven = "ze"
	Eight = "bi"
	Nine  = "so"
)

func Lerfu(i int) string {
	return lojbanDigit(i)
}

func lojbanDigit(i int) string {
	switch i {
	case 0:
		return Zero
	case 1:
		return One
	case 2:
		return Two
	case 3:
		return Three
	case 4:
		return Four
	case 5:
		return Five
	case 6:
		return Six
	case 7:
		return Seven
	case 8:
		return Eight
	case 9:
		return Nine
	default:
		rem := i % 10
		iter := i / 10

		return lojbanDigit(iter) + lojbanDigit(rem)
	}
}
