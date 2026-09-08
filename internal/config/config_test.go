package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("format = \"json\"\nrows = [[\"OS\", \"igpu\"]]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Format != "json" || cfg.Rows[0][0] != "os" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestLoadRejectsUnknownSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("rows = [[\"nope\"]]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() accepted an unknown spec")
	}
}

func TestLoadRejectsMoreThanThreeColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("rows = [[\"os\", \"host\", \"cpu\", \"gpu\"]]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() accepted more than three columns")
	}
}

func TestLoadAcceptsCompleteCustomPalette(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := "color_mode = \"custom\"\n" +
		"[colors]\n" +
		"border = \"#111111\"\nbackground = \"#222222\"\nheader = \"#333333\"\nvalue = \"#444444\"\nsitch = \"#555555\"\n" +
		"[colors.titles]\n"
	for spec := range supportedSpecs {
		contents += spec + " = \"#abcdef\"\n"
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("Load() rejected complete custom palette: %v", err)
	}
}

func TestLoadAcceptsPartialCustomPalette(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := "color_mode = \"custom\"\n" +
		"[colors]\n" +
		"border = \"#111111\"\nbackground = \"\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() rejected partial custom palette: %v", err)
	}
	if cfg.Colors.Background != "" {
		t.Fatalf("expected empty custom background to stay empty, got %q", cfg.Colors.Background)
	}
}

func TestLoadRejectsInvalidCustomColor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := "color_mode = \"custom\"\n" +
		"[colors]\n" +
		"border = \"not-a-color\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() accepted an invalid custom color")
	}
}

func TestLoadAcceptsLogoOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := "logo = \"arch\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() rejected known logo id: %v", err)
	}
	if cfg.Logo != "arch" {
		t.Fatalf("logo id not preserved: %q", cfg.Logo)
	}
}

func TestLoadRejectsMissingLogoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := "logo_file = \"/nope/this/does/not/exist\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() accepted a missing logo_file path")
	}
}

func TestLoadAcceptsReadableLogoFile(t *testing.T) {
	dir := t.TempDir()
	art := filepath.Join(dir, "logo.txt")
	if err := os.WriteFile(art, []byte("my\nlogo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.toml")
	contents := "logo_file = \"" + art + "\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() rejected readable logo_file: %v", err)
	}
	if cfg.LogoFile != art {
		t.Fatalf("logo_file path not preserved: %q", cfg.LogoFile)
	}
}
