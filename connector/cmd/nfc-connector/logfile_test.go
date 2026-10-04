package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRotatingLogCreatesDirectoryAndFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), logDirName)

	f, err := openRotatingLog(dir, maxLogBytes)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer f.Close()

	if f.Name() != filepath.Join(dir, logFileName) {
		t.Fatalf("expected log at %s, got %s", filepath.Join(dir, logFileName), f.Name())
	}
}

func TestOpenRotatingLogAppendsBelowLimit(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)
	if err := os.WriteFile(logPath, []byte("previous\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := openRotatingLog(dir, 1024)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	if _, err := f.WriteString("next\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "previous\nnext\n" {
		t.Fatalf("expected appended content, got %q", content)
	}
	if _, err := os.Stat(logPath + ".old"); !os.IsNotExist(err) {
		t.Fatalf("expected no rotation below limit, stat err=%v", err)
	}
}

func TestOpenRotatingLogRotatesAboveLimit(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)
	oversized := strings.Repeat("x", 2048)
	if err := os.WriteFile(logPath, []byte(oversized), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath+".old", []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := openRotatingLog(dir, 1024)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	f.Close()

	current, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != 0 {
		t.Fatalf("expected fresh log after rotation, got %d bytes", len(current))
	}
	previous, err := os.ReadFile(logPath + ".old")
	if err != nil {
		t.Fatal(err)
	}
	if string(previous) != oversized {
		t.Fatalf("expected .old to hold the rotated log, got %d bytes", len(previous))
	}
}

func TestSupervisedChildEnvMarksChildWithoutMutatingParent(t *testing.T) {
	parent := []string{"PATH=/bin", "NFC_CONNECTOR_ADDR=127.0.0.1:42619"}

	child := supervisedChildEnv(parent)

	if len(parent) != 2 {
		t.Fatalf("parent environment was mutated: %v", parent)
	}
	if got := child[len(child)-1]; got != supervisedEnv+"=1" {
		t.Fatalf("expected child env to end with %s=1, got %q", supervisedEnv, got)
	}
	if child[0] != parent[0] || child[1] != parent[1] {
		t.Fatalf("expected parent variables to be preserved, got %v", child)
	}
}

func TestIsSupervisedChild(t *testing.T) {
	t.Setenv(supervisedEnv, "")
	if isSupervisedChild() {
		t.Fatal("expected unsupervised when variable is empty")
	}

	t.Setenv(supervisedEnv, "1")
	if !isSupervisedChild() {
		t.Fatal("expected supervised when variable is 1")
	}
}
