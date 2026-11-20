package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/n-seiji/henge"
	"github.com/n-seiji/henge/internal/config"
)

const initArg = "init"

const manifestFile = "slack-app-manifest.json"

const (
	envFile      = ".env"
	appKeyFile   = "github-app.pem"
	appWaitLimit = 15 * time.Minute
)

const initNextStepsTemplate = `
Next steps:
  1. Create a Slack App "From a manifest" with slack-app-manifest.json, install it,
     and put the Bot Token (xoxb-) and the Socket Mode App Token (xapp-) in .env.
  2. Fill in the approval channel and approvers, or the allowed users and
     channels, in henge.yaml.
  3. Log in to Codex: codex login
  4. Start the agent in this directory: henge
`

const initNextStepsAnswered = `
Next steps:
  1. Log in to Codex if you have not: codex login
  2. Start the agent in this directory: henge
  3. Invite the bot to the channel you entered and mention it there.
     Other people and channels are approved from that channel in Slack.
`

type initPrompter interface {
	Line(question string) (string, error)
	Secret(question string) (string, error)
}

// Replaced in tests, which cannot reach GitHub.
var newGitHubApp = func(ctx context.Context, req githubAppRequest, out io.Writer) (createdGitHubApp, error) {
	return createGitHubApp(ctx, req, openBrowser, out)
}

type initAnswers struct {
	botToken, appToken string
	userID, channelID  string
	writableRoots      []string
	app                *createdGitHubApp
}

// initAgentDir never overwrites, so running it again in a configured
// directory cannot wipe tokens already written. It asks questions only when
// neither .env nor henge.yaml exists yet; otherwise, or with a nil prompter,
// it writes the commented templates that are missing.
func initAgentDir(dir string, prompter initPrompter, out io.Writer) error {
	manifestPath := filepath.Join(dir, manifestFile)
	if err := writeNew(manifestPath, henge.SlackAppManifest, 0o644, out); err != nil {
		return err
	}
	envPath := filepath.Join(dir, envFile)
	settingsPath := filepath.Join(dir, config.FileName)
	if prompter == nil || exists(envPath) || exists(settingsPath) {
		if err := writeNew(envPath, henge.EnvTemplate, 0o600, out); err != nil {
			return err
		}
		if err := writeNew(settingsPath, henge.SettingsTemplate, 0o600, out); err != nil {
			return err
		}
		fmt.Fprint(out, initNextStepsTemplate)
		return nil
	}

	fmt.Fprintf(out, "\nCreate a Slack App from the manifest, and install it to your workspace:\n  %s\n(or paste %s under \"From a manifest\")\n\n", slackAppCreateURL(), manifestPath)
	answers, err := askInitAnswers(prompter, out)
	if err != nil {
		return err
	}
	if answers.app != nil {
		if err := writeNew(filepath.Join(dir, appKeyFile), []byte(answers.app.PEM), 0o600, out); err != nil {
			return err
		}
	}
	if err := writeNew(envPath, answers.env(), 0o600, out); err != nil {
		return err
	}
	settings, err := answers.settings()
	if err != nil {
		return err
	}
	if err := writeNew(settingsPath, settings, 0o600, out); err != nil {
		return err
	}
	if answers.app != nil {
		fmt.Fprintf(out, "\nInstall the GitHub App on the repositories henge works on:\n  %s/installations/new\n", answers.app.HTMLURL)
	}
	if _, err := exec.LookPath("codex"); err != nil {
		fmt.Fprintln(out, "\nCodex CLI was not found on PATH; install it before starting henge.")
	}
	fmt.Fprint(out, initNextStepsAnswered)
	return nil
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func slackAppCreateURL() string {
	var manifest bytes.Buffer
	_ = json.Compact(&manifest, henge.SlackAppManifest)
	return "https://api.slack.com/apps?new_app=1&manifest_json=" + url.QueryEscape(manifest.String())
}

func askInitAnswers(prompter initPrompter, out io.Writer) (initAnswers, error) {
	var a initAnswers
	var err error
	if a.botToken, err = ask(prompter.Secret, out, "Bot User OAuth Token (xoxb-…, OAuth & Permissions)", prefixed("xoxb-")); err != nil {
		return a, err
	}
	if a.appToken, err = ask(prompter.Secret, out, "App-Level Token with connections:write (xapp-…, Basic Information)", prefixed("xapp-")); err != nil {
		return a, err
	}
	if a.userID, err = ask(prompter.Line, out, "Your Slack member ID (U…, profile menu > Copy member ID)", prefixed("U")); err != nil {
		return a, err
	}
	if a.channelID, err = ask(prompter.Line, out, "Channel ID where you talk to the bot and approve others (C… or G…, channel details)", prefixed("C", "G")); err != nil {
		return a, err
	}
	roots, err := ask(prompter.Line, out, "Repositories henge may work on (comma-separated directories, empty for none)", directories)
	if err != nil {
		return a, err
	}
	a.writableRoots = splitComma(roots)
	createApp, err := ask(prompter.Line, out, "Create a GitHub App so henge can push and open pull requests? (y/N)", yesNo)
	if err != nil {
		return a, err
	}
	if strings.EqualFold(createApp, "y") {
		owner, err := ask(prompter.Line, out, "GitHub organization that owns the repositories (empty for your own account)", anything)
		if err != nil {
			return a, err
		}
		defaultName := "henge-" + randomSuffix()
		name, err := ask(prompter.Line, out, fmt.Sprintf("App name, unique on GitHub (empty for %s)", defaultName), anything)
		if err != nil {
			return a, err
		}
		if name == "" {
			name = defaultName
		}
		ctx, cancel := context.WithTimeout(context.Background(), appWaitLimit)
		defer cancel()
		app, err := newGitHubApp(ctx, githubAppRequest{name: name, owner: owner, web: githubWebURL, api: githubAPIURL}, out)
		if err != nil {
			return a, fmt.Errorf("create GitHub App: %w", err)
		}
		a.app = &app
	}
	return a, nil
}

// Only the secrets go to .env; everything else is in henge.yaml.
func (a initAnswers) env() []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "SLACK_BOT_TOKEN=%s\nSLACK_APP_TOKEN=%s\n", a.botToken, a.appToken)
	if a.app != nil {
		fmt.Fprintf(&b, "HENGE_GITHUB_APP_PRIVATE_KEY=%s\n", appKeyFile)
	}
	return []byte(b.String())
}

// The owner's channel is the approval channel rather than an allowed channel,
// so every other person and channel can be approved from Slack without
// editing henge.yaml again.
func (a initAnswers) settings() ([]byte, error) {
	type slackSettings struct {
		ApprovalChannel string   `yaml:"approval_channel"`
		Approvers       []string `yaml:"approvers,flow"`
		AllowedUsers    []string `yaml:"allowed_users,flow"`
	}
	type sandboxSettings struct {
		WritableRoots []string `yaml:"writable_roots,flow"`
	}
	type githubSettings struct {
		AppID string `yaml:"app_id"`
	}
	doc := struct {
		Slack   slackSettings    `yaml:"slack"`
		Sandbox *sandboxSettings `yaml:"sandbox,omitempty"`
		GitHub  *githubSettings  `yaml:"github,omitempty"`
	}{Slack: slackSettings{ApprovalChannel: a.channelID, Approvers: []string{a.userID}, AllowedUsers: []string{a.userID}}}
	if len(a.writableRoots) > 0 {
		doc.Sandbox = &sandboxSettings{WritableRoots: a.writableRoots}
	}
	if a.app != nil {
		doc.GitHub = &githubSettings{AppID: strconv.FormatInt(a.app.ID, 10)}
	}
	body, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return append([]byte("# Written by henge init. Every setting: https://github.com/n-seiji/henge/blob/main/examples/henge.yaml\n"), body...), nil
}

func directories(answer string) error {
	for _, item := range splitComma(answer) {
		path := item
		if rest, ok := strings.CutPrefix(item, "~/"); ok {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			path = filepath.Join(home, rest)
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("%s is not a directory", item)
		}
	}
	return nil
}

func yesNo(answer string) error {
	switch strings.ToLower(answer) {
	case "", "y", "n":
		return nil
	}
	return errors.New("answer y or n")
}

func anything(string) error { return nil }

func splitComma(value string) []string {
	var items []string
	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func randomSuffix() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
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
