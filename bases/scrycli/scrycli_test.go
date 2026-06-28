package scrycli

import (
	"strings"
	"testing"
)

func TestValidateOptionsAcceptsDocumentedValues(t *testing.T) {
	if err := validateOptions(16, 8, "auto", "never"); err != nil {
		t.Fatalf("validateOptions returned error: %v", err)
	}
}

func TestValidateOptionsRejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name string
		err  string
		w    int
		g    int
		c    string
		p    string
	}{
		{name: "width", err: "-w must be greater than zero", w: 0, g: 8, c: "auto", p: "auto"},
		{name: "group", err: "-g must be greater than zero", w: 16, g: 0, c: "auto", p: "auto"},
		{name: "group width", err: "-g must be less than or equal to -w", w: 8, g: 16, c: "auto", p: "auto"},
		{name: "color", err: "--color must be auto, always, or never", w: 16, g: 8, c: "sometimes", p: "auto"},
		{name: "paging", err: "--paging must be auto, always, or never", w: 16, g: 8, c: "auto", p: "sometimes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOptions(tc.w, tc.g, tc.c, tc.p)
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("error = %v, want containing %q", err, tc.err)
			}
		})
	}
}
