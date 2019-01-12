package tokipona

// pu li wile e ni.

import "go4.org/legal"

const tokiPonaLicense = `This creative work by Christine Dodrill is based on the official Toki Pona book and website: http://tokipona.org`

func init() {
	legal.RegisterLicense(tokiPonaLicense)
}
