package flydns

import (
	"testing"
)

func TestGetApps(t *testing.T) {
	// Call the function to be tested
	result, err := GetApps()

	t.Logf("%v", result)

	// Check for the error
	if err != nil {
		t.Errorf("Expected no error, but got an error: %v", err)
	}
}
