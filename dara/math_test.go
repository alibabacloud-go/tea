package dara

import (
	"testing"
)

// TestRandom tests the Random function to ensure it returns a value in the range [0.0, 1.0)
func TestRandom(t *testing.T) {
	for i := 0; i < 100; i++ {
		val := Random()
		if val < 0.0 || val >= 1.0 {
			t.Errorf("Random() = %v, want [0.0, 1.0)", val)
		}
	}
}

// TestFloor tests the Floor function with various numeric inputs
func TestFloor(t *testing.T) {
	tests := []struct {
		input    Number
		expected int
	}{
		{3.7, 3},
		{-3.7, -4},
		{0.9, 0},
		{0.0, 0},
		{-0.9, -1},
		{int64(3), 3},
		{int32(-3), -3},
	}

	for _, tt := range tests {
		result := Floor(tt.input)
		if result != tt.expected {
			t.Errorf("Floor(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestRound tests the Round function with various numeric inputs
func TestRound(t *testing.T) {
	tests := []struct {
		input    Number
		expected int
	}{
		{3.7, 4},
		{-3.7, -4},
		{2.5, 3},
		{2.4, 2},
		{-2.5, -3},
		{0.0, 0},
		{int64(4), 4},
		{int32(-4), -4},
	}

	for _, tt := range tests {
		result := Round(tt.input)
		if result != tt.expected {
			t.Errorf("Round(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
