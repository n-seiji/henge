package codex

import (
	"cmp"
	"log"
	"slices"
	"strings"
)

// Shorter values are not granted: replacing every occurrence of a short
// string would mangle ordinary text, and leaving it unredacted would let it
// reach Slack.
const minSecretBytes = 8

// SecretsUnavailableToCommands are names a channel may not grant: they are
// the Codex credentials, which the commands it runs must never hold.
func SecretsUnavailableToCommands() []string {
	return slices.Clone(shellEnvExcluded)
}

func grantedSecrets(env, names []string) map[string]string {
	granted := make(map[string]string)
	for _, name := range names {
		if slices.Contains(shellEnvExcluded, name) {
			continue
		}
		value, ok := lookupEnv(env, name)
		switch {
		case !ok:
			log.Printf("codex exec: secret %s is not in the environment Codex inherits; add it to EBIX_CODEX_ENV", name)
		case len(value) < minSecretBytes:
			log.Printf("codex exec: secret %s is shorter than %d bytes and is not given to commands", name, minSecretBytes)
		default:
			granted[name] = value
		}
	}
	return granted
}

func lookupEnv(env []string, name string) (string, bool) {
	value, found := "", false
	for _, entry := range env {
		if key, v, ok := strings.Cut(entry, "="); ok && key == name {
			value, found = v, true
		}
	}
	return value, found
}

// Longer values go first so a value containing another is replaced whole.
func secretRedactor(granted map[string]string) func(string) string {
	if len(granted) == 0 {
		return func(text string) string { return text }
	}
	names := make([]string, 0, len(granted))
	for name := range granted {
		names = append(names, name)
	}
	slices.SortFunc(names, func(a, b string) int {
		return cmp.Or(cmp.Compare(len(granted[b]), len(granted[a])), cmp.Compare(a, b))
	})
	pairs := make([]string, 0, 2*len(names))
	for _, name := range names {
		pairs = append(pairs, granted[name], "[secret:"+name+"]")
	}
	return strings.NewReplacer(pairs...).Replace
}

func (a Activity) redacted(redact func(string) string) Activity {
	a.Command, a.Query, a.Tool, a.Current = redact(a.Command), redact(a.Query), redact(a.Tool), redact(a.Current)
	if a.Paths != nil {
		paths := make([]string, len(a.Paths))
		for i, path := range a.Paths {
			paths[i] = redact(path)
		}
		a.Paths = paths
	}
	return a
}

func (r *TurnResult) redact(redact func(string) string) {
	for i, message := range r.Messages {
		r.Messages[i] = redact(message)
	}
	r.Err = redact(r.Err)
}
