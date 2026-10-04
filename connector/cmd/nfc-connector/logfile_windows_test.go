//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// On Windows a file cannot be renamed while another handle without
// FILE_SHARE_DELETE is open, which includes the duplicated handle held for
// runtime crash output. Rotation must still succeed with the real hook.
func TestRotatingLogRotatesWithCrashOutputOnWindows(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { setCrashOutput(nil) })

	l, err := openRotatingLog(dir, 8, setCrashOutput)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer l.file.Close()
	writeString(t, l, "first\n")
	writeString(t, l, "second\n")

	if got := readFile(t, filepath.Join(dir, logFileName+".old")); got != "first\n" {
		t.Fatalf("expected rotation to move the first file to .old, got %q", got)
	}
	if got := readFile(t, filepath.Join(dir, logFileName)); got != "second\n" {
		t.Fatalf("expected the current log to hold only the latest write, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, logFileName)); err != nil {
		t.Fatal(err)
	}
}
