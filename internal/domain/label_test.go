package domain

import ginkgo "github.com/onsi/ginkgo/v2"

// TestNormalizeLabelUnicodeWhitespace pins the complete Unicode
// whitespace policy: every White_Space rune normalizes to a space, a
// whitespace-only label normalizes to empty, and ordinary printable
// Unicode (including Korean/CJK) is preserved.
var _ = ginkgo.Describe("TestNormalizeLabelUnicodeWhitespace", func() {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"ideographic space only", "\u3000", ""},
		{"medium mathematical space only", "\u205f", ""},
		{"ogham space mark only", "\u1680", ""},
		{"mixed ascii and unicode whitespace only", " \u3000\t\u205f\u1680\n", ""},
		{"ideographic space between words", "a\u3000b", "a b"},
		{"leading and trailing unicode spaces", "\u205fhello\u1680", "hello"},
		{"korean label preserved", "개복치 계정", "개복치 계정"},
		{"cjk email preserved", "사용자@example.com", "사용자@example.com"},
		{"nbsp collapses with neighbors", "a\u00a0\u00a0b", "a b"},
		{"en quad and hair space", "x\u2000y\u200ay", "x y y"},
		{"narrow no-break space", "x\u202fy", "x y"},
		{"control and format still normalized", "\u202eRTL\u202c", "RTL"},
		{"ordinary text unchanged", "simple@example.com", "simple@example.com"},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			if got := NormalizeLabel(tc.in); got != tc.want {
				t.Fatalf("NormalizeLabel(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
})
