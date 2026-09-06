// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package termout

import (
	"fmt"
	"io"
	"os"
	"strings"

	"melovian/internal/brand"
)

const (
	reset   = "\x1b[0m"
	bold    = "\x1b[1m"
	dim     = "\x1b[2m"
	red     = "\x1b[31m"
	green   = "\x1b[32m"
	yellow  = "\x1b[33m"
	blue    = "\x1b[34m"
	magenta = "\x1b[35m"
	cyan    = "\x1b[36m"
	white   = "\x1b[97m"
)

var stdout = io.Writer(os.Stdout)

func SetOutput(w io.Writer) {
	stdout = w
}

func Enabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func paint(code, text string) string {
	if !Enabled() {
		return text
	}
	return code + text + reset
}

func Label(text string) string {
	return paint(cyan, text)
}

func Value(text string) string {
	return paint(white, text)
}

func Success(text string) string {
	return paint(green, text)
}

func Warn(text string) string {
	return paint(yellow, text)
}

func Error(text string) string {
	return paint(red, text)
}

func Dim(text string) string {
	return paint(dim, text)
}

func Bold(text string) string {
	return paint(bold, text)
}

func Banner() {
	title := paint(magenta+bold, brand.Name)
	subtitle := Dim("music server")
	_, _ = fmt.Fprintf(stdout, "\n  %s  %s\n\n", title, subtitle)
}

func Line(label, value string) {
	_, _ = fmt.Fprintf(stdout, "  %s %s\n", Label(label+":"), Value(value))
}

func Note(text string) {
	_, _ = fmt.Fprintf(stdout, "  %s %s\n", Warn("!"), text)
}

func OK(text string) {
	_, _ = fmt.Fprintf(stdout, "  %s %s\n", Success("+"), text)
}

func Fail(text string) {
	_, _ = fmt.Fprintf(stdout, "  %s %s\n", Error("x"), text)
}

func HelpTitle(text string) {
	_, _ = fmt.Fprintln(stdout, Bold(text))
}

func HelpSection(text string) {
	_, _ = fmt.Fprintf(stdout, "\n%s\n", Bold(text))
}

func HelpFlag(name, description string) {
	_, _ = fmt.Fprintf(stdout, "  %s\n    %s\n", Label(name), Dim(description))
}

func HelpText(text string) {
	for line := range strings.SplitSeq(strings.TrimRight(text, "\n"), "\n") {
		_, _ = fmt.Fprintf(stdout, "%s\n", line)
	}
}
