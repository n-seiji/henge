# Installation and settings

How to get ebi-x running, and the settings the operator decides.

## 0. Installation

To install a release binary, use `install.sh` (the README's steps). It checks the checksum, places the binary in `~/.local/bin`, and on macOS re-signs it with the hardened runtime. `EBIX_VERSION=v1.2.3` picks a version and `EBIX_BIN_DIR` the location.

To build from source, install Go (`mise install` sets it up) and run `mise run build`. If you build with `go build` directly, on macOS also run `codesign --force --sign - --options runtime ebi-x` (see [7. Watching PRs](#the-works-github-authentication-github-app) for why).

Run from a terminal, `ebi-x init` asks for the Slack tokens and IDs and writes only the items without defaults to `.env`. Run without a terminal (from a script), it writes `.env.example` with every item described. Neither overwrites existing files.

**One directory, one agent**: ebi-x reads `.env` in the directory it starts in, and keeps `data/` and `.codex/` in `EBIX_HOME` (by default that directory). To run another agent with another Slack app, use another directory and start again from `ebi-x init`. The binary can be shared.

### Keeping it running

There are templates in `examples/`. Both use the agent's directory as the working directory and pass a `PATH` where `codex` and `git` can be found.

| OS | File | Switching to a new binary |
| --- | --- | --- |
| Linux (systemd user service) | `examples/ebi-x.service` | run `install.sh`, then `systemctl --user reload ebi-x` |
| macOS (launchd) | `examples/ebi-x.plist` | run `install.sh`, then `launchctl kill HUP gui/$(id -u)/local.ebi-x` |

The switch behaves like the reload in [9. Operations](#9-operations).

### Releases

Pushing a tag starting with `v` makes GitHub Actions run the tests, then attach Linux and macOS (amd64, arm64) binaries and `checksums.txt` to the release. `install.sh` relies on these file names.

## 1. Slack app

On Slack's app creation screen, choose "From a manifest" and paste the `slack-app-manifest.json` that `ebi-x init` writes (`examples/slack-app-manifest.json` in the repository) to set up everything below at once. Without the manifest, configure the same by hand.

| Item | Setting | Used for |
| --- | --- | --- |
| Socket Mode | on | receiving events without a public URL |
| Bot Token Scopes | `app_mentions:read` `chat:write` `reactions:read` `reactions:write` `channels:history` `files:read` `files:write` `im:history` `assistant:write` `commands` | mentions, posts and reactions; fetching images and other files attached to requests; attaching deliverables; DMs and the agent view; shortcuts |
| Event Subscriptions | `app_mention` `message.channels` `message.groups` `message.im` `app_home_opened` | mentions, replies in subscribed threads (including private channels), DMs, the Home tab |
| App Home | Home Tab and Messages Tab on, "Allow users to send Slash commands and messages from the messages tab" on | the Home tab and DMs |
| Interactivity & Shortcuts | Interactivity on, Message Shortcut "Ask ebi-x" (Callback ID `ebix_request`) | requests from a message |
| Agents & AI Apps | switch to Agent App (`features.agent_view` in the manifest) | the agent view in the sidebar and the Messages tab |

- To use `@channel` with reaction watches ([usage.md](usage.md#reaction-watches)), add `channels:read` and `users:read` (plus `groups:read` for private channels), and for user groups such as `@dev_team`, also `usergroups:read`. The manifest includes them.
- Private channels and Slack Connect channels need `groups:read`, `groups:history` and `message.groups` (included in the manifest).
- Switching to an Agent App cannot be undone, and needs a paid Slack plan.
- After changing scopes, reinstall the app in the workspace. If the Bot Token changes, update `.env` too.

After installing, get the Bot Token (`xoxb-`) and the App Token for Socket Mode (`xapp-`, `connections:write`).

## 2. `.env`

Fill in the `.env` that `ebi-x init` wrote (to run from the repository, copy `.env.example`). The main items are below; `.env.example` also describes each one.

| Variable | Default | Contents |
| --- | --- | --- |
| `SLACK_BOT_TOKEN` / `SLACK_APP_TOKEN` | none (required) | Slack tokens |
| `SLACK_ALLOWED_USER_IDS` / `SLACK_ALLOWED_CHANNEL_IDS` | none | users and channels always allowed. Without an approval channel, at least one of each is required |
| `SLACK_APPROVAL_CHANNEL_ID` / `SLACK_APPROVER_USER_IDS` | none | the approval channel and approvers (see below) |
| `SLACK_ALLOW_WORKFLOWS` / `SLACK_ALLOWED_WORKFLOW_IDS` | `false` / none | whether to accept mentions from Workflow Builder, and which workflows |
| `SLACK_ADMIN_USER_ID` | none | the user shown to people without permission as the one to ask. Unset, the words "the ebi-x administrator" are shown |
| `SLACK_THREAD_SUBSCRIPTION_REACTION` / `SLACK_THREAD_SUBSCRIPTION_TTL` | none / `336h` | the reaction name and duration for thread subscriptions, which also handle replies without a mention. Without a reaction they are off |
| `CODEX_COMMAND` | `codex` | the Codex executable (no arguments) |
| `CODEX_MODEL` | Codex's default | the model to use |
| `CODEX_TIMEOUT` | `30m` | the time limit for one Codex run |
| `CODEX_MAX_PARALLEL_WORK` | `3` | how many threads run at once |
| `EBIX_HOME` | `.` | the directory holding `data/` and `.codex/` |
| `EBIX_WRITABLE_ROOTS` | none | directories Codex can write to (the repositories to work on, and so on) |
| `EBIX_DENIED_READ_PATHS` | none | paths Codex must not read, in addition to the default credential locations |
| `EBIX_ALLOWED_READ_PATHS` | none | default-denied credential locations the work needs (such as `~/.config/gcloud`; see below) |
| `EBIX_CODEX_ENV` | none | names of environment variables passed to Codex in addition to the allowlist (see below) |
| `EBIX_CHECKOUT_IDLE_TTL` | `120h` | how long an unused per-thread clone is kept before it is removed |
| `EBIX_GITHUB_TOKEN` / `EBIX_GITHUB_API_URL` | none | the token and API URL for PR watching (see below) |
| `EBIX_GITHUB_APP_ID` / `EBIX_GITHUB_APP_PRIVATE_KEY` / `EBIX_GITHUB_APP_INSTALLATION_ID` | none | the GitHub App used for the work's GitHub authentication (see below) |

Times use the process's local time zone, so to run on Japan time, start it with `TZ=Asia/Tokyo`.

### Removed settings

If you reuse an old `.env`, delete these items.

| Variable | Handling |
| --- | --- |
| `SLACK_ALLOW_ALL_PUBLIC_CHANNELS` | `true` stops startup. Allow channels one by one with `SLACK_ALLOWED_CHANNEL_IDS` or the approval channel |
| `CODEX_MAX_PARALLEL_PLAN` | unused. Concurrency is `CODEX_MAX_PARALLEL_WORK` |
| `CODEX_WORK_MODEL` | unused. The model is `CODEX_MODEL` |
| `EBIX_SHARED_WRITE_CHANNEL_IDS` | unused. Playbooks and global memory are changed from the approval channel |

## 3. Who and which channels can use it

Whether someone can use ebi-x depends on the fixed permissions in `.env` and what approvers allowed in the approval channel. Both the person and the channel must be allowed. A DM is a conversation between the person and ebi-x only, so it needs no channel permission once the person is allowed.

When a person, channel or workflow without permission mentions ebi-x, it replies in the same thread with what lacks permission, and asks them to ask `SLACK_ADMIN_USER_ID`.

```text
🔒 You do not have permission to use ebi-x. If you want to use it, ask @admin.
```

### Approval channel

Set `SLACK_APPROVAL_CHANNEL_ID` and `SLACK_APPROVER_USER_IDS` and invite the bot, and requests from those not yet allowed can be approved in Slack. No extra scopes are needed.

```text
#sales
@alice    @ebi-x summarize last week's sales
  └ @ebi-x  🔒 You do not have permission to use ebi-x yet, so I asked an approver (approval request).
            Please wait! Once approved, I'll start with this request automatically.

#ebi-x-approvals
@ebi-x    🔐 Request to use ebi-x / Subject: channel #sales
  └ @admin   @ebi-x 許可
  └ @ebi-x  ✅ Channel #sales: approved by @admin.

#sales (the original thread)
  └ @ebi-x  ✅ Channel #sales was approved. Starting the request.
```

- The approver mentions `@ebi-x 許可` or `@ebi-x allow`, or `@ebi-x 拒否` or `@ebi-x deny`, in the request's thread. Only people in `SLACK_APPROVER_USER_IDS` can do this, and the first decision wins. Those denied are turned away without asking from then on.
- Once the needed permissions are in place, waiting requests start automatically. Requests older than 24 hours do not start; ebi-x asks for a new mention instead.
- In the approval channel, mentions in a request's thread, and `@ebi-x 許可`, `@ebi-x 拒否` and `@ebi-x 残り` (usage) outside threads, are approval operations. Other mentions are taken as work requests from someone allowed to use ebi-x.
- Only requests in the approval channel can change playbooks and global memory. Requests in every channel read them, so changes are kept where approvers can see them.
- Decisions are kept in `data/state/approvals.json`. To revoke one, stop ebi-x and delete the entry from this file.

### Mentions from workflows

With `SLACK_ALLOW_WORKFLOWS=true`, mentions from Workflow Builder's "Send a message" are accepted, but only when the Slack event's `workflow_id` is in `SLACK_ALLOWED_WORKFLOW_IDS` or was approved in the approval channel. Anyone in the workspace can create a workflow, so without limiting by ID, people's permissions could be bypassed. Workflows are also subject to channel permissions. Posts from other bots are ignored.

## 4. Repositories to work on

List the directories Codex should write to in `EBIX_WRITABLE_ROOTS`. Top-level directories of Git repositories among them are worked on through per-thread clones, so the original repository is not modified ([design.md](design.md#isolating-working-areas)). A clone's `origin` is the original repository's `origin`, so working branches can be pushed.

Known credential locations such as `~/.ssh`, `~/.aws`, `~/.config/gh` and `~/.netrc` are hidden from Codex by default (see [design.md](design.md#what-is-protected-from-codex) for the list). Those needed for pushing and signed commits are checked per repository at startup and made readable, and the startup log shows which paths were exempted and why.

- For SSH remotes, `~/.ssh` is exempted. For HTTPS, depending on git's credential helper, `~/.git-credentials` (`store`), `~/.config/gh` (`gh`), or `~/.netrc` without a helper. Helpers such as `osxkeychain` need no path, so nothing is exempted.
- With `commit.gpgsign=true`, `~/.ssh` or `~/.gnupg` is exempted depending on `gpg.format`.
- Only the global and system git settings are checked, because the original repository's `.git/config` does not carry over to thread clones.
- If `EBIX_WRITABLE_ROOTS` lists a repository's parent directory, the repositories under it are not checked. To push from them over SSH, write `EBIX_ALLOWED_READ_PATHS=~/.ssh`.
- When the work uses `gcloud`, `bq`, `kubectl`, a private npm registry and so on, write something like `EBIX_ALLOWED_READ_PATHS=~/.config/gcloud,~/.kube,~/.npmrc`. Only entries in the default list are allowed; anything else is a startup error.

Do not put push credentials in `EBIX_DENIED_READ_PATHS`. To prevent pushes, narrow the credentials of the original repository's `origin`.

If `EBIX_WRITABLE_ROOTS` overlaps paths ebi-x manages (the directories in `data/`, `.env`, `EBIX_HOME/.codex`, `CODEX_HOME`, `EBIX_DENIED_READ_PATHS`), it refuses to start.

## 5. Action rules

By default, Codex confirms with the requester before these actions:

- Confirm first: pushing to a remote; creating, merging or closing PRs and issues; sending to Slack, email or external services; deleting or publishing files and resources; actions that cost money; changes to production
- Do without asking: investigating and reading; editing and committing in the thread's working copy; tests and builds; producing deliverables

To change them, write them in Markdown in `$EBIX_HOME/data/rules.md` (see `examples/rules.md`). If the file exists it replaces the defaults, and takes effect from new threads without a restart. It can be up to 16KB, and Codex cannot change it.

The rules are instructions to the model. What can be written and which paths cannot be read are enforced separately by the sandbox.

### Per-channel settings

Write them per channel ID in `$EBIX_HOME/data/channel-rules/channels.yaml` (see `examples/channels.yaml`).

| Item | Contents |
| --- | --- |
| `system_prompt` | instructions added to Codex's developer instructions |
| `read_only` | `true` makes files unwritable and the network unreachable |
| `github` | `false` withholds GitHub credentials (no pushes or PRs) |
| `repositories` | the repositories that can be read (directory names or paths). Other repositories and their clones cannot be read |
| `mcp_servers` | the names of usable MCP servers. `[]` turns them all off |

### Global settings

Write them in `$EBIX_HOME/data/channel-rules/settings.yaml` (see `examples/settings.yaml`). Ask in the approval channel, for example `@ebi-x make the watch reaction ok_hand and the time 9`, and ebi-x shows the change before rewriting the file. Writing it by hand is fine too.

| Item | Contents |
| --- | --- |
| `language` | `ja` (default) or `en`: the language of ebi-x's own fixed Slack messages, such as notices, buttons and the Home tab. Codex still replies in the requester's language |
| `done_watch.reaction` / `done_watch.hour` | the reaction a reaction watch waits for, and the hour to notify ([usage.md](usage.md#reaction-watches)) |
| `sql_write_servers` | the names of MCP servers allowed to run SQL that writes or locks |

If the file cannot be read, ebi-x runs with the defaults and says so in its replies in the approval channel.

- It is reread for every request, so no restart is needed.
- Only requests from the approval channel can change it.
- If the file cannot be read (a YAML error, or an unknown item), requests outside the approval channel do not start ([design.md](design.md#per-channel-settings)).
- Turning off the Slack MCP in private and Slack Connect channels cannot be lifted with `mcp_servers`.

## 6. Playbooks

A playbook is Markdown describing how to do a specific kind of work. Put it directly under `data/playbooks/`, and the list is passed to Codex with a thread's first request and read when the work matches.

```markdown
---
name: release-notes
description: state concretely which requests it is for
---

# Steps …
```

- The frontmatter needs `name` and `description`, and each file can be up to 64KB. `README.md` is not listed.
- There are examples in `examples/playbooks/`.

| Example | Contents |
| --- | --- |
| `playbook-from-thread.md` | turn what was done in a thread into a playbook, or fix an existing one |
| `request-discovery.md` | interview about an improvement request one question at a time and summarize it |

Only the approval channel can change playbooks from Slack. `data/` is outside Git, so keep playbooks safe in the running environment.

## 7. Watching PRs

This works with no extra settings. At startup, for the https origins on GitHub among `EBIX_WRITABLE_ROOTS`, the bot reads the token stored in git's credential helper with `git credential fill` and uses it to read PRs. It does not use `gh auth`. If the origin is SSH or there are no credentials, watching is off.

- Only to use a different token, write it in `EBIX_GITHUB_TOKEN`. Read access to PRs, checks and commit statuses is enough. It is not passed to Codex. A fine-grained token limited to the watched repositories is recommended.
- For GitHub Enterprise Server, write `https://<host>/api/v3` in `EBIX_GITHUB_API_URL`.
- Each PR costs about 180 API calls an hour. With a token limited to 5,000 calls an hour, watch at most about 20 PRs at once.
- Comments and reviews by the token's own owner do not resume work.

### The work's GitHub authentication (GitHub App)

With `EBIX_GITHUB_APP_ID` and `EBIX_GITHUB_APP_PRIVATE_KEY` set, the bot mints an installation token with the App's private key for each run, passes it to Codex, and revokes it when the run ends. The token works only during the run, and even if revoking fails it expires one hour after minting. If CODEX_TIMEOUT is longer than an hour, the token expires in the middle of long runs. Only the bot uses the private key; Codex never sees it. Without these settings, the host's git credential helper and gh settings are used as before.

- `EBIX_GITHUB_APP_PRIVATE_KEY`: the path of the PEM private key file (`~/` allowed), or `keychain:<service name>` (the macOS Keychain). The file is automatically hidden from Codex.
- `EBIX_GITHUB_APP_INSTALLATION_ID`: not needed if the App has one installation. If it has several, the startup log lists them; write the one to use. The token works only for that installation's (organization's) repositories.
- If minting fails, it is logged and that run continues with the host's git and gh settings.
- When storing the key in the Keychain, let only ebi-x read it: `security add-generic-password -s ebi-x-github-app -a ebi-x -T "$PWD/ebi-x" -w "$(cat app.pem)"`. ebi-x calls the Security framework directly instead of the `security` command, so attempts by Codex's work to read it with `security find-generic-password` are refused (the same goes for running `security` by hand, so check in Keychain Access.app).
  - After creating it, check that `security find-generic-password -s ebi-x-github-app -w` does not print the key without asking. If it does, remove `security` under "Access Control" in Keychain Access.app.
  - After a rebuild, the Keychain sees a different app. So after a reload, at startup you are asked "ebi-x wants to use confidential information stored in your keychain"; press "Always Allow". The key is read only once at startup, and startup waits until then (it is logged).
  - On macOS, `mise run build` adds the hardened runtime (`codesign --sign - --options runtime`), so a debugger, that is Codex's work, cannot attach to ebi-x and read the key from memory. If you build with `go build` directly, run the same `codesign`.

## 8. MCP

Codex's MCP servers are configured in `$EBIX_HOME/.codex/config.toml`. `~/.codex/config.toml` is not read.

MCP servers started with `command` are started through ebi-x's SQL guard. When a tool call carries SQL (an `sql` or `statement` argument, or `query` on a tool whose name contains sql, query or execute), only a single read statement such as SELECT, WITH, SHOW or EXPLAIN that does not write or lock passes; anything else is returned as a tool error. List servers that need to write in `sql_write_servers` in `settings.yaml`.

The guard cannot be put in front of MCP servers connected by `url`. Remove their tools that can write with `enabled_tools` or `disabled_tools`, or with the server's own read-only setting.

When a needed tool is not exposed, Codex tells the requester that it is "not exposed", not that it "cannot connect".

### Slack MCP

MCP servers whose name, `url`, `command` or `args` contain `slack` are turned off in closed channels ([design.md](design.md#closed-channels)). If a server is missing from the startup log line `MCP servers turned off for private and Slack Connect channels: [...]`, include `slack` in its name.

### Environment variables

The environment variables passed to Codex are an allowlist; other variables in `.env` or the shell are not passed (see [design.md](design.md#what-is-protected-from-codex) for the list). The names of variables not passed are logged in one line at startup. When an MCP server or model provider reads an environment variable, both of the following are needed:

1. write the variable name in `EBIX_CODEX_ENV` to pass it to Codex
2. write the name in that server's `env_vars` (or write the value directly in `env`), because stdio MCP servers do not inherit Codex's environment as is

```toml
[mcp_servers.example]
command = "example-mcp"
env_vars = ["EXAMPLE_API_TOKEN"]   # also needs EBIX_CODEX_ENV=EXAMPLE_API_TOKEN
env = { EXAMPLE_REGION = "asia-northeast1" }
```

`SLACK_BOT_TOKEN`, `SLACK_APP_TOKEN`, `EBIX_GITHUB_TOKEN` and `EBIX_GITHUB_APP_PRIVATE_KEY` cannot be listed (startup error). Variables whose names contain `KEY`, `SECRET` or `TOKEN` reach Codex itself but are not passed to the commands Codex runs. So authenticating `git push` or `gh` with `GH_TOKEN` and the like does not work; use a credential helper or a GitHub App ([7. Watching PRs](#the-works-github-authentication-github-app)).

## 9. Operations

- **One process per machine**: starting a second one on the same `data/` is an error. Stop the process with the pid in the message before starting. Do not delete the lock file.
- **Reload**: `mise run reload` (build, then send `SIGHUP` to the running ebi-x) swaps in the new binary without stopping work in progress. Without mise: `go build -o ebi-x ./cmd/ebi-x && codesign --force --sign - --options runtime ebi-x && kill -HUP "$(cat data/state/ebi-x.lock)"` (`codesign` on macOS only).
  - The new binary is first started with `-check-config`; if that fails, nothing is swapped and the old one keeps running. `mise run reload` does not send `SIGHUP` if the build fails.
  - Running work continues, and requests that arrive meanwhile are saved and started automatically after the restart. `.env` is reread too.
  - Mentions arriving during the few seconds of the switch may be missed. Reload does not work when started with `go run`, or on Windows.
  - With systemd, set `ExecReload=/bin/kill -HUP $MAINPID` and `systemctl reload ebi-x` does the same.
- **Interrupted requests**: a request whose work was cut off by ebi-x going down becomes `interrupted` and is not retried automatically, because the work may have left side effects outside. Check the situation, then ask again with a new mention.
- **Disk**: per-thread clones are removed automatically after `EBIX_CHECKOUT_IDLE_TTL`, but per-thread working directories under `data/workspace/` are not.
- **Codex sign-in**: when the sign-in expires, a notice goes to the approval channel. Sign in again with `codex login`.
- **Keeping data**: `data/` is excluded by `.gitignore`. State, memory, personal settings and playbooks live there and are never committed.
