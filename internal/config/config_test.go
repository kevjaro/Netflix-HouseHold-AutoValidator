package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	yamlContent := `email:
  imap: "imap.test.com:993"
  login: "test@example.com"
  password: "testpass"
  mailbox: "INBOX"
targetFrom: "info@test.com"
targetSubject: "Test Subject"
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	_ = tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Email.Imap != "imap.test.com:993" {
		t.Errorf("Expected imap 'imap.test.com:993', got '%s'", cfg.Email.Imap)
	}

	if cfg.TargetFrom != "info@test.com" {
		t.Errorf("Expected targetFrom 'info@test.com', got '%s'", cfg.TargetFrom)
	}
}

func TestLoadDefaultsReconcileInterval(t *testing.T) {
	t.Setenv("EMAIL_RECONCILE_INTERVAL", "")

	path := writeTempConfig(t, `email:
  imap: "imap.test.com:993"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Email.ReconcileInterval != "5m" {
		t.Fatalf("ReconcileInterval = %q, want %q", cfg.Email.ReconcileInterval, "5m")
	}
}

func TestLoadRejectsInvalidReconcileInterval(t *testing.T) {
	path := writeTempConfig(t, `email:
  reconcileInterval: "never"
`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load() expected an error for an invalid reconcile interval")
	}
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	path := t.TempDir() + "/config.yaml"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
