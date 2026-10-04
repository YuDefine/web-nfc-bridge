package main

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
)

const (
	logDirName  = "Web NFC Bridge Connector"
	logFileName = "connector.log"
	maxLogBytes = 1 << 20 // 1 MB

	// supervisedEnv marks a child process started by the watchdog. The child's
	// stdout/stderr are pipes the watchdog copies into its log file, so the
	// child must not open the file itself.
	supervisedEnv = "NFC_CONNECTOR_SUPERVISED"
)

// rotatingLog appends to dir/connector.log. A write that would grow the file
// past maxBytes first renames it to connector.log.old (replacing the previous
// one), so at most two files are kept however long the process runs.
// It is safe for concurrent use.
type rotatingLog struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	file     *os.File
	size     int64
	// onOpen runs with every newly opened file (Windows uses it to point
	// runtime crash output at the current file).
	onOpen func(*os.File)
}

func openRotatingLog(dir string, maxBytes int64, onOpen func(*os.File)) (*rotatingLog, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	l := &rotatingLog{
		path:     filepath.Join(dir, logFileName),
		maxBytes: maxBytes,
		onOpen:   onOpen,
	}
	if err := l.open(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *rotatingLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil && l.size > 0 && l.size+int64(len(p)) > l.maxBytes {
		l.rotate()
	}
	if l.file == nil {
		if err := l.open(); err != nil {
			return 0, err
		}
	}

	n, err := l.file.Write(p)
	l.size += int64(n)
	return n, err
}

func (l *rotatingLog) open() error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	l.file = f
	l.size = info.Size()
	if l.onOpen != nil {
		l.onOpen(f)
	}
	return nil
}

// rotate must be called with l.mu held. On failure the file stays nil and the
// next Write reopens it.
func (l *rotatingLog) rotate() {
	_ = l.file.Close()
	l.file = nil

	renameErr := os.Rename(l.path, l.path+".old")
	if err := l.open(); err != nil {
		return
	}
	if renameErr != nil && !errors.Is(renameErr, os.ErrNotExist) {
		// Another process still holds the file (e.g. a connector from an older
		// release). Keep appending and retry after another maxBytes instead of
		// attempting a rename on every write.
		l.size = 0
	}
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
