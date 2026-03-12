//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const maxLogBytes = 1 << 20 // 1 MB

func initLogging() {
	dir, err := os.UserConfigDir()
	if err != nil {
		return // fall back to default (nowhere on windowsgui)
	}

	logDir := filepath.Join(dir, "Web NFC Bridge Connector")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return
	}

	logPath := filepath.Join(logDir, "connector.log")

	// Rotate: if the file exceeds maxLogBytes, rename to .old and start fresh.
	if info, err := os.Stat(logPath); err == nil && info.Size() > maxLogBytes {
		_ = os.Rename(logPath, logPath+".old")
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("--- log init (pid=%d) ---", os.Getpid())

	// Also redirect the child process stdout/stderr to this file when running
	// as watchdog. We redirect os.Stderr so that the watchdog's exec.Command
	// inherits it.
	redirectStderr(f)

	fmt.Fprintf(os.Stderr, "") // ensure stderr fd is valid after redirect
}

// redirectStderr points os.Stderr to the given file so child processes
// spawned with cmd.Stderr = os.Stderr will inherit file-based logging.
func redirectStderr(f *os.File) {
	os.Stderr = f
	os.Stdout = f
}
