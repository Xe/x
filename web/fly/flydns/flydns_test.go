package flydns

import (
	"os"
	"testing"
)

func TestGetApps(t *testing.T) {
	if os.Getenv("DO_FLY") == "" {
		t.Skip()
	}

	// Call the function to be tested
	result, err := GetApps()

	t.Logf("%v", result)

	// Check for the error
	if err != nil {
		t.Errorf("Expected no error, but got an error: %v", err)
	}
}

func TestParseMachine(t *testing.T) {
	input := "1781965b593689 yyz"
	expected := Machine{"1781965b593689", "yyz"}

	result, err := parseMachine(input)
	if err != nil {
		t.Errorf("Expected no error, but got an error: %v", err)
	}

	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
