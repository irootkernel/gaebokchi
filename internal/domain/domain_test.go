package domain

import (
	"math"
	"testing"
)

func TestValidatePercent(t *testing.T) {
	cases := []struct {
		name string
		v    float64
		want bool
	}{
		{"zero", 0, true},
		{"hundred", 100, true},
		{"fractional", 37.5, true},
		{"tiny negative", -0.0001, false},
		{"over 100", 100.0001, false},
		{"negative", -12, false},
		{"nan", math.NaN(), false},
		{"plus inf", math.Inf(1), false},
		{"minus inf", math.Inf(-1), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidatePercent(tc.v); got != tc.want {
				t.Fatalf("ValidatePercent(%v) = %v, want %v", tc.v, got, tc.want)
			}
		})
	}
}

func TestNormalizePercentSignedZero(t *testing.T) {
	if v := NormalizePercentSignedZero(math.Copysign(0, -1)); math.Signbit(v) {
		t.Fatalf("negative zero preserved: %v", v)
	}
	if v := NormalizePercentSignedZero(-5.5); v != -5.5 {
		t.Fatalf("negative value altered: %v", v)
	}
}

func TestStatusCells(t *testing.T) {
	bogus := Status("bogus")
	cases := map[Status]string{
		StatusOK:                   "",
		StatusNotApplicableMonthly: "N/A (monthly)",
		StatusNotApplicableAPIKey:  "N/A (API key)",
		StatusUnavailableNoWeekly:  "N/A (no weekly)",
		StatusUnavailablePeriod:    "N/A (period)",
		StatusUnavailableNoData:    "N/A (no data)",
		StatusAuthRequired:         "AUTH REQUIRED",
		StatusUnsupported:          "UNSUPPORTED",
		StatusCLINotFound:          "NOT INSTALLED",
		StatusHomeNotFound:         "HOME NOT FOUND",
		StatusHomeError:            "HOME ERROR",
		StatusTimeout:              "TIMEOUT",
		StatusProtocolError:        "PROTOCOL ERROR",
		StatusProviderError:        "ERROR",
		StatusCancelled:            "CANCELLED",
		bogus:                      "ERROR",
	}
	for s, want := range cases {
		if got := s.Cell(); got != want {
			t.Errorf("Status(%q).Cell() = %q, want %q", s, got, want)
		}
	}
}

func TestStatusExitClass(t *testing.T) {
	clean := []Status{StatusOK, StatusNotApplicableMonthly, StatusNotApplicableAPIKey}
	incomplete := []Status{
		StatusUnavailableNoWeekly, StatusUnavailablePeriod, StatusUnavailableNoData,
		StatusAuthRequired, StatusUnsupported, StatusCLINotFound, StatusHomeNotFound,
		StatusHomeError, StatusTimeout, StatusProtocolError, StatusProviderError,
	}
	for _, s := range clean {
		if got := s.ExitClass(); got != ExitClean {
			t.Errorf("Status(%q).ExitClass() = %v, want clean", s, got)
		}
	}
	for _, s := range incomplete {
		if got := s.ExitClass(); got != ExitIncomplete {
			t.Errorf("Status(%q).ExitClass() = %v, want incomplete", s, got)
		}
	}
	if got := StatusCancelled.ExitClass(); got != ExitSignal {
		t.Errorf("cancelled ExitClass() = %v, want signal", got)
	}
}

func TestProviderDisplayName(t *testing.T) {
	if ProviderCodex.DisplayName() != "ChatGPT" {
		t.Fatalf("codex display name = %q", ProviderCodex.DisplayName())
	}
	if ProviderGrok.DisplayName() != "Grok" {
		t.Fatalf("grok display name = %q", ProviderGrok.DisplayName())
	}
}
