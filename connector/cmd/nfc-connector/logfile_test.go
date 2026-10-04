package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func writeString(t *testing.T, l *rotatingLog, s string) {
	t.Helper()
	if _, err := l.Write([]byte(s)); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func TestOpenRotatingLogCreatesDirectoryAndFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), logDirName)

	l, err := openRotatingLog(dir, maxLogBytes, nil)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer l.file.Close()
	writeString(t, l, "hello\n")

	if got := readFile(t, filepath.Join(dir, logFileName)); got != "hello\n" {
		t.Fatalf("expected log content, got %q", got)
	}
}

func TestRotatingLogAppendsBelowLimit(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)
	if err := os.WriteFile(logPath, []byte("previous\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	l, err := openRotatingLog(dir, 1024, nil)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer l.file.Close()
	writeString(t, l, "next\n")

	if got := readFile(t, logPath); got != "previous\nnext\n" {
		t.Fatalf("expected appended content, got %q", got)
	}
	if _, err := os.Stat(logPath + ".old"); !os.IsNotExist(err) {
		t.Fatalf("expected no rotation below limit, stat err=%v", err)
	}
}

func TestRotatingLogRotatesOversizedFileOnFirstWrite(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)
	oversized := strings.Repeat("x", 2048)
	if err := os.WriteFile(logPath, []byte(oversized), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath+".old", []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	l, err := openRotatingLog(dir, 1024, nil)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer l.file.Close()
	writeString(t, l, "fresh\n")

	if got := readFile(t, logPath); got != "fresh\n" {
		t.Fatalf("expected fresh log after rotation, got %q", got)
	}
	if got := readFile(t, logPath+".old"); got != oversized {
		t.Fatalf("expected .old to hold the rotated log, got %d bytes", len(got))
	}
}

func TestRotatingLogRotatesWhileRunning(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)

	l, err := openRotatingLog(dir, 10, nil)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer l.file.Close()

	writeString(t, l, "aaaaaa\n") // 7 bytes
	writeString(t, l, "bbbbbb\n") // would reach 14 > 10: rotate first
	writeString(t, l, "cc\n")     // 10 bytes, still within limit
	writeString(t, l, "dddddd\n") // would reach 17 > 10: rotate again

	if got := readFile(t, logPath); got != "dddddd\n" {
		t.Fatalf("expected current log to hold the latest write, got %q", got)
	}
	if got := readFile(t, logPath+".old"); got != "bbbbbb\ncc\n" {
		t.Fatalf("expected .old to hold the previous generation, got %q", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected exactly two log files, got %d", len(entries))
	}
}

func TestRotatingLogWritesOversizedEntryToEmptyFile(t *testing.T) {
	dir := t.TempDir()

	l, err := openRotatingLog(dir, 4, nil)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer l.file.Close()
	writeString(t, l, "longer than the limit\n")

	if got := readFile(t, filepath.Join(dir, logFileName)); got != "longer than the limit\n" {
		t.Fatalf("expected oversized entry to be written, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, logFileName+".old")); !os.IsNotExist(err) {
		t.Fatalf("expected no rotation of an empty file, stat err=%v", err)
	}
}

func TestRotatingLogCallsOnOpenForEachFile(t *testing.T) {
	dir := t.TempDir()
	var opened []string

	l, err := openRotatingLog(dir, 4, func(f *os.File) { opened = append(opened, f.Name()) })
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer l.file.Close()
	writeString(t, l, "1234")
	writeString(t, l, "5678")

	if len(opened) != 2 {
		t.Fatalf("expected onOpen for initial open and rotation, got %v", opened)
	}
	for _, name := range opened {
		if name != filepath.Join(dir, logFileName) {
			t.Fatalf("expected onOpen with current log path, got %s", name)
		}
	}
}

func TestRotatingLogConcurrentWritesKeepEveryLine(t *testing.T) {
	dir := t.TempDir()

	l, err := openRotatingLog(dir, 1<<20, nil)
	if err != nil {
		t.Fatalf("openRotatingLog: %v", err)
	}
	defer l.file.Close()

	const writers, lines = 8, 100
	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range lines {
				_, _ = l.Write([]byte("line\n"))
			}
		}()
	}
	wg.Wait()

	got := readFile(t, filepath.Join(dir, logFileName))
	if strings.Count(got, "line\n") != writers*lines || len(got) != writers*lines*len("line\n") {
		t.Fatalf("expected %d intact lines, got %d bytes", writers*lines, len(got))
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
