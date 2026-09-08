package impl

import (
	"testing"
)

func TestGetPlausibility(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{0.95, "Very High"},
		{0.85, "Very High"},
		{0.75, "High"},
		{0.70, "High"},
		{0.60, "Medium"},
		{0.45, "Medium"},
		{0.30, "Low"},
		{0.05, "Low"},
	}

	for _, tc := range tests {
		result := getPlausibility(tc.score)
		if result != tc.expected {
			t.Errorf("getPlausibility(%v) = %v; want %v", tc.score, result, tc.expected)
		}
	}
}

func containsEmail(perms []Permutation, email string) bool {
	for _, p := range perms {
		if p.Email == email {
			return true
		}
	}
	return false
}

func TestGeneratePermutations_SingleName(t *testing.T) {
	permutations := GeneratePermutations("kuria", "kyneticengynes.com")

	if len(permutations) == 0 {
		t.Fatalf("Expected permutations for a single name, got 0")
	}

	if !containsEmail(permutations, "kuria@kyneticengynes.com") {
		t.Errorf("Missing expected permutation: kuria@kyneticengynes.com")
	}

	if !containsEmail(permutations, "kuria1@kyneticengynes.com") {
		t.Errorf("Missing expected collision permutation: kuria1@kyneticengynes.com")
	}
}

func TestGeneratePermutations_TwoNames(t *testing.T) {
	permutations := GeneratePermutations("kuria Mwangi", "kyneticengynes.com")

	expectedEmails := []string{
		"kuria.mwangi@kyneticengynes.com",
		"kmwangi@kyneticengynes.com",
		"kuriam@kyneticengynes.com",
		"kuria_mwangi@kyneticengynes.com",
	}

	for _, expected := range expectedEmails {
		if !containsEmail(permutations, expected) {
			t.Errorf("Missing expected permutation: %s", expected)
		}
	}
}

func TestGeneratePermutations_ThreeNames(t *testing.T) {
	permutations := GeneratePermutations("kuria Ndungu Mwangi", "kyneticengynes.com")

	expectedEmails := []string{
		"kuria.ndungu.mwangi@kyneticengynes.com",
		"k.n.mwangi@kyneticengynes.com",
		"kuria.mwangi@kyneticengynes.com",
		"kndungumwangi@kyneticengynes.com",
	}

	for _, expected := range expectedEmails {
		if !containsEmail(permutations, expected) {
			t.Errorf("Missing expected permutation: %s", expected)
		}
	}
}

func TestGeneratePermutations_Sanitization(t *testing.T) {
	permutations := GeneratePermutations("  Mary-Jane   O'Connor  ", "startup.io")

	// It should tokenize as ["mary", "jane", "oconnor"]
	if !containsEmail(permutations, "mary.jane.oconnor@startup.io") {
		t.Errorf("Failed to properly sanitize and tokenize complex names. Expected mary.jane.oconnor@startup.io")
	}
}

func TestGeneratePermutations_EmptyOrInvalid(t *testing.T) {
	permutations := GeneratePermutations("12345 !@#$", "startup.io")
	if len(permutations) != 0 {
		t.Errorf("Expected 0 permutations for invalid names, got %d", len(permutations))
	}
}
