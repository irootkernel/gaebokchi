package render

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"

	"gaebokchi/internal/domain"
)

var fixedNow = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
var sgr = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string { return sgr.ReplaceAllString(s, "") }

func measured(percent float64) domain.Result {
	return domain.Result{Provider: domain.ProviderCodex, Account: "a", Status: domain.StatusOK,
		UsedPercent: &percent, Period: domain.PeriodWeekly, Source: domain.SourceCodexAppServer}
}

var _ = ginkgo.It("renders a stable colorless frame and provenance-based countdown", func() {
	t := testT{ginkgo.GinkgoT()}
	r := measured(37)
	r.ResetAt = ptrTime(fixedNow.Add(52 * time.Hour))
	frame, lines := TerminalFrame([]domain.Result{r}, 100, fixedNow, false, false, "⠹")
	want := "\n  🐡 gaebokchi · weekly usage\n\n  ChatGPT  a  ███████▍░░░░░░░░░░░░     37%  ↻ 2d 4h\n\n"
	if plain(frame) != want || lines != 5 {
		t.Fatalf("frame = %q, lines = %d; want %q", frame, lines, want)
	}
	if strings.Contains(frame, "[31m") || strings.Contains(frame, "[32m") || strings.Contains(frame, "[33m") {
		t.Fatal("colorless frame emitted color")
	}
})

var _ = ginkgo.It("keeps a fractional percentage in the terminal frame", func() {
	t := testT{ginkgo.GinkgoT()}
	frame, _ := TerminalFrame([]domain.Result{measured(43.2)}, 100, fixedNow, false, false, "")
	if !strings.Contains(plain(frame), "43.2%") {
		t.Fatalf("fractional percentage missing from frame: %q", frame)
	}
})

var _ = ginkgo.It("renders a colorless grouped frame with a sanitized wide label and no countdown", func() {
	t := testT{ginkgo.GinkgoT()}
	a, b := measured(10), measured(20)
	a.Account = "한글\tname"
	b.Account = "second"
	frame, lines := TerminalFrame([]domain.Result{a, b}, 100, fixedNow, false, false, "")
	want := "\n  🐡 gaebokchi · weekly usage\n\n" +
		"  ChatGPT  한글 name  ██░░░░░░░░░░░░░░░░░░     10%\n" +
		"           second     ████░░░░░░░░░░░░░░░░     20%\n\n"
	if plain(frame) != want || lines != 6 {
		t.Fatalf("frame = %q, lines = %d; want %q", plain(frame), lines, want)
	}
})

func ptrTime(value time.Time) *time.Time { return &value }

var _ = ginkgo.It("rounds bars to eighths and keeps a nonzero sliver", func() {
	t := testT{ginkgo.GinkgoT()}
	cases := []struct {
		percent float64
		fill    string
		track   int
	}{
		{0, "", 20}, {0.1, "▏", 19}, {37, "███████▍", 12},
		{99.99, strings.Repeat("█", 20), 0}, {100, strings.Repeat("█", 20), 0},
	}
	for _, tc := range cases {
		fill, track := usageBar(tc.percent, 20)
		if fill != tc.fill || track != strings.Repeat("░", tc.track) {
			t.Fatalf("%.2f: fill=%q track=%q", tc.percent, fill, track)
		}
	}
})

var _ = ginkgo.It("formats countdowns with truncated whole units", func() {
	t := testT{ginkgo.GinkgoT()}
	cases := []struct {
		duration time.Duration
		want     string
	}{
		{52*time.Hour + 59*time.Minute, "2d 4h"},
		{4*time.Hour + 12*time.Minute + 59*time.Second, "4h 12m"},
		{12*time.Minute + 59*time.Second, "12m"},
		{59 * time.Second, "<1m"},
		{-time.Second, "now"},
	}
	for _, tc := range cases {
		if got := countdown(fixedNow.Add(tc.duration), fixedNow); got != tc.want {
			t.Fatalf("%v: %q, want %q", tc.duration, got, tc.want)
		}
	}
	r := measured(37)
	r.Source = domain.SourceClaudeCLIUsage
	r.ResetAt = ptrTime(fixedNow.Add(time.Hour))
	if resetText(r, fixedNow) != "" {
		t.Fatal("unqualified Claude reset shown")
	}
	r.Source = domain.SourceGrokACPBilling
	r.Period = domain.PeriodMonthly
	if resetText(r, fixedNow) != "" {
		t.Fatal("monthly reset shown")
	}
})

var _ = ginkgo.It("groups providers, sanitizes labels, and counts wide cells", func() {
	t := testT{ginkgo.GinkgoT()}
	a := measured(10)
	a.Account = "한글\x1b[31m\nname"
	b := measured(20)
	b.Account = "second"
	frame, _ := TerminalFrame([]domain.Result{a, b}, 100, fixedNow, false, false, "")
	if cellWidth("한글") != 4 || strings.Count(frame, "ChatGPT") != 1 || strings.Contains(plain(frame), "\x1b") || strings.Contains(plain(frame), "\nname") {
		t.Fatalf("unsafe grouped frame: %q", frame)
	}
})

var _ = ginkgo.It("maps numeric and terminal statuses to approved style classes", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, tc := range []struct {
		pct  float64
		want textStyle
		sgr  string
	}{{49.9, styleGreen, "32"}, {50, styleYellow, "33"}, {79.9, styleYellow, "33"}, {80, styleRed, "31"}} {
		if got := numericStyle(tc.pct); got != tc.want {
			t.Fatalf("%.1f style = %d", tc.pct, got)
		}
		frame, _ := TerminalFrame([]domain.Result{measured(tc.pct)}, 100, fixedNow, true, false, "")
		if !strings.Contains(frame, "\x1b[1;"+tc.sgr+"m") || !strings.Contains(frame, "\x1b["+tc.sgr+"m") {
			t.Fatalf("%.1f frame missing color %s: %q", tc.pct, tc.sgr, frame)
		}
	}
	for _, tc := range []struct {
		status domain.Status
		want   textStyle
		sgr    string
	}{
		{domain.StatusNotApplicableMonthly, styleFaint, "2"}, {domain.StatusNotApplicableAPIKey, styleFaint, "2"},
		{domain.StatusCancelled, styleFaint, "2"}, {domain.StatusAuthRequired, styleYellow, "33"},
		{domain.StatusUnavailableNoWeekly, styleYellow, "33"}, {domain.StatusUnavailablePeriod, styleYellow, "33"},
		{domain.StatusUnavailableNoData, styleYellow, "33"}, {domain.StatusTimeout, styleRed, "31"},
		{domain.StatusUnsupported, styleRed, "31"}, {domain.StatusCLINotFound, styleRed, "31"},
		{domain.StatusHomeNotFound, styleRed, "31"}, {domain.StatusHomeError, styleRed, "31"},
		{domain.StatusProtocolError, styleRed, "31"}, {domain.StatusProviderError, styleRed, "31"},
	} {
		if got := statusStyle(tc.status); got != tc.want {
			t.Fatalf("%s style = %d, want %d", tc.status, got, tc.want)
		}
		frame, _ := TerminalFrame([]domain.Result{{Provider: domain.ProviderGrok, Account: "default", Status: tc.status}}, 100, fixedNow, true, false, "")
		if !strings.Contains(frame, "\x1b["+tc.sgr+"m"+tc.status.Cell()) {
			t.Fatalf("%s frame missing style %s: %q", tc.status, tc.sgr, frame)
		}
		colorless, _ := TerminalFrame([]domain.Result{{Provider: domain.ProviderGrok, Account: "default", Status: tc.status}}, 100, fixedNow, false, false, "")
		if !strings.Contains(colorless, tc.status.Cell()) || strings.Contains(colorless, "\x1b[31m") ||
			strings.Contains(colorless, "\x1b[32m") || strings.Contains(colorless, "\x1b[33m") {
			t.Fatalf("%s colorless status frame: %q", tc.status, colorless)
		}
		if tc.want == styleFaint && !strings.Contains(colorless, "\x1b[2m"+tc.status.Cell()) {
			t.Fatalf("%s lost faint style: %q", tc.status, colorless)
		}
	}
	allowed := regexp.MustCompile(`^\x1b\[(?:0|1|2|31|32|33|1;31|1;32|1;33)m`)
	for _, color := range []bool{true, false} {
		frame, _ := TerminalFrame([]domain.Result{measured(80)}, 100, fixedNow, color, false, "")
		if !color && (!strings.Contains(frame, "\x1b[1m   80%") || strings.Contains(frame, "\x1b[1;m")) {
			t.Fatalf("NO_COLOR must preserve bold with a valid SGR: %q", frame)
		}
		for i := 0; i < len(frame); i++ {
			if frame[i] != '\x1b' {
				continue
			}
			match := allowed.FindString(frame[i:])
			if match == "" {
				t.Fatalf("disallowed escape at %d: %q", i, frame[i:])
			}
			i += len(match) - 1
		}
	}
})

var _ = ginkgo.It("shrinks bars, drops countdown then bar, and clips only live labels", func() {
	t := testT{ginkgo.GinkgoT()}
	r := measured(37)
	r.Account = "한글-alias"
	r.ResetAt = ptrTime(fixedNow.Add(52 * time.Hour))
	for _, columns := range []int{100, 60, 40, 25} {
		frame, _ := TerminalFrame([]domain.Result{r}, columns, fixedNow, false, true, "⠹")
		for _, line := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n") {
			if cellWidth(plain(line)) > columns-1 {
				t.Fatalf("%d columns: overflow line %q", columns, line)
			}
		}
	}
	wide, _ := TerminalFrame([]domain.Result{r}, 100, fixedNow, false, false, "")
	shrunkWithReset, _ := TerminalFrame([]domain.Result{r}, 60, fixedNow, false, false, "")
	withoutReset, _ := TerminalFrame([]domain.Result{r}, 50, fixedNow, false, false, "")
	floorBar, _ := TerminalFrame([]domain.Result{r}, 42, fixedNow, false, false, "")
	narrow, _ := TerminalFrame([]domain.Result{r}, 40, fixedNow, false, false, "")
	tiny, _ := TerminalFrame([]domain.Result{r}, 25, fixedNow, false, false, "")
	fill19, track19 := usageBar(37, 19)
	fill18, track18 := usageBar(37, 18)
	fill10, track10 := usageBar(37, 10)
	if !strings.Contains(wide, "↻ 2d 4h") ||
		!strings.Contains(plain(shrunkWithReset), fill19+track19) || !strings.Contains(shrunkWithReset, "↻ 2d 4h") ||
		!strings.Contains(plain(withoutReset), fill18+track18) || strings.Contains(withoutReset, "↻") ||
		!strings.Contains(plain(floorBar), fill10+track10) || strings.Contains(floorBar, "↻") ||
		strings.Contains(narrow, "↻") || !strings.Contains(plain(narrow), "37%") ||
		strings.Contains(tiny, "░") || !strings.Contains(tiny, r.Account) || !strings.Contains(plain(tiny), "37%") {
		t.Fatalf("width tiers: 100=%q 60=%q 50=%q 42=%q 40=%q 25=%q", wide, shrunkWithReset, withoutReset, floorBar, narrow, tiny)
	}
})

type countedWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	writes int
	failAt int
}

func (w *countedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writes++
	if w.failAt > 0 && w.writes >= w.failAt {
		return 0, errors.New("injected write failure")
	}
	return w.buf.Write(p)
}

func (w *countedWriter) value() (string, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String(), w.writes
}

func oneTarget() []domain.Target {
	return []domain.Target{{ID: "codex:0", Index: 0, Provider: domain.ProviderCodex, FallbackAccount: "a"}}
}

var _ = ginkgo.It("serializes concurrent observations and leaves the final frame with cursor visible", func() {
	t := testT{ginkgo.GinkgoT()}
	w := &countedWriter{}
	r := NewLiveRenderer(w, oneTarget(), 100, 30, true)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Observe(0, measured(37))
		}()
	}
	wg.Wait()
	final := measured(37)
	if err := r.Finish([]domain.Result{final}); err != nil {
		t.Fatal(err)
	}
	output, writes := w.value()
	if writes < 2 || !strings.HasSuffix(output, "\x1b[?25h") || strings.Count(output, "\x1b[?25l") != 1 || !strings.Contains(output, spinnerFrames[0]+" querying…") {
		t.Fatalf("bad live lifecycle: writes=%d output=%q", writes, output)
	}
})

var _ = ginkgo.It("writes one final frame on short terminals and stops after first output failure", func() {
	t := testT{ginkgo.GinkgoT()}
	w := &countedWriter{}
	r := NewLiveRenderer(w, oneTarget(), 40, 5, false)
	r.Observe(0, measured(37))
	if err := r.Finish([]domain.Result{measured(37)}); err != nil {
		t.Fatal(err)
	}
	output, writes := w.value()
	if writes != 1 || strings.Contains(output, "[?25") || !strings.Contains(output, "37%") {
		t.Fatalf("height fallback: writes=%d output=%q", writes, output)
	}

	failing := &countedWriter{failAt: 1}
	r = NewLiveRenderer(failing, oneTarget(), 100, 30, true)
	r.Observe(0, measured(37))
	if err := r.Finish([]domain.Result{measured(37)}); err == nil {
		t.Fatal("write failure lost")
	}
	_, writes = failing.value()
	if writes != 1 {
		t.Fatalf("drew %d times after first failure", writes)
	}

	failing = &countedWriter{failAt: 2}
	r = NewLiveRenderer(failing, oneTarget(), 100, 30, true)
	r.Observe(0, measured(37))
	time.Sleep(20 * time.Millisecond)
	if err := r.Finish([]domain.Result{measured(37)}); err == nil {
		t.Fatal("second write failure lost")
	}
	_, writes = failing.value()
	if writes != 2 {
		t.Fatalf("drew %d times after second-write failure", writes)
	}
})
