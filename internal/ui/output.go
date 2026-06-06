package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// ANSI escape codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
)

// Symbols for rich terminal output
const (
	SymCheck    = "✓"
	SymCross    = "✗"
	SymWarn     = "⚠"
	SymArrow    = "→"
	SymDot      = "●"
	SymDownload = "↓"
	SymPackage  = "📦"
	SymLock     = "🔒"
	SymRocket   = "🚀"
	SymTrash    = "🗑"
)

type Level int

const (
	LevelSilent Level = iota
	LevelQuiet
	LevelDefault
	LevelVerbose
)

type ColorMode int

const (
	ColorAuto   ColorMode = iota
	ColorAlways
	ColorNever
)

type Output struct {
	level     Level
	color     ColorMode
	jsonMode  bool
	writer    io.Writer
	errWriter io.Writer
	isTTY     bool
	width     int
}

type Options struct {
	Level     Level
	Color     ColorMode
	JSON      bool
	Writer    io.Writer
	ErrWriter io.Writer
}

func NewOutput(opts Options) *Output {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	if opts.ErrWriter == nil {
		opts.ErrWriter = os.Stderr
	}

	isTTY := false
	width := 80
	if f, ok := opts.Writer.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
		if isTTY {
			w, _, err := term.GetSize(int(f.Fd()))
			if err == nil && w > 0 {
				width = w
			}
		}
	}

	if os.Getenv("NO_COLOR") != "" {
		opts.Color = ColorNever
	}

	return &Output{
		level:     opts.Level,
		color:     opts.Color,
		jsonMode:  opts.JSON,
		writer:    opts.Writer,
		errWriter: opts.ErrWriter,
		isTTY:     isTTY,
		width:     width,
	}
}

func (o *Output) useColor() bool {
	switch o.color {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		return o.isTTY
	}
}

func (o *Output) style(s, codes string) string {
	if !o.useColor() {
		return s
	}
	return codes + s + Reset
}

// --- Status messages ---

func (o *Output) Header(msg string) {
	if o.level <= LevelQuiet {
		return
	}
	fmt.Fprintf(o.writer, "\n%s\n", o.style(msg, Bold+White))
}

func (o *Output) Success(msg string) {
	if o.level <= LevelQuiet {
		return
	}
	fmt.Fprintf(o.writer, " %s %s\n", o.style(SymCheck, Bold+Green), msg)
}

func (o *Output) Error(msg string) {
	fmt.Fprintf(o.errWriter, " %s %s\n", o.style(SymCross, Bold+Red), o.style(msg, Red))
}

func (o *Output) Warning(msg string) {
	if o.level <= LevelSilent {
		return
	}
	fmt.Fprintf(o.errWriter, " %s %s\n", o.style(SymWarn, Bold+Yellow), msg)
}

func (o *Output) Info(msg string) {
	if o.level < LevelDefault {
		return
	}
	fmt.Fprintf(o.writer, " %s %s\n", o.style(SymDot, Cyan), msg)
}

func (o *Output) Verbose(msg string) {
	if o.level < LevelVerbose {
		return
	}
	fmt.Fprintf(o.writer, "   %s\n", o.style(msg, Dim))
}

func (o *Output) Installing(name, version string) {
	if o.level <= LevelQuiet {
		return
	}
	fmt.Fprintf(o.writer, " %s %s %s\n",
		o.style(SymDownload, Bold+Blue), o.style(name, Bold), o.style(version, Cyan))
}

func (o *Output) Installed(name, version string) {
	if o.level <= LevelQuiet {
		return
	}
	fmt.Fprintf(o.writer, " %s %s %s\n",
		o.style(SymCheck, Bold+Green), o.style(name, Bold), o.style(version, Green))
}

func (o *Output) Removed(name, version string) {
	if o.level <= LevelQuiet {
		return
	}
	fmt.Fprintf(o.writer, " %s %s %s\n",
		o.style(SymCross, Red), o.style(name, Bold), o.style(version, Dim))
}

func (o *Output) Summary(msg string) {
	if o.level <= LevelQuiet {
		return
	}
	fmt.Fprintf(o.writer, "\n %s %s\n", o.style(SymRocket, ""), o.style(msg, Bold+Green))
}

// --- Progress Bar (pacman/apt-style) ---

type ProgressBar struct {
	output    *Output
	total     int64
	current   int64
	label     string
	startTime time.Time
}

func (o *Output) Progress(label string, total int64) *ProgressBar {
	return &ProgressBar{
		output:    o,
		total:     total,
		label:     label,
		startTime: time.Now(),
	}
}

// Braille gradient characters from full to empty
var brailleGradient = []rune{'⣿', '⣷', '⣯', '⣟', '⡿', '⣦', '⣤', '⡀', '⠀'}

func (p *ProgressBar) Set(n int64) {
	p.current = n
	if !p.output.isTTY || p.output.level <= LevelQuiet {
		return
	}

	pct := float64(p.current) / float64(p.total) * 100
	if pct > 100 {
		pct = 100
	}

	barWidth := p.output.width - len(p.label) - 20
	if barWidth < 10 {
		barWidth = 10
	}
	if barWidth > 40 {
		barWidth = 40
	}

	// Calculate filled/partial/empty segments
	fillFloat := float64(barWidth) * pct / 100
	fullCells := int(fillFloat)
	partialFraction := fillFloat - float64(fullCells)

	// Build the bar with braille gradient
	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		if i < fullCells {
			bar.WriteRune('⣿') // fully filled
		} else if i == fullCells {
			// Partial cell — pick gradient character
			idx := int(partialFraction * float64(len(brailleGradient)-1))
			if idx >= len(brailleGradient) {
				idx = len(brailleGradient) - 1
			}
			bar.WriteRune(brailleGradient[idx])
		} else {
			bar.WriteRune('⠀') // empty (braille blank)
		}
	}

	// Speed calculation
	speed := ""
	elapsed := time.Since(p.startTime)
	if elapsed > 0 && p.current > 0 {
		bps := float64(p.current) / elapsed.Seconds()
		if bps > 1024*1024 {
			speed = fmt.Sprintf("  %.1f MB/s", bps/1024/1024)
		} else if bps > 1024 {
			speed = fmt.Sprintf("  %.0f KB/s", bps/1024)
		}
	}

	barStr := bar.String()
	if p.output.useColor() {
		fmt.Fprintf(p.output.writer, "\r %s [%s%s%s] %s%3.0f%%%s%s",
			p.label, Green, barStr, Reset, Bold, pct, Reset, speed)
	} else {
		fmt.Fprintf(p.output.writer, "\r %s [%s] %3.0f%%%s", p.label, barStr, pct, speed)
	}
}

func (p *ProgressBar) Finish() {
	if p.output.isTTY && p.output.level > LevelQuiet {
		fmt.Fprintf(p.output.writer, "\r%s\r", strings.Repeat(" ", p.output.width))
	}
}

// --- Spinner (braille dots animation) ---

type Spinner struct {
	output  *Output
	message string
	done    chan struct{}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (o *Output) Spinner(message string) *Spinner {
	s := &Spinner{output: o, message: message, done: make(chan struct{})}
	if o.isTTY && o.level > LevelQuiet {
		go s.animate()
	}
	return s
}

func (s *Spinner) animate() {
	i := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			frame := spinnerFrames[i%len(spinnerFrames)]
			if s.output.useColor() {
				fmt.Fprintf(s.output.writer, "\r %s%s%s %s", Cyan+Bold, frame, Reset, s.message)
			} else {
				fmt.Fprintf(s.output.writer, "\r %s %s", frame, s.message)
			}
			i++
		}
	}
}

func (s *Spinner) Stop(result string) {
	close(s.done)
	if s.output.isTTY && s.output.level > LevelQuiet {
		fmt.Fprintf(s.output.writer, "\r%s\r", strings.Repeat(" ", s.output.width))
		if result != "" {
			s.output.Success(s.message + ": " + result)
		}
	}
}

// --- Table (with aligned columns and header underline) ---

func (o *Output) Table(headers []string, rows [][]string) {
	if o.jsonMode {
		o.tableAsJSON(headers, rows)
		return
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Header row
	for i, h := range headers {
		fmt.Fprintf(o.writer, " %s  ", o.style(fmt.Sprintf("%-*s", widths[i], h), Bold+Underline))
	}
	fmt.Fprintln(o.writer)

	// Data rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprintf(o.writer, " %-*s  ", widths[i], cell)
			}
		}
		fmt.Fprintln(o.writer)
	}
}

func (o *Output) tableAsJSON(headers []string, rows [][]string) {
	var result []map[string]string
	for _, row := range rows {
		item := make(map[string]string)
		for i, cell := range row {
			if i < len(headers) {
				item[headers[i]] = cell
			}
		}
		result = append(result, item)
	}
	o.JSON(result)
}

// --- JSON output ---

func (o *Output) JSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		o.Error(fmt.Sprintf("JSON marshal error: %v", err))
		return
	}
	fmt.Fprintln(o.writer, string(data))
}
