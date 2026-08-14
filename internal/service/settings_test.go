package service

import (
	"context"
	"testing"

	"github.com/chiheng-app/chiheng/internal/store"
)

func TestSettings_whenDefaults(t *testing.T) {
	t.Parallel()
	s := NewSettings(openStore(t))

	prefs, err := s.GetPreferences(context.Background())
	if err != nil {
		t.Fatalf("GetPreferences() error = %v", err)
	}
	if prefs.Locale != "zh-CN" || prefs.Theme != "system" {
		t.Fatalf("GetPreferences() = %+v, want defaults", prefs)
	}
}

func TestSettings_whenSaveAndRead(t *testing.T) {
	t.Parallel()
	s := NewSettings(openStore(t))

	want := Preferences{Locale: "en-US", Theme: "dark", RefreshInterval: 30, DisclaimerVersion: "2.0", LogLevel: "debug"}
	if err := s.SavePreferences(context.Background(), want); err != nil {
		t.Fatalf("SavePreferences() error = %v", err)
	}
	got, err := s.GetPreferences(context.Background())
	if err != nil {
		t.Fatalf("GetPreferences() error = %v", err)
	}
	if got.Locale != want.Locale || got.Theme != want.Theme || got.RefreshInterval != want.RefreshInterval {
		t.Fatalf("GetPreferences() = %+v, want %+v", got, want)
	}
}

func TestFileSecretStore_whenSetGet(t *testing.T) {
	t.Parallel()
	fs, err := NewFileSecretStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileSecretStore() error = %v", err)
	}
	ctx := context.Background()

	if err := fs.Set(ctx, "ai.api_key", "sk-test-value"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	value, err := fs.Get(ctx, "ai.api_key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value != "sk-test-value" {
		t.Fatalf("Get() = %q, want sk-test-value", value)
	}
}

func TestFileSecretStore_whenMissing(t *testing.T) {
	t.Parallel()
	fs, err := NewFileSecretStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileSecretStore() error = %v", err)
	}
	_, err = fs.Get(context.Background(), "missing")
	if err != store.ErrNotFound {
		t.Fatalf("Get() error = %v, want store.ErrNotFound", err)
	}
}

func TestFileSecretStore_whenDelete(t *testing.T) {
	t.Parallel()
	fs, err := NewFileSecretStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileSecretStore() error = %v", err)
	}
	ctx := context.Background()
	if err := fs.Set(ctx, "k", "v"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := fs.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := fs.Get(ctx, "k"); err != store.ErrNotFound {
		t.Fatalf("Get() after delete = %v, want store.ErrNotFound", err)
	}
}

func TestSettings_whenAIProfileNoSecret(t *testing.T) {
	t.Parallel()
	s := NewSettings(openStore(t))
	profile, err := s.GetAIProfile(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetAIProfile() error = %v", err)
	}
	if profile.HasSecret || profile.Available {
		t.Fatalf("GetAIProfile() = %+v, want no secret and unavailable", profile)
	}
}
