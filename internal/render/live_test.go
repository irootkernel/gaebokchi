package render

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"

	"gaebokchi/internal/domain"
)

// screenModel handles only the terminal operations permitted by ARCH-10.
type screenModel struct {
	lines   []string
	row     int
	column  int
	visible bool
}

func (m *screenModel) ensureRow() {
	for len(m.lines) <= m.row {
		m.lines = append(m.lines, "")
	}
}

func (m *screenModel) apply(s string) error {
	chars := []rune(s)
	for i := 0; i < len(chars); i++ {
		switch chars[i] {
		case '\r':
			m.column = 0
		case '\n':
			m.row++
			m.column = 0
		case '\x1b':
			if i+1 >= len(chars) || chars[i+1] != '[' {
				return strconv.ErrSyntax
			}
			i += 2
			start := i
			for i < len(chars) && (chars[i] < 'A' || chars[i] > 'z') {
				i++
			}
			if i >= len(chars) {
				return strconv.ErrSyntax
			}
			arg := string(chars[start:i])
			switch chars[i] {
			case 'm':
			case 'A':
				n, err := strconv.Atoi(arg)
				if err != nil || n > m.row {
					return strconv.ErrSyntax
				}
				m.row -= n
			case 'K':
				if arg != "2" {
					return strconv.ErrSyntax
				}
				m.ensureRow()
				m.lines[m.row] = ""
			case 'J':
				m.ensureRow()
				m.lines = m.lines[:m.row+1]
			case 'h', 'l':
				if arg != "?25" {
					return strconv.ErrSyntax
				}
				m.visible = chars[i] == 'h'
			default:
				return strconv.ErrSyntax
			}
		default:
			m.ensureRow()
			line := []rune(m.lines[m.row])
			if m.column == len(line) {
				line = append(line, chars[i])
			} else if m.column < len(line) {
				line[m.column] = chars[i]
			} else {
				line = append(line, []rune(strings.Repeat(" ", m.column-len(line)))...)
				line = append(line, chars[i])
			}
			m.lines[m.row] = string(line)
			m.column++
		}
	}
	return nil
}

var _ = ginkgo.It("leaves exactly the completed frame on a modeled screen after cancellation", func() {
	t := testT{ginkgo.GinkgoT()}
	targets := []domain.Target{
		{ID: "codex:0", Index: 0, Provider: domain.ProviderCodex, FallbackAccount: "first"},
		{ID: "grok:0", Index: 1, Provider: domain.ProviderGrok, FallbackAccount: "default"},
	}
	w := &countedWriter{}
	live := NewLiveRenderer(w, targets, 100, 30, true)
	result := measured(37)
	result.Account = "first@example.com"
	result.ResetAt = ptrTime(time.Now().Add(52*time.Hour + time.Minute))
	live.Observe(0, result)
	time.Sleep(20 * time.Millisecond)
	cancelled := domain.Result{Provider: domain.ProviderGrok, Account: "default", Status: domain.StatusCancelled}
	live.Observe(1, cancelled)
	results := []domain.Result{result, cancelled}
	if err := live.Finish(results); err != nil {
		t.Fatal(err)
	}
	output, _ := w.value()
	model := &screenModel{visible: true}
	if err := model.apply(output); err != nil {
		t.Fatalf("invalid terminal sequence: %v", err)
	}
	wantFrame, _ := TerminalFrame(results, 100, time.Now(), true, false, "")
	want := strings.Split(strings.TrimSuffix(plain(wantFrame), "\n"), "\n")
	for len(model.lines) > len(want) && model.lines[len(model.lines)-1] == "" {
		model.lines = model.lines[:len(model.lines)-1]
	}
	if !reflect.DeepEqual(model.lines, want) || !model.visible {
		t.Fatalf("screen=%q, cursor=%t; want=%q", model.lines, model.visible, want)
	}
})
