---
name: slides
description: Create, revise and deliver slides from Slack — for requests such as "make slides about X", "shorten page 3", "this version is OK, finalize it", "give me the pptx" or "resend the attachments", write the draft in this thread's clone of the slides repository, attach preview images and a PDF to the same thread, and deliver the approved version's pptx only after the user explicitly approves it
---

# Create, revise and deliver slides in Slack

Write a draft from the request, revise it while showing previews in the same thread, and deliver as an editable pptx only the version the user said OK to.

## Assumptions

- The slides repository is in `EBIX_WRITABLE_ROOTS`, and the work turn's prompt shows it as "original path → this thread's clone". Always work in that clone; do not touch the original path or other threads' clones.
- For how to write drafts, the design rules and where builds go, read and follow `README.md`, `docs/workflow.md` and `scripts/build.sh` in the clone. Where they differ from this playbook, the slides repository's rules win, except for the approval gate.
- Ask the bot to attach files to Slack with an "## Attachments" heading in the work turn's final reply. Do not send to Slack yourself.
- Do not push or merge to main, or publish to Google Slides. Push a working branch only when asked.

## Version record

Keep `slides-state.md` in the thread's working directory (cwd) and record the following. For later requests in the same thread, read this file first to pick up the target draft and the latest version.

- Target: the draft directory in the slides clone (for example, `<draft directory>/<slide name>`)
- Versions: version number (v1, v2, …), the hash of the commit holding that version's draft, when it was shown, a summary of the changes, and the paths of the previews attached
- Approval: the approved version number and commit hash, the Slack user who approved, and the gist of their message. "Not approved" if none
- Delivery: the path of the delivered pptx, and the version number and commit hash it came from

Pin each version with a commit to the clone's working branch. Commit the draft each time you build previews, and record that hash as the version. If a file is missing or its contents disagree, do not fill in by guessing; ask the user.

## 1. New slides ("make slides about X")

1. Do not ask again for what the request or thread already says. Only if the subject (company, purpose, rough page count, and so on) cannot be decided, ask one question and end the turn.
2. Create the draft directory and write the draft following the slides repository's steps. It is not approved yet, so do not set `status: approved`.
3. Commit the draft and record it as v1 in `slides-state.md`.
4. Build draft previews.

   ```sh
   pnpm build "<target directory>" images --draft
   pnpm build "<target directory>" pdf --draft
   ```

5. Copy the generated files to `previews/v1/` under cwd. If there are many page images (more than about five), make one overview image of all pages, and keep only a few representative page images. Always attach the full PDF.
6. In the final reply, give the version number (v1), a summary of the structure, points to check, and "Reply with changes, or reply 'v1 is OK' if this version is fine", and list the absolute paths of the overview image (or page images) and the full PDF under "## Attachments".

## 2. Revisions ("shorten page 3")

1. Read the target draft and latest version from `slides-state.md`, and revise that draft in the clone. Do not start a new draft.
2. Commit the revision and record it under a new version number (v2, v3, …). A revision of an approved version is also a new version, and the approval goes back to not approved.
3. Build previews in `previews/v<N>/` as for new slides, and attach them with a summary of the changes and the version number.

## 3. Approval and delivery ("this version is OK, finalize it")

- Only the user's explicit message about a version already shown ("v2 is OK", "finalize this version", and so on) counts as approval. The first request, no answer, time passing, a revision request, finished builds or attachments, and reactions from the bot or others are not approval.
- If the version "this version" means differs from the latest or cannot be told, ask which version in one question and end the turn.
- Check that the approved version's commit is the clone's `HEAD` and that the working tree has no uncommitted changes. If not, do not deliver; explain the situation and ask.
- Record the approval in `slides-state.md` first, then set `status: approved` on drafts that need it, commit, and run the normal build. Do not set `status: approved` before recording the approval. Do not use options that bypass the existing approval gate or modify its scripts.

  ```sh
  pnpm build "<target directory>"
  ```

- The pptx is converted from the PDF, so before delivery render the pptx to images or similar and check for layout breakage (overflowing or missing text, font substitution, shifted images). If it is broken, do not deliver; tell the user what is wrong and how it could be fixed.
- If all is well, copy the pptx to `deliverables/`, add the delivery to `slides-state.md`, and attach the pptx under "## Attachments". State the version number it came from in the body.
- If a revision is requested after approval, go back to "2. Revisions" and get approval again as a new version.

## 4. Resending attachments ("resend the attachments")

- If the bot reported that attaching failed, the deliverables are still in the working area. Check `slides-state.md` and the earlier paths, and list the same files under "## Attachments" without rebuilding them.
- Rebuild from the recorded version's commit only if the files are gone, and say in the body that you rebuilt them.

## Do not

- Build or attach the normal-build pptx before the user explicitly approves.
- Read or write other threads' clones, working directories or `slides-state.md`.
- Attach everything in the working directory at once. List only the previews, PDF and pptx relevant to the request.
- Put credentials or `.env` contents in drafts, attachments or Slack messages.
