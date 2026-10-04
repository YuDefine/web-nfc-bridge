//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
)

// initLogging sends all output to %LOCALAPPDATA%\Web NFC Bridge Connector\connector.log.
// Windows builds use -H=windowsgui, which leaves the process without a console,
// so without this a crash on startup leaves no trace.
func initLogging() {
	if isSupervisedChild() {
		// stdout/stderr are pipes the watchdog copies into its log file; that
		// includes Go runtime crash output, which is written to fd 2.
		return
	}

	// os.UserCacheDir is %LocalAppData% on Windows.
	dir, err := os.UserCacheDir()
	if err != nil {
		return
	}

	w, err := openRotatingLog(filepath.Join(dir, logDirName), maxLogBytes, func(f *os.File) {
		// Unhandled panics and fatal runtime errors write to fd 2, which does
		// not exist under -H=windowsgui; send them to the current log file.
		_ = debug.SetCrashOutput(f, debug.CrashOptions{})
	})
	if err != nil {
		return
	}

	// runWatchdog passes log.Writer() to the child as stdout/stderr, so this
	// process is the only writer of the log file.
	log.SetOutput(w)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("--- log init (pid=%d) ---", os.Getpid())
}
