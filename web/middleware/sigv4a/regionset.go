package sigv4a

import "strings"

// regionSetMatches reports whether region is covered by a SigV4A
// X-Amz-Region-Set value: a comma-separated list of region names where an
// entry may be "*" or carry a trailing "*" for prefix matching ("us-west-*").
// A "*" anywhere but the end of an entry is not a wildcard; entries match
// case-sensitively, matching how the verifier pins Region elsewhere.
func regionSetMatches(regionSet, region string) bool {
	if region == "" {
		return false
	}
	for entry := range strings.SplitSeq(regionSet, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if prefix, ok := strings.CutSuffix(entry, "*"); ok {
			if strings.HasPrefix(region, prefix) {
				return true
			}
			continue
		}
		if entry == region {
			return true
		}
	}
	return false
}
