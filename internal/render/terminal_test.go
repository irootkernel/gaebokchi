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
	want := "\n  🐡 gaebokchi · weekly usage\n\n  ChatGPT  a  ███████░░░░░░░░░░░░░     37%  ↻ 2d 4h\n\n"
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

var _ = ginkgo.It("renders two agy weekly rows with blanked second provider label and reset countdowns", func() {
	t := testT{ginkgo.GinkgoT()}
	g := domain.Result{
		TargetID:    "agy:0",
		Provider:    domain.ProviderAgy,
		Account:     "gemini",
		Status:      domain.StatusOK,
		UsedPercent: pct(74.15718734264374),
		Period:      domain.PeriodWeekly,
		ResetAt:     ptrTime(fixedNow.Add(4*24*time.Hour + 3*time.Hour)),
		Source:      domain.SourceAgyCLIUsage,
	}
	c := domain.Result{
		TargetID:    "agy:1",
		Provider:    domain.ProviderAgy,
		Account:     "claude/gpt",
		Status:      domain.StatusOK,
		UsedPercent: pct(12.76919841766357),
		Period:      domain.PeriodWeekly,
		ResetAt:     ptrTime(fixedNow.Add(6*24*time.Hour + 3*time.Hour)),
		Source:      domain.SourceAgyCLIUsage,
	}
	frame, lines := TerminalFrame([]domain.Result{g, c}, 100, fixedNow, false, false, "")
	want := "\n  🐡 gaebokchi · weekly usage\n\n" +
		"  agy  gemini      ███████████████░░░░░   74.2%  ↻ 4d 3h\n" +
		"       claude/gpt  ███░░░░░░░░░░░░░░░░░   12.8%  ↻ 6d 3h\n\n"
	if plain(frame) != want || lines != 6 {
		t.Fatalf("agy frame = %q, lines = %d; want %q", plain(frame), lines, want)
	}

	cu := domain.Result{
		TargetID:    "cursor:0",
		Provider:    domain.ProviderCursor,
		Account:     "default",
		Status:      domain.StatusOK,
		UsedPercent: pct(25),
		Period:      domain.PeriodWeekly,
		ResetAt:     ptrTime(fixedNow.Add(3*24*time.Hour + 2*time.Hour)),
		Source:      domain.SourceCursorCLIUsage,
	}
	cuFrame, cuLines := TerminalFrame([]domain.Result{cu}, 100, fixedNow, false, false, "")
	wantCu := "\n  🐡 gaebokchi · weekly usage\n\n" +
		"  Cursor  default  █████░░░░░░░░░░░░░░░     25%  ↻ 3d 2h\n\n"
	if plain(cuFrame) != wantCu || cuLines != 5 {
		t.Fatalf("cursor frame = %q, lines = %d; want %q", plain(cuFrame), cuLines, wantCu)
	}

	cuMonthly := domain.Result{
		TargetID: "cursor:0",
		Provider: domain.ProviderCursor,
		Account:  "default",
		Status:   domain.StatusNotApplicableMonthly,
		Period:   domain.PeriodMonthly,
		ResetAt:  ptrTime(fixedNow.Add(15 * 24 * time.Hour)),
		Source:   domain.SourceCursorCLIUsage,
	}
	cuMonthlyFrame, _ := TerminalFrame([]domain.Result{cuMonthly}, 100, fixedNow, false, false, "")
	if !strings.Contains(plain(cuMonthlyFrame), "Cursor  default  N/A (monthly)") || strings.Contains(plain(cuMonthlyFrame), "↻") {
		t.Fatalf("cursor monthly frame = %q", plain(cuMonthlyFrame))
	}

	zc := domain.Result{
		TargetID:    "zcode:0",
		Provider:    domain.ProviderZCode,
		Account:     "default",
		Status:      domain.StatusOK,
		UsedPercent: pct(42),
		Period:      domain.PeriodFiveHour,
		ResetAt:     ptrTime(fixedNow.Add(3*time.Hour + 42*time.Minute)),
		Source:      domain.SourceZCodeCLIUsage,
	}
	bothFrame, bothLines := TerminalFrame([]domain.Result{cu, zc}, 100, fixedNow, false, false, "")
	wantBoth := "\n  🐡 gaebokchi · weekly usage\n\n" +
		"  Cursor  default  █████░░░░░░░░░░░░░░░     25%  ↻ 3d 2h\n\n" +
		"  🐡 gaebokchi · 5h usage\n\n" +
		"  ZCode   default  ████████░░░░░░░░░░░░     42%  ↻ 3h 42m\n\n"
	if plain(bothFrame) != wantBoth || bothLines != 9 {
		t.Fatalf("weekly+5h frame = %q, lines = %d; want %q", plain(bothFrame), bothLines, wantBoth)
	}

	onlyZcFrame, onlyZcLines := TerminalFrame([]domain.Result{zc}, 100, fixedNow, false, false, "")
	wantOnlyZc := "\n  🐡 gaebokchi · 5h usage\n\n" +
		"  ZCode  default  ████████░░░░░░░░░░░░     42%  ↻ 3h 42m\n\n"
	if plain(onlyZcFrame) != wantOnlyZc || onlyZcLines != 5 {
		t.Fatalf("5h-only frame = %q, lines = %d; want %q", plain(onlyZcFrame), onlyZcLines, wantOnlyZc)
	}
})

func ptrTime(value time.Time) *time.Time { return &value }

var _ = ginkgo.It("rounds bars to whole cells and leaves precision in the percentage", func() {
	t := testT{ginkgo.GinkgoT()}
	cases := []struct {
		percent float64
		fill    string
		track   int
	}{
		{0, "", 20}, {0.1, "", 20}, {6, "█", 19},
		{8, "██", 18}, {37, "███████", 13}, {42, "████████", 12},
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
	r.Provider = domain.ProviderClaude
	r.Source = domain.SourceClaudeCLIUsage
	r.ResetAt = ptrTime(fixedNow.Add(time.Hour))
	if got := resetText(r, fixedNow); got != "↻ 1h 0m" {
		t.Fatalf("Claude reset = %q, want ↻ 1h 0m", got)
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

var _ = ginkgo.It("shrinks bars independently of reset lines and clips live overflow", func() {
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
	inlineAtLimit, _ := TerminalFrame([]domain.Result{r}, 61, fixedNow, false, false, "")
	fullWithReset, _ := TerminalFrame([]domain.Result{r}, 60, fixedNow, false, false, "")
	shrunkWithReset, _ := TerminalFrame([]domain.Result{r}, 50, fixedNow, false, false, "")
	floorBar, _ := TerminalFrame([]domain.Result{r}, 42, fixedNow, false, false, "")
	narrow, _ := TerminalFrame([]domain.Result{r}, 40, fixedNow, false, false, "")
	tiny, _ := TerminalFrame([]domain.Result{r}, 25, fixedNow, false, false, "")
	fill18, track18 := usageBar(37, 18)
	fill10, track10 := usageBar(37, 10)
	fill20, track20 := usageBar(37, 20)
	if !strings.Contains(plain(wide), "   37%  ↻ 2d 4h\n") ||
		!strings.Contains(plain(inlineAtLimit), "   37%  ↻ 2d 4h\n") ||
		!strings.Contains(plain(fullWithReset), "\n    ↻ 2d 4h\n") ||
		!strings.Contains(plain(fullWithReset), fill20+track20) ||
		!strings.Contains(plain(shrunkWithReset), fill18+track18) ||
		!strings.Contains(plain(floorBar), fill10+track10) ||
		!strings.Contains(narrow, "↻ 2d 4h") || !strings.Contains(plain(narrow), "37%") ||
		strings.Contains(tiny, "░") || !strings.Contains(tiny, r.Account) || !strings.Contains(plain(tiny), "37%") {
		t.Fatalf("width tiers: 100=%q 60=%q 50=%q 42=%q 40=%q 25=%q", wide, fullWithReset, shrunkWithReset, floorBar, narrow, tiny)
	}
})

var _ = ginkgo.It("switches all reset rows together without changing bar widths", func() {
	t := testT{ginkgo.GinkgoT()}
	account := strings.Repeat("a", 23)
	first := measured(8)
	first.Account = account
	first.ResetAt = ptrTime(fixedNow.Add(52 * time.Hour))
	second := measured(6)
	second.Account = "short"
	second.ResetAt = first.ResetAt
	grok := measured(0)
	grok.Provider = domain.ProviderGrok
	grok.Account = "default"
	grok.Source = domain.SourceGrokACPBilling
	grok.ResetAt = ptrTime(fixedNow.Add(11*time.Hour + 20*time.Minute))
	claude := measured(42)
	claude.Provider = domain.ProviderClaude
	claude.Account = "default"
	claude.Source = domain.SourceClaudeCLIUsage
	for _, tc := range []struct {
		columns, barCells int
		inline            bool
	}{{76, 20, true}, {75, 20, false}, {72, 20, false}, {62, 17, false}} {
		frame, count := TerminalFrame([]domain.Result{first, second, grok, claude}, tc.columns, fixedNow, false, false, "")
		clean := plain(frame)
		wantLines, wantSublines := 11, 3
		if tc.inline {
			wantLines, wantSublines = 8, 0
		}
		if count != wantLines || strings.Count(clean, "\n    ↻ ") != wantSublines {
			t.Fatalf("%d columns: reset placement or line count: %q", tc.columns, frame)
		}
		row := 0
		for _, line := range strings.Split(clean, "\n") {
			if !strings.Contains(line, "%") {
				continue
			}
			if row >= 4 {
				t.Fatalf("%d columns: extra numeric row: %q", tc.columns, frame)
			}
			fields := strings.Fields(line)
			percentField := -1
			for i, field := range fields {
				if strings.HasSuffix(field, "%") {
					percentField = i
					break
				}
			}
			if percentField < 1 {
				t.Fatalf("%d columns: missing percentage in row %d: %q", tc.columns, row, line)
			}
			fill, track := usageBar([]float64{8, 6, 0, 42}[row], tc.barCells)
			if fields[percentField-1] != fill+track || strings.Contains(line, "↻") != (tc.inline && row < 3) {
				t.Fatalf("%d columns: row %d layout or bar mismatch: %q", tc.columns, row, line)
			}
			row++
		}
		if row != 4 {
			t.Fatalf("%d columns: got %d numeric rows, want 4", tc.columns, row)
		}
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
	r := NewLiveRenderer(w, oneTarget(), 40, 6, false)
	r.Observe(0, measured(37))
	if err := r.Finish([]domain.Result{measured(37)}); err != nil {
		t.Fatal(err)
	}
	output, writes := w.value()
	if writes != 1 || strings.Contains(output, "[?25") || !strings.Contains(output, "37%") {
		t.Fatalf("height fallback: writes=%d output=%q", writes, output)
	}

	claude := measured(37)
	claude.Provider = domain.ProviderClaude
	claude.Source = domain.SourceClaudeCLIUsage
	claude.ResetAt = ptrTime(time.Now().Add(52 * time.Hour))
	w = &countedWriter{}
	r = NewLiveRenderer(w, []domain.Target{{ID: "claude:0", Index: 0, Provider: domain.ProviderClaude, FallbackAccount: "a"}}, 40, 6, false)
	if err := r.Finish([]domain.Result{claude}); err != nil {
		t.Fatal(err)
	}
	if _, writes = w.value(); writes != 1 {
		t.Fatalf("Claude reset line not counted in the frame height: writes=%d", writes)
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
