package log

import (
	"os"

	"github.com/fatih/color"
)

// Predefine color functions for different log levels
var (
	InfoColor    = color.New(color.FgCyan)
	SuccessColor = color.New(color.FgGreen)
	WarnColor    = color.New(color.FgYellow)
	ErrorColor   = color.New(color.FgRed)
	DetailColor  = color.New(color.FgWhite) // For less important details
)

// Info prints an informational message (cyan).
func Info(format string, a ...interface{}) {
	InfoColor.Fprintf(os.Stdout, format+"\n", a...)
}

// Success prints a success message (green).
func Success(format string, a ...interface{}) {
	SuccessColor.Fprintf(os.Stdout, format+"\n", a...)
}

// Warn prints a warning message (yellow) to stderr.
func Warn(format string, a ...interface{}) {
	WarnColor.Fprintf(os.Stderr, "Warning: "+format+"\n", a...)
}

// Error prints an error message (red) to stderr.
func Error(format string, a ...interface{}) {
	ErrorColor.Fprintf(os.Stderr, "Error: "+format+"\n", a...)
}

// Fatal prints an error message (red) to stderr and exits with status 1.
func Fatal(format string, a ...interface{}) {
	Error(format, a...)
	os.Exit(1)
}

// Detail prints less important details (usually white/default).
func Detail(format string, a ...interface{}) {
	DetailColor.Fprintf(os.Stdout, format+"\n", a...)
}

// Printf allows printing with a specific color.
func Printf(c *color.Color, format string, a ...interface{}) {
	c.Printf(format, a...)
}

// Fprintf allows printing to a specific writer with a specific color.
func Fprintf(w *os.File, c *color.Color, format string, a ...interface{}) {
	c.Fprintf(w, format, a...)
}
