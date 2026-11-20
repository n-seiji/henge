# Usage

How to ask ebi-x for things in Slack. For installation and settings, see [setup.md](setup.md).

The examples below are in English for readability. The bot's own fixed messages are in Japanese, and some commands are Japanese words; those are shown with their meaning.

## How to ask

| Method | How |
| --- | --- |
| Mention | Write `@ebi-x do X` in an allowed channel. |
| DM | Write in ebi-x's Messages tab without a mention. Outside a thread it is a new request; a reply in a thread continues it. Group DMs are ignored. |
| Shortcut | Choose "…" → "Ask ebi-x" on a message and write the request in the form. The work happens in that message's thread. Not available in DMs. |

- One Slack thread is one Codex session. Mention it again in the same thread and it continues with what happened so far.
- Add the subscription reaction the operator chose to a thread's parent message, and replies from allowed people in that thread are handled without a mention. It works in public and private channels and lasts 14 days by default.
- The content of private channels and Slack Connect channels is not visible from other channels. Slack lookups there stay inside that channel.
- Per-channel instructions and limits can be changed by asking ebi-x in the approval channel.
- Do not paste tokens or private keys into Slack with a request. If the request or thread holds a string shaped like a GitHub, Slack, cloud or other token, ebi-x stops before sending it to Codex and asks for it to be deleted and revoked.
- A mention or DM that only checks it is alive, such as `alive`, `ping` or `生きてる？` ("are you alive?"), is answered without work, saying whether Codex is reachable. It answers even where it is not allowed.

## While it works

### Progress

While working, the thread's status shows what Codex is doing now.

```text
ebi-x is working on "find the cause"… 1/4 done
ebi-x is running a command (go test)… 1/4 done
ebi-x is editing files (bot.go and 2 more)… 2/4 done
```

Command arguments are not shown, because they may contain tokens.

There is a limit on how many threads run at once; while it is reached, requests show as waiting their turn. Requests in the same thread are handled one at a time.

### Stopping

Mention `@ebi-x 停止` (stop) in the same thread to stop running or waiting work. In subscribed threads, "停止" without a mention also works.

- Only a whole message such as `停止` or `stop` counts as stopping. "How do I stop it?" is a normal request.
- Changes and commits made so far stay; nothing is rolled back.
- It also stops the thread's follow-ups, PR watches, schedules, reminders and reaction watches.

### Waiting for an answer

When it cannot continue without the requester's answer (confirmation before sending something outside or deleting, a request that splits into very different readings, and so on), ebi-x adds 🙋 instead of ✅ and asks.

```text
@alice  @ebi-x clean up old branches
  └ @ebi-x  There are 12 merged branches. …
            🙋 @alice is being waited on. Mention ebi-x in this thread to answer.
            - May I delete the remote branches too?
```

Only the requester's own reply counts as approval. If someone else tries to approve, it asks the requester again. An action explicitly asked for in the request ("go as far as opening a PR", and so on) counts as approved within that scope. Work that costs money is confirmed first, with the basis and a rough amount. The operator sets the amount below which no confirmation is needed in the action rules.

## Results

- The mention gets ✅ when done, and ❌ on failure or interruption.
- PDFs, images, pptx and CSV files Codex made are attached to the thread. If attaching fails, the files stay in the working area, so you can ask it to resend the attachments.

## Continuing later

### Follow-ups

When waiting for CI or a deploy, checking again after a while, or doing something at a given time, ebi-x plans its own "check later" and continues in the same session at that time.

```text
@alice  @ebi-x deploy it and tell me when it's done
  └ @ebi-x  I started the deploy. …
            ⏰ Follow-up around 10/9 16:30 JST: check that the deploy finished and report
  └ @ebi-x  ⏰ Starting the scheduled follow-up: check that the deploy finished and report
  └ @ebi-x  The deploy finished. …
```

- A follow-up is due between 5 minutes and 7 days ahead, one per thread. It can chain up to 10 times without a human message.

### Schedules

Ask something like "check CI failures at 9 on weekday mornings", and it works on the given weekdays and time.

```text
@alice  @ebi-x every weekday at 9am, check main's CI failures and report
  └ @ebi-x  🔁 Schedule set: weekdays 09:00, "check main's CI failures and report" (next 10/12 09:00 JST)
(weekdays at 9:00, in the same channel)
@ebi-x  🔁 Schedule (weekdays 09:00): check main's CI failures and report
  └ @ebi-x  No failures since yesterday. …
```

- Days are daily, weekdays, weekends, or a list of weekdays, with a time. Nothing more frequent than once a day.
- Each run works in a new thread without carrying over the previous one.
- To stop it, mention "停止" (stop) in the thread where it was set.

### Reminders

Ask something like "remind me to send the estimate at 10 tomorrow", and at that time it mentions the requester once in the same thread. It does no work.

```text
@alice  @ebi-x remind me to send the estimate at 10 tomorrow
  └ @ebi-x  🔔 I'll remind you in this thread at 11/19 10:00 JST: send the estimate
(11/19 10:00, in the same thread)
  └ @ebi-x  @alice 🔔 Reminder: send the estimate
```

- It can be set from 1 minute to 366 days ahead.
- For work at a given time ask for a follow-up, and for repetition a schedule.
- To cancel, ask it to cancel the reminder in the thread where it was set, or mention "停止" (stop).

### Watching PRs

When ebi-x opens a PR in its work, it watches the PR. On a CI failure, review or comment, it handles it in the same session and reports.

```text
@alice  @ebi-x fix this bug and open a PR
  └ @ebi-x  I fixed it and opened a PR. …
            👀 Watching the PR below. I'll handle CI failures, reviews and comments in this thread and report.
  └ @ebi-x  🔔 The PR has activity (…/pull/42): CI failed (failed check: test). Starting on it.
  └ @ebi-x  The test failure was caused by time zone handling. I fixed it and pushed. …
  └ @ebi-x  🎉 The PR was merged: …/pull/42
```

- Work resumes on CI failures of the head commit, and on reviews and comments from people with write access to the repository (OWNER, MEMBER, COLLABORATOR). In a public repository where anyone can comment, comments alone do not start it.
- When CI passes, or the PR is merged or closed, it only reports. Watching ends once the PR is merged or closed.
- Text on GitHub is treated as data, not instructions.

## Reaction watches

After handing out a task with @mentions, ebi-x can chase whether everyone has finished. End the message with `@ebi-x 監視して` ("watch this"), and on weekdays at 19:00 it reminds, in the thread, the people @mentioned in that message who have not yet added a `:done:` reaction. It runs on the bot alone, without Codex.

```text
@alice  @bob @carol please submit the year-end forms. Add :done: when you have @ebi-x 監視して
  └ @ebi-x  👀 Starting the :done: watch. It covers the 2 people @mentioned in the original message. On weekdays at 19:00, …
(the next weekday at 19:00)
  └ @ebi-x  @carol
            It looks like you have not added :done: yet. Once you're done, add :done: to the original message. (reminder 1)
(once everyone has added :done:)
  └ @ebi-x  ✅ Everyone added :done:, so the watch has ended.
```

- It covers the people @mentioned in that message (or in the thread's first message if it has none). `@channel`, `@here` and user groups expand to the members in the channel. It does not start for more than 100 people.
- By default the reaction is `:done:` and reminders go out on weekdays at 19:00 (the time on the machine running ebi-x). The operator can change both ([setup.md](setup.md#global-settings)). Holidays are not skipped.
- The interval can be changed in weekdays, such as `1日おきに監視して` (every other day) or `3日ごとに監視して` (every 3 days). The default is daily, up to 7 days.
- It ends after 5 reminders, or when the original message is deleted.
- To stop it, mention `@ebi-x 監視をとめて` ("stop watching") in the thread.

## Home tab

Open ebi-x's Home tab to see the current state of what you asked for:

- requests working or waiting their turn, with links to their threads
- scheduled follow-ups
- watched PRs and their CI state
- schedules and their next times
- reaction watches (how many people, the interval, the next time)
- your personal settings

## Personal settings

Ask it to keep a way of replying from now on, such as "answer in English from now on" or "keep it shorter", and it saves that as your setting.

- The content is not shown in the thread; it is told to you in a message only you can see.
- Even with several people in a thread, only the current speaker's settings are used.

## Making and fixing playbooks from Slack

Where `playbook-from-thread` is installed, people allowed to use ebi-x can ask in the approval channel, for example `@ebi-x make this into a playbook` or `@ebi-x fix this playbook so it asks one question at a time`. ebi-x asks about missing points one at a time, shows the finished draft, asks "Is this content OK?", and saves it after approval.

When work turns up a fact that contradicts a playbook or is not written in it yet, ebi-x adds a line to the playbook's notes and says so ("📝 Updated the notes of playbook "release-notes": …"). Once three notes pile up, it folds them into the body and says so ("📘 … folded into the body"). When the body changes, it also leaves a record in the approval channel.

## Codex usage limits

With an approval channel, ebi-x checks Codex's usage limits (5 hours and 1 week) every 30 minutes and, in the approval channel:

- posts every hour what is left in each window and the trend over the last 12 hours (hours without change are skipped)
- notifies when less than 10% is left, and when it is used up
- posts a summary of the previous day after midnight
- answers with what is left now when mentioned with "usage", "残り" (remaining) and the like

On weekends it posts and notifies nothing; if usage is still low at Monday's first check, it notifies then.

The limits include use of the same account outside ebi-x. When signed in with an API key there are no usage limits, so it does nothing.

When work stops on a usage limit or an expired sign-in, it says so in the request's thread along with when to ask again, and notifies the approval channel too (at most once an hour per kind).

When three requests fail for other reasons within an hour, it also notifies the approval channel, suspecting a settings or environment problem.
