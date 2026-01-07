package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

// Symbols
const (
	symbolSuccess = "✓"
	symbolError   = "✗"
	symbolWarning = "⚠"
	symbolInfo    = "ℹ"
)

var (
	// Out is the standard output writer
	Out io.Writer = os.Stdout

	// Err is the error output writer
	Err io.Writer = os.Stderr

	// useColors determines if colors should be used
	useColors = isTerminal()

	// verboseMode tracks if verbose output is enabled
	verboseMode = false

	// quietMode tracks if quiet mode is enabled
	quietMode = false
)

// SetVerbose sets the verbose mode
func SetVerbose(v bool) {
	verboseMode = v
}

// SetQuiet sets the quiet mode
func SetQuiet(q bool) {
	quietMode = q
}

// isTerminal checks if we're running in a terminal that supports colors
func isTerminal() bool {
	// Simple check for Windows - check if TERM is set or if we're in a console
	if os.Getenv("TERM") != "" && os.Getenv("TERM") != "dumb" {
		return true
	}
	// Check for common CI environments where we might not want colors
	if os.Getenv("CI") != "" || os.Getenv("NO_COLOR") != "" {
		return false
	}
	return true
}

// colorize wraps text in color codes if colors are enabled
func colorize(color, text string) string {
	if !useColors {
		return text
	}
	return color + text + colorReset
}

// Success prints a success message with checkmark
func Success(msg string) {
	if quietMode {
		return
	}
	fmt.Fprintf(Out, "%s %s\n", colorize(colorGreen, symbolSuccess), msg)
}

// Warning prints a warning message
func Warning(msg string) {
	if quietMode {
		return
	}
	fmt.Fprintf(Out, "%s %s\n", colorize(colorYellow, symbolWarning), msg)
}

// Error prints an error message
func Error(msg string) {
	fmt.Fprintf(Err, "%s %s\n", colorize(colorRed, symbolError), msg)
}

// Info prints an informational message
func Info(msg string) {
	if quietMode {
		return
	}
	fmt.Fprintln(Out, msg)
}

// Verbose prints only if verbose mode is enabled
func Verbose(msg string) {
	if !verboseMode {
		return
	}
	fmt.Fprintf(Out, "%s\n", colorize(colorGray, msg))
}

// Print prints a message (respects quiet mode)
func Print(msg string) {
	if quietMode {
		return
	}
	fmt.Fprintln(Out, msg)
}

// Printf prints formatted message (respects quiet mode)
func Printf(format string, args ...interface{}) {
	if quietMode {
		return
	}
	fmt.Fprintf(Out, format, args...)
}

// Println prints a message with newline (always, ignores quiet mode)
func Println(msg string) {
	fmt.Fprintln(Out, msg)
}

// Table prints tabular data
func Table(headers []string, rows [][]string) {
	if quietMode {
		return
	}

	if len(headers) == 0 {
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
	for i, h := range headers {
		fmt.Fprintf(Out, "%-*s", widths[i]+2, h)
	}
	fmt.Fprintln(Out)

	// Print separator
	for _, w := range widths {
		fmt.Fprint(Out, strings.Repeat("-", w+2))
	}
	fmt.Fprintln(Out)

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprintf(Out, "%-*s", widths[i]+2, cell)
			}
		}
		fmt.Fprintln(Out)
	}
}
