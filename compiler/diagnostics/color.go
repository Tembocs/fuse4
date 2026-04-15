package diagnostics

import (
	"os"
	"runtime"
	"syscall"
)

// ANSI escape codes for terminal coloring.
const (
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorGreen  = "\033[32m"
	ColorCyan   = "\033[36m"
	ColorDim    = "\033[2m"
	ColorBold   = "\033[1m"
	ColorReset  = "\033[0m"

	ColorBoldRed    = "\033[1;31m"
	ColorBoldYellow = "\033[1;33m"
	ColorBoldGreen  = "\033[1;32m"
	ColorBoldCyan   = "\033[1;36m"
)

// UseColor reports whether colored output should be used on the given file.
// It respects the NO_COLOR convention (https://no-color.org/) and checks
// whether the file is a terminal.
func UseColor(f *os.File) bool {
	// NO_COLOR convention: if set (to any value), disable color.
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}

	// FORCE_COLOR overrides terminal detection.
	if v := os.Getenv("FORCE_COLOR"); v != "" && v != "0" {
		return true
	}

	if !isTerminal(f) {
		return false
	}

	// On Windows, attempt to enable ANSI escape processing.
	if runtime.GOOS == "windows" {
		enableWindowsANSI(f)
	}

	return true
}

// Colorize wraps s in the given ANSI code if color is enabled.
func Colorize(s, code string, enabled bool) string {
	if !enabled || code == "" {
		return s
	}
	return code + s + ColorReset
}

// SeverityColor returns the ANSI color code for a diagnostic severity.
func SeverityColor(sev Severity) string {
	switch sev {
	case Error:
		return ColorBoldRed
	case Warning:
		return ColorBoldYellow
	case Note:
		return ColorBoldCyan
	default:
		return ""
	}
}

// isTerminal reports whether f is connected to a terminal.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}

	if runtime.GOOS == "windows" {
		return isWindowsTerminal(f)
	}

	// Unix: check for character device.
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// isWindowsTerminal checks if the file handle is a Windows console.
func isWindowsTerminal(f *os.File) bool {
	var mode uint32
	handle := syscall.Handle(f.Fd())
	err := syscall.GetConsoleMode(handle, &mode)
	return err == nil
}

// enableWindowsANSI enables virtual terminal processing on Windows 10+.
func enableWindowsANSI(f *os.File) {
	const enableVirtualTerminalProcessing = 0x0004

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	handle := syscall.Handle(f.Fd())
	var mode uint32
	if syscall.GetConsoleMode(handle, &mode) == nil {
		mode |= enableVirtualTerminalProcessing
		setConsoleMode.Call(uintptr(handle), uintptr(mode))
	}

}
