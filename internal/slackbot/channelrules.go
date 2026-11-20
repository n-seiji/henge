package slackbot

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/n-seiji/ebi-x/internal/codex"
	"github.com/n-seiji/ebi-x/internal/workspace"
)

const maxChannelRulesBytes = 256 << 10

const channelRulesBrokenNotice = "⚠️ チャンネルごとの設定ファイルを読めません（%v）。直すまで、承認チャンネル以外の依頼は始められません。"

type channelRule struct {
	SystemPrompt string    `yaml:"system_prompt"`
	ReadOnly     bool      `yaml:"read_only"`
	GitHub       *bool     `yaml:"github"`
	Repositories *[]string `yaml:"repositories"`
	MCPServers   *[]string `yaml:"mcp_servers"`
	Secrets      []string  `yaml:"secrets"`
	// Every name any channel lists under secrets.
	scopedSecrets []string
}

type channelRulesFile struct {
	Channels map[string]channelRule `yaml:"channels"`
}

func (rule channelRule) githubAllowed() bool {
	return rule.GitHub == nil || *rule.GitHub
}

func (rule channelRule) repoAllowed(repo string) bool {
	if rule.Repositories == nil {
		return true
	}
	return slices.Contains(*rule.Repositories, repo) || slices.Contains(*rule.Repositories, filepath.Base(repo))
}

func (rule channelRule) access(slackLimited bool) codex.Access {
	access := codex.Access{WithoutSlack: slackLimited, NoGitHub: !rule.githubAllowed(), Instructions: rule.instructions(),
		Secrets: rule.Secrets, ScopedSecrets: rule.scopedSecrets}
	if rule.MCPServers != nil {
		access.MCPServers = append([]string{}, (*rule.MCPServers)...)
	}
	return access
}

func (rule channelRule) instructions() string {
	var lines []string
	if text := strings.TrimSpace(rule.SystemPrompt); text != "" {
		lines = append(lines, text)
	}
	if rule.ReadOnly {
		lines = append(lines, "このチャンネルの作業ではファイルを変更できず、ネットワークにも接続できません。")
	}
	if !rule.githubAllowed() {
		lines = append(lines, "このチャンネルの作業では GitHub への push や PR の作成はできません。")
	}
	if rule.Repositories != nil {
		lines = append(lines, "このチャンネルの作業では、決められたリポジトリしか読めません。")
	}
	if rule.MCPServers != nil && len(*rule.MCPServers) == 0 {
		lines = append(lines, "このチャンネルの作業では MCP のツールを使えません。")
	}
	if len(rule.Secrets) > 0 {
		lines = append(lines, "このチャンネルの作業では、コマンドが環境変数 "+strings.Join(rule.Secrets, "・")+" で secret を使えます。値そのものは表示せず、ファイル・コミット・返答にも書かないでください。返答や進捗に値が含まれると、bot が伏せ字に置き換えます。")
	}
	if len(lines) == 0 {
		return ""
	}
	return "# このチャンネルの設定（ebi-x の運用者が決めたもの。依頼より優先する）\n\n" + strings.Join(lines, "\n")
}

func (rule channelRule) limitLease(lease *workspace.Lease) []string {
	if rule.Repositories == nil {
		return nil
	}
	var hidden []string
	var pending []string
	for _, repo := range lease.PendingRepos {
		if rule.repoAllowed(repo) {
			pending = append(pending, repo)
		} else {
			hidden = append(hidden, repo)
		}
	}
	var checkouts []workspace.Checkout
	for _, checkout := range lease.Checkouts {
		if rule.repoAllowed(checkout.Repo) {
			checkouts = append(checkouts, checkout)
		} else {
			hidden = append(hidden, checkout.Repo, checkout.Path, checkout.Path+".gitdir")
		}
	}
	var roots []string
	for _, root := range lease.WritableRoots {
		if !slices.Contains(hidden, root) {
			roots = append(roots, root)
		}
	}
	lease.PendingRepos, lease.Checkouts, lease.WritableRoots = pending, checkouts, roots
	return hidden
}

func (b *Bot) channelRule(channel string) (channelRule, error) {
	rules, err := readChannelRules(b.config.ChannelRulesFile, b.config.CodexEnv)
	if err != nil {
		// Not stopped like the others: the approval channel is where the file gets fixed.
		if b.sharedWritable(channel) {
			return channelRule{}, nil
		}
		// Fails instead of running without rules, which could drop a limit.
		return channelRule{}, err
	}
	rule := rules.Channels[channel]
	for _, other := range rules.Channels {
		for _, name := range other.Secrets {
			if !slices.Contains(rule.scopedSecrets, name) {
				rule.scopedSecrets = append(rule.scopedSecrets, name)
			}
		}
	}
	return rule, nil
}

func (b *Bot) channelRulesDir() string {
	if b.config.ChannelRulesFile == "" {
		return ""
	}
	return filepath.Dir(b.config.ChannelRulesFile)
}

func (b *Bot) channelRulesNotice() string {
	if err := ValidateChannelRules(b.config.ChannelRulesFile, b.config.CodexEnv); err != nil {
		return fmt.Sprintf(channelRulesBrokenNotice, err)
	}
	return ""
}

// Secrets come only from EBIX_CODEX_ENV, which the operator sets outside the
// agent's reach: the file is written by approval-channel work, and naming any
// variable here would let that work read the operator's whole environment.
func readChannelRules(path string, codexEnv []string) (channelRulesFile, error) {
	if path == "" {
		return channelRulesFile{}, nil
	}
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return channelRulesFile{}, nil
	}
	if err != nil {
		return channelRulesFile{}, fmt.Errorf("read channel rules %q: %w", path, err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxChannelRulesBytes+1))
	if err != nil {
		return channelRulesFile{}, fmt.Errorf("read channel rules %q: %w", path, err)
	}
	if len(data) > maxChannelRulesBytes {
		return channelRulesFile{}, fmt.Errorf("channel rules %q exceed %d bytes", path, maxChannelRulesBytes)
	}
	var rules channelRulesFile
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	// Unknown fields are rejected, not ignored: a misspelled limit would silently do nothing.
	decoder.KnownFields(true)
	if err := decoder.Decode(&rules); err != nil && !errors.Is(err, io.EOF) {
		return channelRulesFile{}, fmt.Errorf("parse channel rules %q: %w", path, err)
	}
	for channel, rule := range rules.Channels {
		for _, name := range rule.Secrets {
			if slices.Contains(codex.SecretsUnavailableToCommands(), name) || !slices.Contains(codexEnv, name) {
				return channelRulesFile{}, fmt.Errorf("channel rules %q: secret %s of %s is not one of EBIX_CODEX_ENV", path, name, channel)
			}
		}
	}
	return rules, nil
}

func ValidateChannelRules(path string, codexEnv []string) error {
	_, err := readChannelRules(path, codexEnv)
	return err
}
