//go:build !windows

package main

func initLogging() {
	// On non-Windows platforms, the connector runs under launchd/systemd
	// which captures stdout/stderr. No file logging needed.
}
