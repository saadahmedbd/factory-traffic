package domain

import (
	"testing"
)

func TestParseDirection(t *testing.T) {
	tests := []struct {
		input   string
		want    Direction
		wantErr bool
	}{
		{"NORTH", DirectionNorth, false},
		{"SOUTH", DirectionSouth, false},
		{"EAST", DirectionEast, false},
		{"WEST", DirectionWest, false},
		{"UP", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := ParseDirection(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseDirection(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseDirection(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
