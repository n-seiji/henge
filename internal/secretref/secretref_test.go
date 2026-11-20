package secretref

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeOP puts an op command on PATH that prints value for want and fails
// for any other reference.
func fakeOP(t *testing.T, want, value string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n[ \"$1 $2 $3\" = \"read --no-newline " + want + "\" ] || { echo \"[ERROR] not found\" >&2; exit 1; }\nprintf '%s' '" + value + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "op"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestResolve(t *testing.T) {
	fakeOP(t, "op://vault/slack/bot", "xoxb-from-1password")
	tests := []struct {
		name, value, want, wantErr string
	}{
		{name: "plain value", value: "xoxb-plain", want: "xoxb-plain"},
		{name: "1Password reference", value: "op://vault/slack/bot", want: "xoxb-from-1password"},
		{name: "missing 1Password item", value: "op://vault/slack/missing", wantErr: "not found"},
		{name: "reference without a field", value: "op://vault/slack", wantErr: "op://vault/item/field"},
		{name: "empty Keychain service", value: "keychain:", wantErr: "names no Keychain service"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Resolve(context.Background(), "SLACK_BOT_TOKEN", test.value)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) || !strings.Contains(err.Error(), "SLACK_BOT_TOKEN") {
					t.Fatalf("Resolve(%q) error = %v, want one naming the setting and %q", test.value, err, test.wantErr)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("Resolve(%q) = %q, %v; want %q", test.value, got, err, test.want)
			}
		})
	}
}
