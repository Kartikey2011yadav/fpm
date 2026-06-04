package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
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
	level    Level
	color    ColorMode
	json     bool
	writer   io.Writer
	errWriter io.Writer
	isTTY    bool
}

type Options struct {
	Level    Level
	Color    ColorMode
	JSON     bool
	Writer   io.Writer
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
	if f, ok := opts.Writer.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	// Respect NO_COLOR env var
	if os.Getenv("NO_COLOR") != "" {
		opts.Color = ColorNever
	}

	return &Output{
		level:     opts.Level,
		color:     opts.Color,
		json:      opts.JSON,
		writer:    opts.Writer,
		errWriter: opts.ErrWriter,
		isTTY:     isTTY,
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

// Status messages

func (o *Output) Success(msg string) {
	if o.level <= LevelQuiet {
		return
	}
	if o.useColor() {
		fmt.Fprintf(o.writer, "\033[32m✓\033[0m %s\n", msg)
	} else {
		fmt.Fprintf(o.writer, "  %s\n", msg)
	}
}

func (o *Output) Warning(msg string) {
	if o.level <= LevelSilent {
		return
	}
	if o.useColor() {
		fmt.Fprintf(o.errWriter, "\033[33m⚠\033[0m %s\n", msg)
	} else {
		fmt.Fprintf(o.errWriter, "warning: %s\n", msg)
	}
}

func (o *Output) Error(msg string) {
	if o.useColor() {
		fmt.Fprintf(o.errWriter, "\033[31m✗\033[0m %s\n", msg)
	} else {
		fmt.Fprintf(o.errWriter, "error: %s\n", msg)
	}
}

func (o *Output) Info(msg string) {
	if o.level < LevelDefault {
		return
	}
	fmt.Fprintf(o.writer, "  %s\n", msg)
}

func (o *Output) Verbose(msg string) {
	if o.level < LevelVerbose {
		return
	}
	if o.useColor() {
		fmt.Fprintf(o.writer, "\033[2m  %s\033[0m\n", msg)
	} else {
		fmt.Fprintf(o.writer, "  [debug] %s\n", msg)
	}
}

func (o *Output) Header(msg string) {
	if o.level <= LevelQuiet {
		return
	}
	if o.useColor() {
		fmt.Fprintf(o.writer, "\033[1m%s\033[0m\n", msg)
	} else {
		fmt.Fprintf(o.writer, "%s\n", msg)
	}
}

// JSON output

func (o *Output) JSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		o.Error(fmt.Sprintf("failed to marshal JSON: %v", err))
		return
	}
	fmt.Fprintln(o.writer, string(data))
}

// Table output

func (o *Output) Table(headers []string, rows [][]string) {
	if o.json {
		o.tableAsJSON(headers, rows)
		return
	}

	// Calculate column widths
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

	// Print header
	if o.useColor() {
		fmt.Fprintf(o.writer, "\033[1m")
	}
	for i, h := range headers {
		fmt.Fprintf(o.writer, "%-*s  ", widths[i], h)
	}
	if o.useColor() {
		fmt.Fprintf(o.writer, "\033[0m")
	}
	fmt.Fprintln(o.writer)

	// Separator
	for i, w := range widths {
		fmt.Fprintf(o.writer, "%s  ", strings.Repeat("─", w))
		_ = i
	}
	fmt.Fprintln(o.writer)

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprintf(o.writer, "%-*s  ", widths[i], cell)
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

// Progress

type ProgressBar struct {
	output  *Output
	total   int64
	current int64
	label   string
}

func (o *Output) Progress(label string, total int64) *ProgressBar {
	return &ProgressBar{
		output: o,
		total:  total,
		label:  label,
	}
}

func (p *ProgressBar) Set(n int64) {
	p.current = n
	if !p.output.isTTY || p.output.level <= LevelQuiet {
		return
	}

	pct := float64(p.current) / float64(p.total) * 100
	barWidth := 30
	filled := int(float64(barWidth) * float64(p.current) / float64(p.total))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Fprintf(p.output.writer, "\r  %s [%s] %.0f%%", p.label, bar, pct)
}

func (p *ProgressBar) Finish() {
	if p.output.isTTY && p.output.level > LevelQuiet {
		fmt.Fprintf(p.output.writer, "\r%s\r", strings.Repeat(" ", 80))
	}
}

// Spinner

type Spinner struct {
	output  *Output
	message string
	active  bool
}

func (o *Output) Spinner(message string) *Spinner {
	s := &Spinner{output: o, message: message, active: true}
	if o.isTTY && o.level > LevelQuiet {
		fmt.Fprintf(o.writer, "  %s...", message)
	}
	return s
}

func (s *Spinner) Stop(result string) {
	s.active = false
	if s.output.isTTY && s.output.level > LevelQuiet {
		fmt.Fprintf(s.output.writer, "\r  %s: %s\n", s.message, result)
	}
}
