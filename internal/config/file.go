package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FileName is the operator's settings file, read from the working directory
// like .env. Codex can neither read nor write it, because it holds the
// sandbox's own limits.
const FileName = "henge.yaml"

type fileSettings struct {
	Slack struct {
		ApprovalChannel            string   `yaml:"approval_channel"`
		Approvers                  []string `yaml:"approvers"`
		AllowedUsers               []string `yaml:"allowed_users"`
		AllowedChannels            []string `yaml:"allowed_channels"`
		AllowWorkflows             *bool    `yaml:"allow_workflows"`
		AllowedWorkflows           []string `yaml:"allowed_workflows"`
		Admin                      string   `yaml:"admin"`
		ThreadSubscriptionReaction string   `yaml:"thread_subscription_reaction"`
		ThreadSubscriptionTTL      string   `yaml:"thread_subscription_ttl"`
	} `yaml:"slack"`
	Codex struct {
		Command         string   `yaml:"command"`
		Model           string   `yaml:"model"`
		Timeout         string   `yaml:"timeout"`
		MaxParallelWork *int     `yaml:"max_parallel_work"`
		PassEnv         []string `yaml:"pass_env"`
	} `yaml:"codex"`
	Sandbox struct {
		WritableRoots    []string `yaml:"writable_roots"`
		DeniedReadPaths  []string `yaml:"denied_read_paths"`
		AllowedReadPaths []string `yaml:"allowed_read_paths"`
		CheckoutIdleTTL  string   `yaml:"checkout_idle_ttl"`
	} `yaml:"sandbox"`
	GitHub struct {
		APIURL            string `yaml:"api_url"`
		AppID             string `yaml:"app_id"`
		AppInstallationID string `yaml:"app_installation_id"`
	} `yaml:"github"`
}

// fileKeys maps each setting that moved out of the environment to its place
// in henge.yaml. The rest of Load still reads these names, so a value keeps
// one meaning and one validation wherever it comes from.
var fileKeys = map[string]string{
	"SLACK_APPROVAL_CHANNEL_ID":          "slack.approval_channel",
	"SLACK_APPROVER_USER_IDS":            "slack.approvers",
	"SLACK_ALLOWED_USER_IDS":             "slack.allowed_users",
	"SLACK_ALLOWED_CHANNEL_IDS":          "slack.allowed_channels",
	"SLACK_ALLOW_WORKFLOWS":              "slack.allow_workflows",
	"SLACK_ALLOWED_WORKFLOW_IDS":         "slack.allowed_workflows",
	"SLACK_ADMIN_USER_ID":                "slack.admin",
	"SLACK_THREAD_SUBSCRIPTION_REACTION": "slack.thread_subscription_reaction",
	"SLACK_THREAD_SUBSCRIPTION_TTL":      "slack.thread_subscription_ttl",
	"CODEX_COMMAND":                      "codex.command",
	"CODEX_MODEL":                        "codex.model",
	"CODEX_TIMEOUT":                      "codex.timeout",
	"CODEX_MAX_PARALLEL_WORK":            "codex.max_parallel_work",
	"HENGE_CODEX_ENV":                    "codex.pass_env",
	"HENGE_WRITABLE_ROOTS":               "sandbox.writable_roots",
	"HENGE_DENIED_READ_PATHS":            "sandbox.denied_read_paths",
	"HENGE_ALLOWED_READ_PATHS":           "sandbox.allowed_read_paths",
	"HENGE_CHECKOUT_IDLE_TTL":            "sandbox.checkout_idle_ttl",
	"HENGE_GITHUB_API_URL":               "github.api_url",
	"HENGE_GITHUB_APP_ID":                "github.app_id",
	"HENGE_GITHUB_APP_INSTALLATION_ID":   "github.app_installation_id",
}

func (s *fileSettings) values() map[string]string {
	list := func(items []string) string { return strings.Join(items, ",") }
	values := map[string]string{
		"SLACK_APPROVAL_CHANNEL_ID":          s.Slack.ApprovalChannel,
		"SLACK_APPROVER_USER_IDS":            list(s.Slack.Approvers),
		"SLACK_ALLOWED_USER_IDS":             list(s.Slack.AllowedUsers),
		"SLACK_ALLOWED_CHANNEL_IDS":          list(s.Slack.AllowedChannels),
		"SLACK_ALLOWED_WORKFLOW_IDS":         list(s.Slack.AllowedWorkflows),
		"SLACK_ADMIN_USER_ID":                s.Slack.Admin,
		"SLACK_THREAD_SUBSCRIPTION_REACTION": s.Slack.ThreadSubscriptionReaction,
		"SLACK_THREAD_SUBSCRIPTION_TTL":      s.Slack.ThreadSubscriptionTTL,
		"CODEX_COMMAND":                      s.Codex.Command,
		"CODEX_MODEL":                        s.Codex.Model,
		"CODEX_TIMEOUT":                      s.Codex.Timeout,
		"HENGE_CODEX_ENV":                    list(s.Codex.PassEnv),
		"HENGE_WRITABLE_ROOTS":               list(s.Sandbox.WritableRoots),
		"HENGE_DENIED_READ_PATHS":            list(s.Sandbox.DeniedReadPaths),
		"HENGE_ALLOWED_READ_PATHS":           list(s.Sandbox.AllowedReadPaths),
		"HENGE_CHECKOUT_IDLE_TTL":            s.Sandbox.CheckoutIdleTTL,
		"HENGE_GITHUB_API_URL":               s.GitHub.APIURL,
		"HENGE_GITHUB_APP_ID":                s.GitHub.AppID,
		"HENGE_GITHUB_APP_INSTALLATION_ID":   s.GitHub.AppInstallationID,
	}
	if s.Slack.AllowWorkflows != nil {
		values["SLACK_ALLOW_WORKFLOWS"] = strconv.FormatBool(*s.Slack.AllowWorkflows)
	}
	if s.Codex.MaxParallelWork != nil {
		values["CODEX_MAX_PARALLEL_WORK"] = strconv.Itoa(*s.Codex.MaxParallelWork)
	}
	return values
}

// applySettingsFile fills the moved settings that the environment and .env
// left unset. The environment still wins so that a one-off override needs no
// file edit.
func applySettingsFile(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var settings fileSettings
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	// A misspelled key would otherwise silently leave a limit at its default.
	decoder.KnownFields(true)
	if err := decoder.Decode(&settings); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	for name, value := range settings.values() {
		if value == "" {
			continue
		}
		if _, exists := os.LookupEnv(name); exists {
			continue
		}
		if err := os.Setenv(name, value); err != nil {
			return fmt.Errorf("apply %s: %w", path, err)
		}
	}
	return nil
}

// Hosts set up before henge.yaml keep working; the log names where each
// setting now belongs.
func logSettingsInEnvironment() {
	var moved []string
	for name, key := range fileKeys {
		if _, exists := os.LookupEnv(name); exists {
			moved = append(moved, name+" -> "+key)
		}
	}
	if len(moved) > 0 {
		slices.Sort(moved)
		log.Printf("set in the environment or .env; move to %s (.env is for secrets): %s", FileName, strings.Join(moved, ", "))
	}
}

// fileError names the henge.yaml key of a rejected setting, wherever its
// value came from, since henge.yaml is where it is meant to be set.
func fileError(err error) error {
	if err == nil {
		return nil
	}
	for name, key := range fileKeys {
		if strings.HasPrefix(err.Error(), name) {
			return fmt.Errorf("%s %s: %w", FileName, key, err)
		}
	}
	return err
}
