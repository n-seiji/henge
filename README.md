# ebi-x

ebi-x is a resident bot that hands work requested in Slack to Codex and posts the result back to the same thread.

```text
@alice  @ebi-x find out why CI is failing
  └ ebi-x is running a command (go test)… 1/3 done
  └ @ebi-x  The cause was time zone handling. I fixed it and opened a PR. …
            👀 Watching the PR below. I'll handle CI failures and reviews in this thread.
```

## What it does

- **Requests**: ask with a mention, a DM to ebi-x, or the "…" menu on a message. Reply in the thread to continue in the same Codex session.
- **Visible progress**: while working, it shows what Codex is doing now, and "stop" stops it. The Home tab lists the state of your requests.
- **Work after the conversation ends**: follow-ups that check again later, schedules on given weekdays and times, one-time reminders at a given time, and handling CI and reviews on the PRs it opened.
- **Deliverables**: it attaches PDFs, images, pptx and CSV files to the thread.
- **Safe execution**: only approved people and channels can use it, and Codex runs in a per-thread working area inside a sandbox. Before actions that need confirmation, such as sending something outside, it asks the requester.

## Getting started

You need Codex CLI 0.149.0 or later (after `codex login`), a Slack workspace where you can create apps, and a macOS or Linux machine that can stay running.

1. Install it.

   ```sh
   curl -fsSL https://raw.githubusercontent.com/n-seiji/henge/main/install.sh | sh
   ```

2. Make a directory for the agent and run `ebi-x init` there. `.env` and `data/` are read from this directory.

   ```sh
   mkdir ~/ebi-x && cd ~/ebi-x && ebi-x init
   ```

   Paste the `slack-app-manifest.json` it writes into "From a manifest" on Slack's app creation screen and install the app in your workspace. Then answer the questions with the Bot Token (`xoxb-`), the App Token for Socket Mode (`xapp-`), your member ID, and the ID of the channel to talk to the bot in, and `.env` is created. That channel becomes the approval channel, where other people and channels are allowed from Slack.

3. Sign in to Codex and start it.

   ```sh
   codex login
   TZ=Asia/Tokyo ebi-x
   ```

4. Invite ebi-x to the channel you answered and mention it with `@ebi-x alive?` to check that it answers.

How to keep it running, and how to build from source, are in [docs/setup.md](docs/setup.md#0-installation).

## Documentation

| File | Contents |
| --- | --- |
| [docs/setup.md](docs/setup.md) | Installation and settings (Slack app, `.env`, permissions, operations) |
| [docs/usage.md](docs/usage.md) | How to use it in Slack, and how each feature behaves |
| [docs/design.md](docs/design.md) | Design decisions and their reasons |
| [examples/](examples/) | The Slack app manifest, service templates, and example action rules and playbooks |

## Development

```sh
mise run test   # go test -race ./...
mise run lint   # go vet ./... and a gofmt check
```

### Keeping what must not be published out

This repository is public. Secrets such as tokens, and words that must not appear such as company or channel names, are stopped before commit and in CI.

```sh
mise run hooks      # check staged changes and the commit message on every commit
mise run scan       # check every commit's changes and messages
mise run vulncheck  # check dependencies and code for known vulnerabilities
```

Write the words that must not appear in `.denylist` at the repository root, one per line (it is not in Git; lines starting with `#` are ignored, and case does not matter). In CI, put the same content in the Actions secret `DENYLIST`, and every commit and PR title and body is checked.

To apply code changes to a running ebi-x, use `mise run reload`. It swaps the process without stopping work in progress ([docs/setup.md](docs/setup.md#9-operations)).
