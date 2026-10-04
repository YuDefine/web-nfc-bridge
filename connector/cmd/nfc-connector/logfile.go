package main

import (
	"os"
	"path/filepath"
)

const (
	logDirName  = "Web NFC Bridge Connector"
	logFileName = "connector.log"
	maxLogBytes = 1 << 20 // 1 MB

	// supervisedEnv marks a child process started by the watchdog. The child
	// inherits the watchdog's log file as stdout/stderr, so it must not open
	// (or rotate) the file a second time.
	supervisedEnv = "NFC_CONNECTOR_SUPERVISED"
)

// openRotatingLog opens dir/connector.log for appending. When the existing
// file is larger than maxBytes it is first renamed to connector.log.old, so at
// most two files (current + previous) are kept.
func openRotatingLog(dir string, maxBytes int64) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	logPath := filepath.Join(dir, logFileName)
	if info, err := os.Stat(logPath); err == nil && info.Size() > maxBytes {
		_ = os.Rename(logPath, logPath+".old")
	}

	return os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

// supervisedChildEnv returns the environment for a watchdog child process.
func supervisedChildEnv(environ []string) []string {
	env := make([]string, 0, len(environ)+1)
	env = append(env, environ...)
	return append(env, supervisedEnv+"=1")
}

func isSupervisedChild() bool {
	return os.Getenv(supervisedEnv) == "1"
}
