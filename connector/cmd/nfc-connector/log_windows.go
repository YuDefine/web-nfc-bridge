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
		// stdout/stderr are already the watchdog's log file (inherited handles),
		// including Go runtime crash output.
		return
	}

	// os.UserCacheDir is %LocalAppData% on Windows.
	dir, err := os.UserCacheDir()
	if err != nil {
		return
	}

	f, err := openRotatingLog(filepath.Join(dir, logDirName), maxLogBytes)
	if err != nil {
		return
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("--- log init (pid=%d) ---", os.Getpid())

	// The watchdog passes os.Stdout/os.Stderr to its child, so the child's
	// output (and runtime crash output) lands in the same file.
	os.Stdout = f
	os.Stderr = f

	// Unhandled panics and fatal runtime errors write to fd 2, which does not
	// exist under -H=windowsgui; send them to the log file as well.
	_ = debug.SetCrashOutput(f, debug.CrashOptions{})
}
