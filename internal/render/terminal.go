package render

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"gaebokchi/internal/domain"
	"golang.org/x/text/width"
)

type span struct {
	text  string
	style textStyle
}

type frameLine []span

const (
	rowGap          = "  "
	percentageCells = 6
	frameFixedLines = 4
)

func frameLineCount(results int) int { return results + frameFixedLines }

type textStyle uint8

const (
	styleBold textStyle = 1 << iota
	styleFaint
	styleGreen
	styleYellow
	styleRed
)

func cellWidth(s string) int {
	n := 0
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			continue
		}
		switch width.LookupRune(r).Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			n += 2
		default:
			n++
		}
	}
	return n
}

func lineWidth(line frameLine) int {
	n := 0
	for _, part := range line {
		n += cellWidth(part.text)
	}
	return n
}

func pad(s string, cells int) string {
	return s + strings.Repeat(" ", max(0, cells-cellWidth(s)))
}

func renderLine(line frameLine, color bool) string {
	var b strings.Builder
	for _, part := range line {
		var params []string
		if part.style&styleBold != 0 {
			params = append(params, "1")
		}
		if part.style&styleFaint != 0 {
			params = append(params, "2")
		}
		if color {
			switch {
			case part.style&styleGreen != 0:
				params = append(params, "32")
			case part.style&styleYellow != 0:
				params = append(params, "33")
			case part.style&styleRed != 0:
				params = append(params, "31")
			}
		}
		if len(params) > 0 {
			b.WriteString("\x1b[")
			b.WriteString(strings.Join(params, ";"))
			b.WriteByte('m')
		}
		b.WriteString(part.text)
		if len(params) > 0 {
			b.WriteString("\x1b[0m")
		}
	}
	return b.String()
}

func clipLine(line frameLine, limit int) frameLine {
	if limit < 1 {
		return nil
	}
	if lineWidth(line) <= limit {
		return line
	}
	remaining := limit - 1
	clipped := make(frameLine, 0, len(line)+1)
	full := false
	for _, part := range line {
		var b strings.Builder
		for _, r := range part.text {
			w := cellWidth(string(r))
			if w > remaining {
				full = true
				break
			}
			b.WriteRune(r)
			remaining -= w
		}
		if b.Len() > 0 {
			clipped = append(clipped, span{b.String(), part.style})
		}
		if full || remaining == 0 {
			break
		}
	}
	return append(clipped, span{"…", 0})
}

func usageBar(percent float64, cells int) (string, string) {
	units := int(math.Round(percent * float64(cells) * 8 / 100))
	if percent > 0 && units == 0 {
		units = 1
	}
	units = max(0, min(cells*8, units))
	full, eighth := units/8, units%8
	fill := strings.Repeat("█", full)
	if eighth > 0 {
		fill += string([]rune("▏▎▍▌▋▊▉")[eighth-1])
	}
	return fill, strings.Repeat("░", cells-cellWidth(fill))
}

func countdown(reset, now time.Time) string {
	d := reset.Sub(now)
	if d <= 0 {
		return "now"
	}
	minutes := int(d / time.Minute)
	if minutes == 0 {
		return "<1m"
	}
	if minutes >= 24*60 {
		return fmt.Sprintf("%dd %dh", minutes/(24*60), minutes/60%24)
	}
	if minutes >= 60 {
		return fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
	}
	return fmt.Sprintf("%dm", minutes)
}

func resetText(result domain.Result, now time.Time) string {
	if result.Status != domain.StatusOK || result.ResetAt == nil || result.Period != domain.PeriodWeekly {
		return ""
	}
	if result.Source != domain.SourceCodexAppServer && result.Source != domain.SourceGrokACPBilling {
		return ""
	}
	return "↻ " + countdown(*result.ResetAt, now)
}

func numericStyle(percent float64) textStyle {
	switch {
	case percent < 50:
		return styleGreen
	case percent < 80:
		return styleYellow
	default:
		return styleRed
	}
}

func statusStyle(status domain.Status) textStyle {
	switch status {
	case domain.StatusNotApplicableMonthly, domain.StatusNotApplicableAPIKey, domain.StatusCancelled:
		return styleFaint
	case domain.StatusAuthRequired, domain.StatusUnavailableNoWeekly, domain.StatusUnavailablePeriod, domain.StatusUnavailableNoData:
		return styleYellow
	default:
		return styleRed
	}
}

func resultLine(result domain.Result, provider, account string, providerCells, accountCells, columns int, now time.Time, spinner string) frameLine {
	base := frameLine{{rowGap, 0}, {pad(provider, providerCells), styleBold}, {rowGap, 0}, {pad(account, accountCells), 0}, {rowGap, 0}}
	if result.Status == "" {
		return append(base, span{spinner + " querying…", styleFaint})
	}
	if result.Status != domain.StatusOK || result.UsedPercent == nil {
		return append(base, span{UsageCell(result), statusStyle(result.Status)})
	}
	percentage := fmt.Sprintf("%*s", percentageCells, UsageCell(result))
	reset := resetText(result, now)
	barCells := 20
	// Reserve one terminal cell so intermediate redraws cannot wrap.
	limit := columns - 1
	barWidth := func() int { return barCells + cellWidth(rowGap) + percentageCells }
	baseWidth := lineWidth(base)
	if baseWidth+barWidth()+cellWidth(rowGap)+cellWidth(reset) > limit && reset != "" {
		barCells = max(10, min(20, limit-baseWidth-cellWidth(rowGap)-percentageCells-cellWidth(rowGap)-cellWidth(reset)))
	}
	if baseWidth+barWidth()+cellWidth(rowGap)+cellWidth(reset) > limit {
		reset = ""
		barCells = 20
	}
	if baseWidth+barWidth() > limit {
		barCells = max(10, min(barCells, limit-baseWidth-cellWidth(rowGap)-percentageCells))
	}
	style := numericStyle(*result.UsedPercent)
	if baseWidth+barWidth() <= limit {
		fill, track := usageBar(*result.UsedPercent, barCells)
		base = append(base, span{fill, style}, span{track, styleFaint}, span{rowGap, 0})
	}
	base = append(base, span{percentage, styleBold | style})
	if reset != "" {
		base = append(base, span{rowGap, 0}, span{reset, styleFaint})
	}
	return base
}

// TerminalFrame renders one inline frame. Pending results have an empty status.
// Intermediate frames clip overflow; the final frame preserves every label.
func TerminalFrame(results []domain.Result, columns int, now time.Time, color, intermediate bool, spinner string) (string, int) {
	providerCells, accountCells := 0, 0
	for _, result := range results {
		providerCells = max(providerCells, cellWidth(result.Provider.DisplayName()))
		accountCells = max(accountCells, cellWidth(SanitizeLabel(result.Account)))
	}
	lines := make([]frameLine, 0, frameLineCount(len(results)))
	lines = append(lines, nil, frameLine{{"  🐡 ", 0}, {"gaebokchi", styleBold}, {" · weekly usage", styleFaint}}, nil)
	var previous domain.ProviderKind
	for _, result := range results {
		provider := result.Provider.DisplayName()
		if result.Provider == previous {
			provider = ""
		}
		previous = result.Provider
		line := resultLine(result, provider, SanitizeLabel(result.Account), providerCells, accountCells, columns, now, spinner)
		lines = append(lines, line)
	}
	lines = append(lines, nil)
	var b strings.Builder
	for _, line := range lines {
		if intermediate {
			line = clipLine(line, columns-1)
		}
		b.WriteString(renderLine(line, color))
		b.WriteByte('\n')
	}
	return b.String(), len(lines)
}
