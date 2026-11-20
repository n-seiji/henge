// Package secretref reads secrets that a setting names instead of holding:
// "keychain:<service>" for the macOS Keychain and "op://vault/item/field" for
// 1Password. Anything else is the secret itself.
package secretref

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"slices"
	"strings"
	"time"
)

const (
	KeychainPrefix    = "keychain:"
	onePasswordPrefix = "op://"
)

// Long enough for someone to answer a 1Password or macOS approval prompt.
const readTimeout = 2 * time.Minute

func IsReference(value string) bool {
	return strings.HasPrefix(value, KeychainPrefix) || strings.HasPrefix(value, onePasswordPrefix)
}

// Check rejects a malformed reference without reading it, so a configuration
// check never prompts.
func Check(name, value string) error {
	if service, ok := strings.CutPrefix(value, KeychainPrefix); ok && strings.TrimSpace(service) == "" {
		return fmt.Errorf("%s: %q names no Keychain service", name, value)
	}
	if path, ok := strings.CutPrefix(value, onePasswordPrefix); ok {
		parts := strings.Split(path, "/")
		if len(parts) < 3 || slices.Contains(parts, "") {
			return fmt.Errorf("%s: %q is not a 1Password secret reference (op://vault/item/field)", name, value)
		}
	}
	return nil
}

// name only labels log lines and errors; the value never appears in either.
func Resolve(ctx context.Context, name, value string) (string, error) {
	secret, err := ResolveBytes(ctx, name, value)
	return string(secret), err
}

func ResolveBytes(ctx context.Context, name, value string) ([]byte, error) {
	if err := Check(name, value); err != nil {
		return nil, err
	}
	if service, ok := strings.CutPrefix(value, KeychainPrefix); ok {
		log.Printf("reading %s from the Keychain; allow henge if macOS asks", name)
		secret, err := readKeychain(service)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return secret, nil
	}
	if strings.HasPrefix(value, onePasswordPrefix) {
		log.Printf("reading %s from 1Password; approve henge if 1Password asks", name)
		secret, err := readOnePassword(ctx, value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return secret, nil
	}
	return []byte(value), nil
}

// The op command rather than the 1Password SDK: it already handles both a
// service account token and the desktop app's sign-in, and adds no
// dependency.
func readOnePassword(ctx context.Context, ref string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "op", "read", "--no-newline", ref)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("read %s: the 1Password CLI (op) is not installed", ref)
		}
		if text := strings.TrimSpace(stderr.String()); text != "" {
			return nil, fmt.Errorf("read %s: %s: %w", ref, text, err)
		}
		return nil, fmt.Errorf("read %s: %w", ref, err)
	}
	if stdout.Len() == 0 {
		return nil, fmt.Errorf("read %s: the field is empty", ref)
	}
	return stdout.Bytes(), nil
}
