package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"
)

// GenerateValidationHeader generates the x-browser-validation value
// by concatenating the API key (or determined from the UA) with the UA string,
// computing its SHA-1 hash, and encoding it in Base64.
func GenerateValidationHeader(ua string) string {
	key := getAPIKeyFromUA(ua)

	// Combine the API key and the user agent
	data := key + ua

	// Compute SHA-1 hash
	hash := sha1.Sum([]byte(data))

	// Encode the hash in Base64
	encoded := base64.StdEncoding.EncodeToString(hash[:])

	return encoded
}

// getAPIKeyFromUA determines the API key based on the user agent string.
func getAPIKeyFromUA(ua string) string {
	if strings.Contains(ua, "Windows NT") {
		return "AIzaSyA2KlwBX3mkFo30om9LUFYQhpqLoa_BNhE"
	} else if strings.Contains(ua, "Mac OS X") {
		return "AIzaSyDr2UxVnv_U85AbhhY8XSHSIavUW0DC-sY"
	} else if strings.Contains(ua, "Linux") {
		return "AIzaSyBqJZh-7pA44blAaAkH6490hUFOwX0KCYM"
	}
	return "" // Fallback, but should not be reached for valid UA strings
}

// Example usage
func main() {
	// Example User Agent for Windows
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"

	// Generate the validation string
	result := GenerateValidationHeader(ua)
	fmt.Println("Generated x-browser-validation:", result)
}
