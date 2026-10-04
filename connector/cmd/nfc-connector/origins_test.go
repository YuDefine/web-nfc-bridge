package main

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func withExtraAllowedOrigins(t *testing.T, value string) {
	t.Helper()
	previous := extraAllowedOrigins
	extraAllowedOrigins = value
	t.Cleanup(func() { extraAllowedOrigins = previous })
}

func TestResolveAllowedOriginsDefaultsToPublicOrigins(t *testing.T) {
	withExtraAllowedOrigins(t, "")

	got := resolveAllowedOrigins("")

	want := strings.Split(publicAllowedOrigins, ",")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected public origins only\n got: %v\nwant: %v", got, want)
	}
}

func TestResolveAllowedOriginsAppendsBuildTimeExtras(t *testing.T) {
	withExtraAllowedOrigins(t, "https://downstream.example,https://other.example")

	got := resolveAllowedOrigins("")

	want := append(strings.Split(publicAllowedOrigins, ","), "https://downstream.example", "https://other.example")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected public + extra origins\n got: %v\nwant: %v", got, want)
	}
}

func TestResolveAllowedOriginsEnvironmentReplacesDefaults(t *testing.T) {
	withExtraAllowedOrigins(t, "https://downstream.example")

	got := resolveAllowedOrigins(" https://only.example , ,https://only.example,http://localhost:* ")

	want := []string{"https://only.example", "http://localhost:*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected environment list trimmed and deduplicated\n got: %v\nwant: %v", got, want)
	}
}

func TestResolveAllowedOriginsIgnoresBlankEnvironment(t *testing.T) {
	withExtraAllowedOrigins(t, "")

	got := resolveAllowedOrigins("   ")

	if !slices.Contains(got, "https://nfc.yudefine.com.tw") {
		t.Fatalf("expected blank environment to fall back to defaults, got %v", got)
	}
}

func TestResolveAllowedOriginsDeduplicatesExtrasAlreadyPublic(t *testing.T) {
	withExtraAllowedOrigins(t, "https://nfc.yudefine.com.tw")

	got := resolveAllowedOrigins("")

	if len(got) != len(strings.Split(publicAllowedOrigins, ",")) {
		t.Fatalf("expected duplicate extra origin to be dropped, got %v", got)
	}
}
