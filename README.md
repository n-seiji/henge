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
- **Deliverables**: it attaches PDFs, images, pptx and CSV files to the thread. Playbooks can cover slide decks from draft to delivery.
- **Safe execution**: only approved people and channels can use it, and Codex runs in a per-thread working area inside a sandbox. Before actions that need confirmation, such as sending something outside, it asks the requester.

## Getting started

You need Go (`mise install` sets it up), Codex CLI 0.149.0 or later (signed in), and a Slack workspace where you can create apps.

1. Create a Slack app from `examples/slack-app-manifest.json` and install it in your workspace. Get the Bot Token (`xoxb-`) and the App Token for Socket Mode (`xapp-`).
2. Create `.env` and write the tokens and the IDs of the users and channels to allow.

   ```sh
   cp .env.example .env
   ```

3. Build and start it.

   ```sh
   mise run build   # or go build -o ebi-x ./cmd/ebi-x
   TZ=Asia/Tokyo ./ebi-x
   ```

4. Invite ebi-x to an allowed channel and mention it with `@ebi-x alive?` to check that it answers.

## Documentation

| File | Contents |
| --- | --- |
| [docs/setup.md](docs/setup.md) | Installation and settings (Slack app, `.env`, permissions, operations) |
| [docs/usage.md](docs/usage.md) | How to use it in Slack, and how each feature behaves |
| [docs/design.md](docs/design.md) | Design decisions and their reasons |
| [examples/](examples/) | The Slack app manifest, and example action rules and playbooks |

## Development

```sh
mise run test   # go test -race ./...
mise run lint   # go vet ./... and a gofmt check
```

To apply code changes to a running ebi-x, use `mise run reload`. It swaps the process without stopping work in progress ([docs/setup.md](docs/setup.md#9-operations)).
