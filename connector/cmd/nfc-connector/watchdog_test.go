package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const crashingChildEnv = "NFC_CONNECTOR_TEST_CRASHING_CHILD"

// TestCrashingChildHelper is the child process for
// TestSuperviseOnceCapturesChildCrashOutput; it does nothing in a normal run.
func TestCrashingChildHelper(t *testing.T) {
	if os.Getenv(crashingChildEnv) != "1" {
		t.Skip("helper process only")
	}
	if !isSupervisedChild() {
		os.Exit(3)
	}
	os.Stderr.WriteString("child: starting\n")
	panic("child crashed")
}

func TestSuperviseOnceCapturesChildCrashOutput(t *testing.T) {
	t.Setenv(crashingChildEnv, "1")
	dir := t.TempDir()
	out, err := openRotatingLog(dir, maxLogBytes, nil)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer out.Close()

	err = superviseOnce(os.Args[0], []string{"-test.run=^TestCrashingChildHelper$"}, out)

	if err == nil {
		t.Fatal("expected the crashing child to exit with an error")
	}
	logged, readErr := os.ReadFile(filepath.Join(dir, logFileName))
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, want := range []string{"child: starting", "panic: child crashed", "goroutine "} {
		if !strings.Contains(string(logged), want) {
			t.Fatalf("expected child output %q in log, got:\n%s", want, logged)
		}
	}
}
