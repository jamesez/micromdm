package buildversion

import (
	"fmt"
	"testing"
)

func TestSplitting(t *testing.T) {
	tests := []struct {
		input    BuildVersion
		expected splitVersion
	}{
		{
			input:    BuildVersion("23A1234t"),
			expected: splitVersion{Major: 23, Minor: "A", Build: 1234, Patch: "t"},
		},
		{
			input:    BuildVersion("22B78"),
			expected: splitVersion{Major: 22, Minor: "B", Build: 78, Patch: ""},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.input), func(t *testing.T) {
			t.Parallel()

			s, err := tt.input.split()

			if err != nil {
				t.Fatal(err)
			}

			if s.Major != tt.expected.Major ||
				s.Minor != tt.expected.Minor ||
				s.Build != tt.expected.Build ||
				s.Patch != tt.expected.Patch {
				t.Errorf("failed to split %s, %+v", string(tt.input), s)
			}
		})
	}
}

func TestBuildVersionComparison(t *testing.T) {
	tests := []struct {
		one            BuildVersion
		two            BuildVersion
		oneLessThanTwo bool
	}{
		// same
		{
			one:            "12A34",
			two:            "12A34",
			oneLessThanTwo: false,
		},
		// different major
		{
			one:            "12A34",
			two:            "13A34",
			oneLessThanTwo: true,
		},
		// different minor
		{
			one:            "12B34",
			two:            "12A34",
			oneLessThanTwo: false,
		},
		// different build & size
		{
			one:            "12A34",
			two:            "12A3456",
			oneLessThanTwo: true,
		},
		// different patch
		{
			one:            "12A34i",
			two:            "12A34j",
			oneLessThanTwo: true,
		},
		// missing patch on lesser
		{
			one:            "12A34",
			two:            "12A34i",
			oneLessThanTwo: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		name := fmt.Sprintf("%s < %s", string(tt.one), string(tt.two))
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			want := tt.oneLessThanTwo

			got, err := tt.one.LessThan(tt.two)
			if err != nil {
				t.Fatal(err)
			}

			if got != want {
				t.Errorf("build version %s; got %t, want %t", name, got, want)
			}
		})
	}
}
