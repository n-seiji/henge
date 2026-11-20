package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/n-seiji/henge"
)

const initArg = "init"

const manifestFile = "slack-app-manifest.json"

const initNextStepsTemplate = `
Next steps:
  1. Create a Slack App "From a manifest" with slack-app-manifest.json, install it,
     and put the Bot Token (xoxb-) and the Socket Mode App Token (xapp-) in .env.
  2. Fill in the allowed user and channel IDs in .env.
  3. Log in to Codex: codex login
  4. Start the agent in this directory: henge
`

const initNextStepsAnswered = `
Next steps:
  1. Log in to Codex: codex login
  2. Start the agent in this directory: henge
  3. Invite the bot to the channel you entered and mention it there.
     Other people and channels are approved from that channel in Slack.
`

type initPrompter interface {
	Line(question string) (string, error)
	Secret(question string) (string, error)
}

// initAgentDir never overwrites, so running it again in a configured
// directory cannot wipe tokens already written to .env. With a nil prompter
// .env is the commented template.
func initAgentDir(dir string, prompter initPrompter, out io.Writer) error {
	manifestPath := filepath.Join(dir, manifestFile)
	if err := writeNew(manifestPath, henge.SlackAppManifest, 0o644, out); err != nil {
		return err
	}
	envPath := filepath.Join(dir, ".env")
	if _, err := os.Lstat(envPath); err == nil {
		fmt.Fprintf(out, "kept existing %s\n", envPath)
		fmt.Fprint(out, initNextStepsTemplate)
		return nil
	}
	if prompter == nil {
		if err := writeNew(envPath, henge.EnvTemplate, 0o600, out); err != nil {
			return err
		}
		fmt.Fprint(out, initNextStepsTemplate)
		return nil
	}

	fmt.Fprintf(out, "\nCreate a Slack App \"From a manifest\" with %s and install it to your workspace.\n", manifestPath)
	values, err := askSlackSettings(prompter, out)
	if err != nil {
		return err
	}
	if err := writeNew(envPath, minimalEnv(values), 0o600, out); err != nil {
		return err
	}
	fmt.Fprint(out, initNextStepsAnswered)
	return nil
}

type envValue struct{ key, value string }

func askSlackSettings(prompter initPrompter, out io.Writer) ([]envValue, error) {
	botToken, err := ask(prompter.Secret, out, "Bot User OAuth Token (xoxb-…, OAuth & Permissions)", prefixed("xoxb-"))
	if err != nil {
		return nil, err
	}
	appToken, err := ask(prompter.Secret, out, "App-Level Token with connections:write (xapp-…, Basic Information)", prefixed("xapp-"))
	if err != nil {
		return nil, err
	}
	userID, err := ask(prompter.Line, out, "Your Slack member ID (U…, profile menu > Copy member ID)", prefixed("U"))
	if err != nil {
		return nil, err
	}
	channelID, err := ask(prompter.Line, out, "Channel ID where you talk to the bot and approve others (C… or G…, channel details)", prefixed("C", "G"))
	if err != nil {
		return nil, err
	}
	// The owner's channel is the approval channel rather than an allowed
	// channel, so every other person and channel can be approved from Slack
	// without editing .env again.
	return []envValue{
		{"SLACK_BOT_TOKEN", botToken},
		{"SLACK_APP_TOKEN", appToken},
		{"SLACK_ALLOWED_USER_IDS", userID},
		{"SLACK_APPROVAL_CHANNEL_ID", channelID},
		{"SLACK_APPROVER_USER_IDS", userID},
		{"SLACK_ADMIN_USER_ID", userID},
	}, nil
}

func ask(read func(string) (string, error), out io.Writer, question string, valid func(string) error) (string, error) {
	for {
		answer, err := read(question + ": ")
		if err != nil {
			return "", err
		}
		answer = strings.TrimSpace(answer)
		if err := valid(answer); err != nil {
			fmt.Fprintf(out, "  %v\n", err)
			continue
		}
		return answer, nil
	}
}

func prefixed(prefixes ...string) func(string) error {
	return func(answer string) error {
		for _, prefix := range prefixes {
			if strings.HasPrefix(answer, prefix) && len(answer) > len(prefix) {
				return nil
			}
		}
		return fmt.Errorf("must start with %s", strings.Join(prefixes, " or "))
	}
}

// minimalEnv writes only what has no default, so that a new agent's .env
// stays short; every other setting is documented in .env.example.
func minimalEnv(values []envValue) []byte {
	var b strings.Builder
	for _, v := range values {
		fmt.Fprintf(&b, "%s=%s\n", v.key, v.value)
	}
	return []byte(b.String())
}

func writeNew(path string, data []byte, perm fs.FileMode, out io.Writer) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if errors.Is(err, fs.ErrExist) {
		fmt.Fprintf(out, "kept existing %s\n", path)
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Fprintf(out, "created %s\n", path)
	return nil
}

type terminalPrompter struct {
	in  *bufio.Reader
	fd  int
	out io.Writer
}

// newTerminalPrompter returns nil when stdin is not a terminal, so that
// scripted runs get the template instead of blocking on questions.
func newTerminalPrompter() initPrompter {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil
	}
	return &terminalPrompter{in: bufio.NewReader(os.Stdin), fd: fd, out: os.Stdout}
}

func (p *terminalPrompter) Line(question string) (string, error) {
	fmt.Fprint(p.out, question)
	return p.in.ReadString('\n')
}

func (p *terminalPrompter) Secret(question string) (string, error) {
	fmt.Fprint(p.out, question)
	secret, err := term.ReadPassword(p.fd)
	fmt.Fprintln(p.out)
	return string(secret), err
}
