package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadGitHubApp(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "app.pem")
	if err := os.WriteFile(keyFile, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	keyPath, err := canonicalPath(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		env     map[string]string
		want    *GitHubApp
		wantErr string
	}{
		"unset": {},
		"key file": {
			env:  map[string]string{"HENGE_GITHUB_APP_ID": "123", "HENGE_GITHUB_APP_PRIVATE_KEY": keyFile},
			want: &GitHubApp{ID: "123", KeyLocation: keyPath, KeyPath: keyPath},
		},
		"keychain": {
			env:  map[string]string{"HENGE_GITHUB_APP_ID": "123", "HENGE_GITHUB_APP_PRIVATE_KEY": "keychain:henge-github-app", "HENGE_GITHUB_APP_INSTALLATION_ID": "42"},
			want: &GitHubApp{ID: "123", InstallationID: 42, KeyLocation: "keychain:henge-github-app"},
		},
		"id only":            {env: map[string]string{"HENGE_GITHUB_APP_ID": "123"}, wantErr: "set together"},
		"non-numeric id":     {env: map[string]string{"HENGE_GITHUB_APP_ID": "henge", "HENGE_GITHUB_APP_PRIVATE_KEY": keyFile}, wantErr: "numeric"},
		"missing key file":   {env: map[string]string{"HENGE_GITHUB_APP_ID": "123", "HENGE_GITHUB_APP_PRIVATE_KEY": keyFile + ".missing"}, wantErr: "HENGE_GITHUB_APP_PRIVATE_KEY"},
		"empty keychain":     {env: map[string]string{"HENGE_GITHUB_APP_ID": "123", "HENGE_GITHUB_APP_PRIVATE_KEY": "keychain:"}, wantErr: "Keychain"},
		"bad installation":   {env: map[string]string{"HENGE_GITHUB_APP_ID": "123", "HENGE_GITHUB_APP_PRIVATE_KEY": keyFile, "HENGE_GITHUB_APP_INSTALLATION_ID": "x"}, wantErr: "INSTALLATION_ID"},
		"installation alone": {env: map[string]string{"HENGE_GITHUB_APP_INSTALLATION_ID": "42"}, wantErr: "set together"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, key := range []string{"HENGE_GITHUB_APP_ID", "HENGE_GITHUB_APP_PRIVATE_KEY", "HENGE_GITHUB_APP_INSTALLATION_ID"} {
				t.Setenv(key, test.env[key])
			}
			got, err := loadGitHubApp()
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("loadGitHubApp() error = %v, want it to mention %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadGitHubApp() error = %v", err)
			}
			if (got == nil) != (test.want == nil) || (got != nil && *got != *test.want) {
				t.Fatalf("loadGitHubApp() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestLoadHidesGitHubAppKeyFromCodex(t *testing.T) {
	clearConfigEnv(t)
	setRequiredEnv(t)
	home := t.TempDir()
	withWorkingDir(t, home)
	t.Setenv("HENGE_HOME", home)
	keyFile := filepath.Join(t.TempDir(), "app.pem")
	if err := os.WriteFile(keyFile, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HENGE_GITHUB_APP_ID", "123")
	t.Setenv("HENGE_GITHUB_APP_PRIVATE_KEY", keyFile)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GitHubApp == nil || !slices.Contains(cfg.ProtectedPaths, cfg.GitHubApp.KeyPath) {
		t.Fatalf("ProtectedPaths = %v, want the App key %+v", cfg.ProtectedPaths, cfg.GitHubApp)
	}
}

func TestParseCodexEnvRejectsGitHubAppKey(t *testing.T) {
	if _, err := parseCodexEnv("HENGE_GITHUB_APP_PRIVATE_KEY"); err == nil {
		t.Fatal("parseCodexEnv() error = nil, want the App key refused")
	}
}
