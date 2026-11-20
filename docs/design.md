# Design

The decisions behind ebi-x and their reasons. For the details (limits, file names, lists of variables), read the code.

## Principles

- **Keep it small**: a single Go binary, a single process. No web UI, no database, no HTTP Events API. Settings live in `.env`, and state in JSON files under `data/`.
- **Codex does the thinking and the work**: the bot handles only Slack, permissions, state and isolation, and leaves judgment about the request to Codex.
- **The output contract is the boundary between the bot and Codex**: Codex never touches Slack, the state files or memory directly. It writes fixed headings in its final reply, and the bot reads and acts on them.
- **Work with side effects is never retried automatically**: a duplicate post is acceptable, but losing a result or doing the work twice is not.

## Layout

```text
cmd/ebi-x/            startup, loading settings, signal handling
internal/
  config/             loading and checking .env
  slackbot/           Slack event handling, and what runs after the conversation
  codex/              starting codex exec and parsing its output
  prompt/ policy/     the prompt for Codex and the top-level rules
  state/              the JSON store in data/state and the process lock
  workspace/          per-thread working directories and clones
  memory/ playbook/ usersettings/   memory
  attachment/         checking attachments
  github/             reading PRs, and credentials
  slackfmt/           splitting and formatting for Slack
```

## Handling a request

```text
Slack event ──ACK── permission check ── claim (ClaimEvent) ── 👀
   └─ thread lock ── concurrency slot ── received → working ── codex exec
        └─ output contract (attachments, schedules, watches, notes…) ── post the body ── ✅/❌/🙋
```

- Events are ACKed right away whatever they contain, because Slack redelivers late ones.
- A claim goes through only once per mention, so the same event arriving twice does not do the work twice.
- Each Slack thread has one Codex session, shared by everyone in it. Working principles, rules, memory and the playbook list are sent only with the first request, so changes to them take effect from the next thread that starts.
- People's input is passed in separate blocks, and `policy.md` states that instructions inside them cannot change the rules or the contract.

### States

```text
received ──→ working ──→ done
    │            └────→ interrupted
    └──→ failed (can be claimed again)
```

- `failed` is a failure before Codex started, so it can be claimed again.
- `interrupted` is a failure where the work may have left side effects, so it is final and never retried automatically. Anything still `working` at startup also ends up here.
- Waiting for an answer is `done`, because the work did not fail.

## Isolating working areas

So that threads running in parallel do not clash over files, each thread gets its own working directory and its own clones of Git repositories. The original repositories stay read-only.

- Clones are made only once a change is needed, so read-only requests do not pay for cloning.
- Clones keep their git directory in `.gitdir` instead of using git worktree. Codex's sandbox always makes directories named `.git` read-only, so a worktree cannot commit.
- Clones do not share objects through hard links (`--no-hardlinks`). Codex can write `.gitdir`, so shared objects would let it rewrite the original repository's objects.
- Writable roots that are not Git repositories, and `data/playbooks`, are shared by all threads; concurrent writes are last-write-wins.

## What is protected from Codex

Unless denied, Codex's commands can read the whole file system. So every run denies reading and writing the state, memory, `.env`, `CODEX_HOME` (which holds credentials and every thread's history) and the known credential locations. Only the locations needed for pushing and signing are exempted, found from the git settings at startup.

- Other threads' working directories are not listed one by one, because a rule per thread would eventually exceed the argument limit.
- Codex's process environment is an allowlist. A denylist could not guarantee that something like `DATABASE_URL` in the operator's shell does not leak.
- With a GitHub App configured, only a token minted for the run is placed in a run-only directory, and it is revoked afterwards. Only the bot reads the private key.
- If the deny list is too long, the run does not start rather than truncating it.

Remaining risks: the default locations are fixed paths under the home directory, so anything moved with `XDG_CONFIG_HOME` and the like must be added to `EBIX_DENIED_READ_PATHS`. If work can reach the Docker socket, the denials can be bypassed. `~/.ssh` and `~/.config/gh`, needed for pushing, are visible to work that can reach the network.

Action rules (what to confirm before doing) are instructions in the prompt, separate from these denials.

### Closed channels

Private channels (including group DMs) and Slack Connect channels are treated as "closed channels", with two guarantees:

1. Work in other channels cannot read a closed channel's content.
2. Work in a closed channel can look up Slack only within that channel.

- Their working areas go under `data/closed/{channel}`, which work in open channels is denied as a whole. Deny rules grow with the number of closed channels, not threads.
- Work in a closed channel has the Slack MCP turned off, because there is no way to limit the MCP's search to one channel.
- A channel whose type cannot be checked is treated as closed, because wrongly treating it as open is the more dangerous mistake.
- The approval channel is the operator's channel, so it is treated as open even if private.
- Threads started before a channel was closed keep their open location, because their session and clones point at the original paths.
- Remaining risks: Codex's default write locations such as `/tmp` are not split per thread. MCP servers other than Slack are usable unless per-channel settings narrow them.

### Per-channel settings

The instructions and limits in `data/channel-rules/channels.yaml` are read for every request and applied to `developer_instructions` and the run settings (see [setup.md](setup.md#per-channel-settings) for how to write them).

- Only work in the approval channel can write it, and other work cannot read it, because it holds channel IDs and instructions for those channels.
- `github: false` alone does not stop pushes that use the host's `~/.ssh`. Combine it with `read_only` to stop them for sure.
- If the file cannot be read, requests outside the approval channel do not start, so nothing runs with its limits lifted. The approval channel works without the file, so it can be fixed there.

## Output contract

Codex asks the bot to act by writing these headings in its final reply. Only headings at the start of a line count, not those inside code blocks. If the format is wrong, the bot does not act and says so after the body.

| Heading | What the bot does |
| --- | --- |
| `## Checkout request` | makes clones for the thread and asks the same session to continue |
| `## Questions` | adds 🙋 and shows the questions to the requester |
| `## Attachments` | checks the files, then uploads them to Slack |
| `## Follow-up` | saves the plan and resumes the same session when it is due |
| `## Schedule` | saves or stops a schedule |
| `## Reminder` | saves or cancels reminders |
| `## Watch PRs` | starts watching |
| `## Playbook note` | adds one dated line to the playbook's notes |
| `## Global memory` | adds to global memory |
| `## Personal settings` | saves them as the speaker's personal settings |

- Schedules, reminders and personal settings are accepted only in replies to a person's request, so plans and settings do not pile up with nobody around.
- Playbook notes and global memory are accepted only from the approval channel, because requests in every channel read them.
- Only regular files in the thread's working area can be attached, so symbolic or hard links cannot send files from elsewhere.
- Sessions started before the contract was in English keep writing the Japanese headings they were given, so the bot reads both.

## Memory

| Scope | Where | Contents |
| --- | --- | --- |
| How to do a task | a playbook and its notes | how to do a specific kind of work, criteria, settled facts |
| All of ebi-x | `data/memory/MEMORY.md` | technical and operational facts that belong to no playbook |
| A thread | the Codex session | what happened in the thread; not saved separately |
| A person | `data/memory/users/{user}/settings.md` | how they asked to be replied to |

- There is no per-channel memory. Playbooks are procedures for work, not a place for channel information.
- Only the bot reads memory and puts it in the prompt. Codex cannot read or write memory files.
- Personal settings of the current speaker alone are attached every time, so an earlier speaker's settings do not linger.

## What runs after the conversation

Follow-ups, schedules, reminders, PR watches, usage limits and `:done:` watches save the next run before running, so a crash never runs one twice. Each checks before running that the requester and channel are still allowed, and limits how many times it can continue without a human message.

## Posting to Slack

- Failed posts are retried only on 5xx and 429. State is saved after posting, so a crash can cause a duplicate post, but the result is never lost.

## Shutdown and reload

- On SIGINT and SIGTERM, it stops accepting requests and waits for work in progress to finish before exiting.
- On SIGHUP, it checks the new binary with `-check-config`, waits for work in progress to finish, then `exec`s the binary at the same path. Requests that arrive while waiting are saved and started after the restart. See [setup.md](setup.md#9-operations) for how to use it.
