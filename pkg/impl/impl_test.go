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

func TestGetMiddleInitials(t *testing.T) {
	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{"kuria", "kuria", "mwangi"}, "k"},
		{[]string{"kuria", "kuria", "ndungu", "mwangi"}, "kn"},
		{[]string{"kuria", "mwangi"}, ""}, // No middle parts
	}

	for _, tc := range tests {
		result := getMiddleInitials(tc.parts)
		if result != tc.expected {
			t.Errorf("getMiddleInitials(%v) = %v; want %v", tc.parts, result, tc.expected)
		}
	}
}

func TestGeneratePermutations_SingleName(t *testing.T) {
	permutations := GeneratePermutations("kuria", "kyneticengynes.com")

	// For a single name, only MinParts: 1 rules should trigger.
	// Based on our rules, that includes "first" and "first1" (2 rules).
	expectedCount := 2
	if len(permutations) != expectedCount {
		t.Fatalf("Expected %d permutations for a single name, got %d", expectedCount, len(permutations))
	}

	// Verify the highest scored item is first
	if permutations[0].Score < permutations[1].Score {
		t.Errorf("Permutations are not sorted by score in descending order")
	}

	// Verify the standard first name email is generated
	found := false
	for _, p := range permutations {
		if p.Email == "kuria@kyneticengynes.com" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Missing expected permutation: kuria@kyneticengynes.com")
	}
}

func TestGeneratePermutations_TwoNames(t *testing.T) {
	permutations := GeneratePermutations("kuria Mwangi", "kyneticengynes.com")

	// Verify standard enterprise format is present
	found := false
	for _, p := range permutations {
		if p.Email == "kuria.mwangi@kyneticengynes.com" {
			found = true
			if p.Pattern != "first.last" {
				t.Errorf("Expected pattern 'first.last', got %s", p.Pattern)
			}
			break
		}
	}
	if !found {
		t.Errorf("Missing expected permutation: kuria.mwangi@kyneticengynes.com")
	}

	// Verify collision fallback is present
	foundCollision := false
	for _, p := range permutations {
		if p.Email == "kuria.mwangi1@kyneticengynes.com" {
			foundCollision = true
			break
		}
	}
	if !foundCollision {
		t.Errorf("Missing expected collision permutation: kuria.mwangi1@kyneticengynes.com")
	}
}

func TestGeneratePermutations_ThreeNames(t *testing.T) {
	permutations := GeneratePermutations("kuria Kuria Mwangi", "kyneticengynes.com")

	// Test a specific 3-part rule like fmlast
	found := false
	for _, p := range permutations {
		if p.Email == "kkmwangi@kyneticengynes.com" {
			found = true
			if p.Pattern != "fmlast" {
				t.Errorf("Expected pattern 'fmlast', got %s", p.Pattern)
			}
			break
		}
	}
	if !found {
		t.Errorf("Missing expected permutation: kkmwangi@kyneticengynes.com")
	}
}

func TestGeneratePermutations_Sanitization(t *testing.T) {
	permutations := GeneratePermutations("  Nga'nga   Wanja  ", "startup.io")

	// It should tokenize as ["nganga", "wanja"]
	found := false
	for _, p := range permutations {
		if p.Email == "nganga.wanja@startup.io" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Failed to properly sanitize and tokenize complex names. Expected nganga.wanja@startup.io")
	}
}

func TestGeneratePermutations_EmptyOrInvalid(t *testing.T) {
	// Name entirely of numbers/symbols
	permutations := GeneratePermutations("12345 !@#$", "startup.io")

	if len(permutations) != 0 {
		t.Errorf("Expected 0 permutations for invalid names, got %d", len(permutations))
	}
}
