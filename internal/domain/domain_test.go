package domain

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"math"
)

var _ = ginkgo.Describe("TestValidatePercent", func() {
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
		ginkgo.It(tc.name, func() {
			gomega.Expect(ValidatePercent(tc.v)).To(gomega.Equal(tc.want), tc.name)
		})
	}
})

var _ = ginkgo.It("TestNormalizePercentSignedZero", func() {
	t := testT{ginkgo.GinkgoT()}
	if v := NormalizePercentSignedZero(math.Copysign(0, -1)); math.Signbit(v) {
		t.Fatalf("negative zero preserved: %v", v)
	}
	if v := NormalizePercentSignedZero(-5.5); v != -5.5 {
		t.Fatalf("negative value altered: %v", v)
	}
})

var _ = ginkgo.Describe("TestStatusCells", func() {
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
		ginkgo.It(string(s), func() {
			t := testT{ginkgo.GinkgoT()}
			if got := s.Cell(); got != want {
				t.Fatalf("Status(%q).Cell() = %q, want %q", s, got, want)
			}
		})
	}
})

var _ = ginkgo.Describe("TestStatusExitClass", func() {
	clean := []Status{StatusOK, StatusNotApplicableMonthly, StatusNotApplicableAPIKey}
	incomplete := []Status{
		StatusUnavailableNoWeekly, StatusUnavailablePeriod, StatusUnavailableNoData,
		StatusAuthRequired, StatusUnsupported, StatusCLINotFound, StatusHomeNotFound,
		StatusHomeError, StatusTimeout, StatusProtocolError, StatusProviderError,
	}
	for _, s := range clean {
		ginkgo.It("clean: "+string(s), func() {
			t := testT{ginkgo.GinkgoT()}
			if got := s.ExitClass(); got != ExitClean {
				t.Fatalf("Status(%q).ExitClass() = %v, want clean", s, got)
			}
		})
	}
	for _, s := range incomplete {
		ginkgo.It("incomplete: "+string(s), func() {
			t := testT{ginkgo.GinkgoT()}
			if got := s.ExitClass(); got != ExitIncomplete {
				t.Fatalf("Status(%q).ExitClass() = %v, want incomplete", s, got)
			}
		})
	}
	ginkgo.It("cancelled", func() {
		t := testT{ginkgo.GinkgoT()}
		if got := StatusCancelled.ExitClass(); got != ExitSignal {
			t.Fatalf("cancelled ExitClass() = %v, want signal", got)
		}
	})
})

var _ = ginkgo.It("TestProviderDisplayName", func() {
	t := testT{ginkgo.GinkgoT()}
	if ProviderCodex.DisplayName() != "ChatGPT" {
		t.Fatalf("codex display name = %q", ProviderCodex.DisplayName())
	}
	if ProviderGrok.DisplayName() != "Grok" {
		t.Fatalf("grok display name = %q", ProviderGrok.DisplayName())
	}
	if ProviderClaude.DisplayName() != "Claude" {
		t.Fatalf("claude display name = %q", ProviderClaude.DisplayName())
	}
	if ProviderAgy.DisplayName() != "agy" {
		t.Fatalf("agy display name = %q", ProviderAgy.DisplayName())
	}
})

var _ = ginkgo.Describe("TestRemainingFractionToUsedPercent", func() {
	cases := []struct {
		name  string
		token string
		want  float64
		ok    bool
	}{
		{"zero remaining", "0", 100, true},
		{"full remaining", "1", 0, true},
		{"full remaining decimal", "1.0", 0, true},
		{"signed zero remaining", "-0.0", 100, true},
		{"exact decimal 0.9", "0.9", 10, true},
		{"captured gemini weekly", "0.2584281265735626", 74.15718734264374, true},
		{"captured 3p weekly", "0.8723080158233643", 12.76919841766357, true},
		{"scientific notation", "1e-2", 99, true},
		{"negative fraction", "-0.0001", 0, false},
		{"above one", "1.0001", 0, false},
		{"empty token", "", 0, false},
		{"malformed token", "abc", 0, false},
		{"unbounded exponent", "1e-1000000000", 0, false},
		{"exponent above bound", "1e-129", 0, false},
		{"over 64 coefficient digits", "0." + "12345678901234567890123456789012345678901234567890123456789012345", 0, false},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			got, ok := RemainingFractionToUsedPercent(tc.token)
			gomega.Expect(ok).To(gomega.Equal(tc.ok), tc.name)
			if tc.ok {
				gomega.Expect(got).To(gomega.Equal(tc.want), tc.name)
				gomega.Expect(math.Signbit(got)).To(gomega.BeFalse(), tc.name)
			}
		})
	}
})
