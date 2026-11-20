---
name: playbook-from-thread
description: Turn a thread's conversation into a playbook — for requests such as "make this into a playbook" or "playbook this", draft a reusable playbook from the thread and the request, and save or update it only after the content is reviewed and approved. "Just a draft" stays a draft
---

# Turn a thread's conversation into a playbook

Distill what was done in the thread (investigation steps, criteria, settled specifications, corrected mistakes) into a playbook draft that can be reused from next time.

## 1. Purpose

- Stop redoing the same question or investigation from scratch every time
- Turn corrections and remarks people made in the thread into lasting rules
- Separate what only holds for this thread from steps that generalize

## 2. What to extract (reread the thread and organize)

1. **Trigger**: which request wording or keywords should start this procedure (this becomes the condition in the description)
2. **Steps**: the reproducible steps of the investigation or work actually done. Drop trial and error and failed paths; keep only the shortest path
3. **Criteria**: which columns and conditions decided things. Always include places where people corrected you, in the form "Wrong: … / Right: …"
4. **Settled specifications and facts**: undocumented specifications settled in the thread. Content that belongs in existing documentation is split out as a proposed addition to that document
5. **Output format**: the structure of the answer the requester was happy with

## 3. What to leave out

- Specific IDs, company names, personal names and concrete investigation results (replace with placeholders such as `<user_id>` when an example is needed)
- Mentions of one-off events (things that do not generalize)
- Conversational noise such as inner monologue, error logs and apologies

## 4. Draft format

Use the same format as existing playbooks, complete enough to work as soon as it is placed in `playbooks/`:

```markdown
---
name: <short kebab-case name>
description: <a specific description for being picked from the list, including trigger keywords and the kinds of requests>
---

# <Title>

<Purpose in one or two lines>

## Steps
...

## Criteria
...

## Output format
...
```

- The description is the only routing clue, so state concretely which requests it is for
- Do not repeat content of existing playbooks; refer to the relevant playbook instead
- If a **fix** to an existing playbook is enough, do not create a new one; propose a diff saying what to add where in which file
- When fixing an existing playbook, fold in the "Latest notes" shown under it in the list. Notes are facts learned from work since the body was last updated, and folded notes stop being shown once the body is saved. If you leave a note out, give the reason with the draft

## 5. Conversation, saving and applying

- Do not ask again for information in the form, request or thread. If something is missing, ask one question at a time and wait for the answer before moving on.
- For both new playbooks and fixes, always present the complete draft in Slack, ask "Is this content OK? If so, I'll save and apply it", and end the turn. Do not treat the first request alone ("make it", "turn it into a playbook", "fix it") as approval of the finished draft.
- Present the target name, whether it is new or an update, the purpose, trigger, steps, criteria, completion conditions and output, plus the full body to be saved. For updates, add the changes too. If it is long, split it rather than abbreviate. Keep drafts awaiting approval in `playbook-drafts/` in the workspace, and do not write to the live directory yet.
- Save and apply only after the requester explicitly approves the latest version presented ("OK", "save it like this", and so on). No answer, time passing and change requests are not approval. When changes are asked for, fix the draft and present it again for confirmation.
- Approval applies only to the target and body presented. Do not add your own changes when saving. If "just a proposal" or "just a draft" was specified, save to `playbook-drafts/` in the workspace and do not change live playbooks.
- The live location is the directory this playbook itself is in. Find it from this playbook's absolute path in the list, and pass that path and the changes to the work turn. Do not save to another directory you guessed, or to memory.
- Before creating a new one, check the name and description of existing playbooks. If it duplicates one, update the existing one. Ask in one question only when the target to create or update is unclear.
- Make a complete version that fits the current requirements, stating the trigger, inputs, steps, criteria, completion conditions and output. Do not automate sending, filing, deleting and the like on your own when the user did not ask for it.
- The file name is lowercase letters, digits and hyphens only, plus `.md`. If the file exists, read all of it and update only what was asked.
- Reread the target file right before saving; if it changed since the draft was made, stop, present a draft that takes the change in, and wait for approval again. Move the existing version to `.history/` in the same directory with a timestamp. When creating a new one, do not overwrite a file with the same name.
- Check that it is UTF-8, within 64KB, and starts with frontmatter with a non-empty name and description. Write a temporary file with a `.tmp` extension in the same directory and replace the target with rename once done.
- Reread after saving and confirm the request is reflected. If it failed, do not report it as saved; say where the draft is and what failed.
- In Slack, reply briefly with the saved name, what the playbook does, an example of calling it, and where it is saved. There is no need to repeat the full text already presented before approval.
- The list is loaded at the first request of a new thread, so a saved playbook is available from the next thread that starts. Do not restart the bot.

## 6. Do not

- Include personal information or credentials that appeared in the thread in the draft
- Write guesses that the thread did not settle as steps or criteria
- Start making a playbook unasked (act only on an explicit request)
