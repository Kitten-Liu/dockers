package main

import (
	"log"
	"strings"
)

const (
	ANSI_COLOR_NOTICE = "\033[1;36m"
	ANSI_COLOR_INFO   = "\033[1;34m"
	ANSI_COLOR_WARN   = "\033[1;33m"
	ANSI_COLOR_ERROR  = "\033[1;31m"
	ANSI_COLOR_DEBUG  = "\033[0;36m"
	ANSI_COLOR_RESET  = "\033[0m"
)

func LogNotice(msg string, v ...any) {
	LogWithAnsiColor(ANSI_COLOR_NOTICE, msg, v...)
}

func LogInfo(msg string, v ...any) {
	LogWithAnsiColor(ANSI_COLOR_INFO, msg, v...)
}

func LogWarn(msg string, v ...any) {
	LogWithAnsiColor(ANSI_COLOR_WARN, msg, v...)
}

func LogError(msg string, v ...any) {
	LogWithAnsiColor(ANSI_COLOR_ERROR, msg, v...)
}

func LogDebug(msg string, v ...any) {
	LogWithAnsiColor(ANSI_COLOR_DEBUG, msg, v...)
}

func LogWithAnsiColor(ansiCode string, msg string, v ...any) {
	s := []string{ansiCode, msg, ANSI_COLOR_RESET}
	temp := strings.Join(s, "")
	log.Printf(temp, v...)
}
