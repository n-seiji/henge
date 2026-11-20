// Package ebix holds the files `ebi-x init` writes, so that a binary
// installed without the repository can still start a new agent directory.
package ebix

import _ "embed"

// The repository copies stay the source; embedding them avoids a second copy
// that could drift.

//go:embed .env.example
var EnvTemplate []byte

//go:embed examples/slack-app-manifest.json
var SlackAppManifest []byte
