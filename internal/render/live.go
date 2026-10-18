package render

import (
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"gaebokchi/internal/domain"
)

var spinnerFrames = [...]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// LiveRenderer owns all terminal writes for one finite query. Observe is safe
// for concurrent Runner callbacks and does no terminal I/O itself.
type LiveRenderer struct {
	out       io.Writer
	columns   int
	color     bool
	live      bool
	resultsMu sync.Mutex
	results   []domain.Result
	updates   chan struct{}
	stop      chan struct{}
	done      chan struct{}
	drawMu    sync.Mutex
	lastLines int
	writeErr  error
	frame     int
}

func NewLiveRenderer(out io.Writer, targets []domain.Target, columns, rows int, color bool) *LiveRenderer {
	hasWeekly, hasFiveHour := false, false
	for _, target := range targets {
		if target.Provider.IsFiveHour() {
			hasFiveHour = true
		} else {
			hasWeekly = true
		}
	}
	// Every target may add one reset line below its row.
	maxLines := len(targets) + sectionFixedLines(hasWeekly, hasFiveHour) + len(targets)
	r := &LiveRenderer{
		out: out, columns: columns, color: color,
		live:    maxLines < rows,
		results: make([]domain.Result, len(targets)),
		updates: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{}),
	}
	for _, target := range targets {
		r.results[target.Index] = domain.Result{
			TargetID: target.ID, Provider: target.Provider, Account: target.FallbackAccount,
		}
	}
	if r.live {
		r.draw(false, true)
		go r.loop()
	} else {
		close(r.done)
	}
	return r
}

func (r *LiveRenderer) Observe(index int, result domain.Result) {
	r.resultsMu.Lock()
	r.results[index] = result
	r.resultsMu.Unlock()
	if r.live {
		select {
		case r.updates <- struct{}{}:
		default:
		}
	}
}

func (r *LiveRenderer) snapshot() []domain.Result {
	r.resultsMu.Lock()
	defer r.resultsMu.Unlock()
	return append([]domain.Result(nil), r.results...)
}

func (r *LiveRenderer) loop() {
	defer close(r.done)
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-r.updates:
			r.draw(false, false)
		case <-ticker.C:
			r.drawMu.Lock()
			r.frame = (r.frame + 1) % len(spinnerFrames)
			r.drawMu.Unlock()
			r.draw(false, false)
		}
	}
}

func (r *LiveRenderer) draw(final, first bool) {
	r.drawMu.Lock()
	defer r.drawMu.Unlock()
	if r.writeErr != nil {
		return
	}
	frame, lines := TerminalFrame(r.snapshot(), r.columns, time.Now(), r.color, !final, spinnerFrames[r.frame])
	var output strings.Builder
	if first {
		output.WriteString("\x1b[?25l")
	}
	output.WriteByte('\r')
	if r.lastLines > 0 {
		output.WriteString("\x1b[")
		output.WriteString(strconv.Itoa(r.lastLines))
		output.WriteByte('A')
	}
	for _, line := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n") {
		output.WriteString("\x1b[2K")
		output.WriteString(line)
		output.WriteByte('\n')
	}
	output.WriteString("\x1b[J")
	if final {
		output.WriteString("\x1b[?25h")
	}
	value := output.String()
	n, err := io.WriteString(r.out, value)
	if err == nil && n != len(value) {
		err = io.ErrShortWrite
	}
	if err != nil {
		r.writeErr = err
		return
	}
	r.lastLines = lines
}

// Finish stops progress and emits the ordered, disambiguated final result.
// A short terminal uses one plain frame write and never hides the cursor.
func (r *LiveRenderer) Finish(results []domain.Result) error {
	if r.live {
		close(r.stop)
		<-r.done
	}
	r.resultsMu.Lock()
	copy(r.results, results)
	r.resultsMu.Unlock()
	if r.live {
		r.draw(true, false)
	} else {
		frame, _ := TerminalFrame(results, r.columns, time.Now(), r.color, false, "")
		n, err := io.WriteString(r.out, frame)
		if err == nil && n != len(frame) {
			err = io.ErrShortWrite
		}
		r.writeErr = err
	}
	return r.writeErr
}
