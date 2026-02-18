package utils

import (
	"fmt"
	"time"
)

// ANSI colour codes — make terminal output easier to read while debugging
const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
)

func ts() string {
	return time.Now().Format("15:04:05")
}

func Info(format string, a ...interface{}) {
	fmt.Printf("%s[%s] [INFO]  %s%s\n", blue, ts(), fmt.Sprintf(format, a...), reset)
}

func Success(format string, a ...interface{}) {
	fmt.Printf("%s[%s] [OK]    %s%s\n", green, ts(), fmt.Sprintf(format, a...), reset)
}

func Warn(format string, a ...interface{}) {
	fmt.Printf("%s[%s] [WARN]  %s%s\n", yellow, ts(), fmt.Sprintf(format, a...), reset)
}

func Error(format string, a ...interface{}) {
	fmt.Printf("%s[%s] [ERROR] %s%s\n", red, ts(), fmt.Sprintf(format, a...), reset)
}

func Section(title string) {
	fmt.Printf("\n%s[%s] ══════════ %s ══════════%s\n\n", cyan, ts(), title, reset)
}