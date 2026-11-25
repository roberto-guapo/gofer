# Maintaining my Gofer fork

These notes belong to the `pi` branch of [roberto-guapo/gofer](https://github.com/roberto-guapo/gofer). They describe my maintained version, including changes that upstream may not accept. Closing a pull request does not delete the changes from my fork.

For installation and updates on the Raspberry Pi, start with [Installing my fork on a Raspberry Pi](./docs/raspberry-pi-fork.md). Downloading an upstream release does **not** include my unmerged changes.

## Branches and remotes

| Name | Purpose |
| --- | --- |
| `origin` | My fork: `https://github.com/roberto-guapo/gofer.git` |
| `upstream` | Original project: `https://github.com/cristianadrielbraun/gofer.git` |
| `upstream/master` | Local record of upstream's master branch, refreshed by `git fetch upstream`. |
| `master` | A clean copy of upstream's `master` with nothing of mine on it. Never commit to it; bring it up to date instead. |
| `pi` | `master` plus my own commits, listed below. Merge upstream improvements here and deploy this branch. |
| `backup/*` | Recovery points made before syncing or rebuilding `pi`. Delete them once the new `pi` has been running without problems. |

To bring `master`, and my fork's copy of it, up to date with upstream:

```sh
git fetch upstream
git fetch . upstream/master:master
git push origin master
```

The second command only fast-forwards. If it refuses, something was committed to `master`; find it with `git log upstream/master..master` and move it to its own branch before trying again.

On November 23, 2026, `pi` was rebuilt as upstream's `master` with my commits applied on top, replacing the earlier history of merges from `master` and feature branches. The older branches `feature/system-monitor`, `feature/calendar-week-start` and `fork-docs` are kept for reference only; `pi` no longer merges them.

## What `pi` adds to upstream

As of November 24, 2026:

| Change | Status upstream |
| --- | --- |
| Optional Linux [system monitor](./docs/system-monitor.md) in the sidebar | Fork only. Not submitted. |
| Stop the message frame growing without end around emails that size themselves to it | Fixed on `pi`; not yet reported or submitted upstream. A pull request needs a test that fails without the fix. |
| Show Outlook subfolders by their own name, such as "Conflicts" instead of "Sync Issues/Conflicts" | Fixed on `pi`, with a test. Not yet reported or submitted; the upstream-based branch is `fix/outlook-subfolder-names`. |
| These notes and [the Pi installation guide](./docs/raspberry-pi-fork.md) | Fork only. |

Kept off `pi` on purpose:

- **Stop rereading whole Outlook folders for messages without a sender**, on branch `fix/outlook-sender-repair-once`. Without it, an Outlook folder holding an item that has no sender, such as a draft in Deleted Items, is downloaded in full on every sync. The fix raises the database schema to version 154, and a schema number upstream hasn't taken could clash with a later upstream one. Wait for upstream to accept it, or renumber it, before deploying. Until then, emptying that folder in Outlook on the web stops the repeated downloads.

Earlier changes no longer need maintaining here:

- Mail card action buttons ([#124](https://github.com/cristianadrielbraun/gofer/pull/124)), compose tab order ([#125](https://github.com/cristianadrielbraun/gofer/pull/125)), opening the database on Windows ([#126](https://github.com/cristianadrielbraun/gofer/pull/126)) and the Raspberry Pi setup guides ([#128](https://github.com/cristianadrielbraun/gofer/pull/128)) were merged upstream.
- Keeping messages unread in unread-only views ([#127](https://github.com/cristianadrielbraun/gofer/pull/127)) was declined. Upstream did it differently instead: reading a message marks it read but leaves it in the Unread list until the list is refreshed.
- My first-day-of-week setting was replaced by upstream's, which offers Saturday, Sunday or Monday and also covers date pickers. Upstream saves the setting differently, so a week start chosen with my version is ignored until it is chosen again in Settings › Regional.

## Bring in the author's improvements

Run these Git commands from the repository on the development computer. They work in PowerShell or a Linux shell. Start with a clean working tree: commit work that should be kept, or stash it and remember to restore it later.

```sh
git switch pi
git status
git fetch upstream
git log --oneline HEAD..upstream/master
```

Create a recovery branch before merging. Replace `YYYYMMDD` with today's date; use a new name if it already exists.

```sh
git branch backup/pi-before-sync-YYYYMMDD
git merge upstream/master
```

Review and test the result before publishing or installing it. Run the Go tests on Linux, such as WSL, the Pi or a Docker build: on Windows, many of them fail for reasons unrelated to Gofer, such as Unix socket paths, symlinks and timer precision.

```sh
git diff --check
go test ./internal/systemmonitor ./internal/views ./internal/i18n
go test ./internal/handler -run SystemMonitor
node scripts/test-system-monitor.cjs
```

Run other tests relevant to upstream's changes, or `go test ./...` for a large update, and build the release as described in the [Pi instructions](./docs/raspberry-pi-fork.md#build-my-version). A test that fails here should be checked against `upstream/master` in the same environment before blaming the merge. In the app, check the system monitor and open a few HTML newsletters before relying on the new build.

```sh
git push -u origin pi
```

Use ordinary merges and pushes for this long-lived branch. Routine upstream updates do not require rebasing or force-pushing. If Git rejects a push because `origin/pi` moved, fetch and inspect those commits before merging them; do not overwrite them blindly.

### If there is a conflict

1. Run `git status` to see the affected files.
2. For generated `*_templ.go` files, don't merge by hand. Take either version, then run `templ generate --ignore-pattern='(^|/)dist(/|$)'` from the repository root with the templ version named in `go.mod`, so they are rebuilt from the merged `.templ` files.
3. Edit each other file to keep the intended behavior from both versions. Remove the `<<<<<<<`, `=======`, and `>>>>>>>` markers.
4. Stage each resolved file with `git add path/to/file`.
5. Finish with `git commit`, then test before pushing.

To cancel a merge while it is still in progress:

```sh
git merge --abort
```

If upstream accepts or independently implements one of my changes, compare the behavior and resolve overlap during the merge. Do not automatically delete the feature or cherry-pick it again. A squash merge upstream may use different commit IDs for the same work.

### Rebuild `pi` from upstream

When `pi`'s history gets tangled, for example after reverting a change upstream replaced, it can be rebuilt instead of merged: start from `upstream/master` and reapply only my own commits.

```sh
git fetch upstream
git log --oneline --no-merges upstream/master..pi   # the commits to keep
git switch -c pi-rebuild upstream/master
git cherry-pick COMMIT_SHA ...                     # oldest first; regenerate templ files on conflict
git diff --stat pi pi-rebuild                      # should list only intended differences
git branch backup/pi-before-rebuild-YYYYMMDD pi
git branch -f pi pi-rebuild
git switch pi
git branch -D pi-rebuild
```

This rewrites `pi`'s history. Once `pi` is on GitHub, publishing a rebuilt `pi` needs `git push --force-with-lease origin pi`, and the Pi's checkout then has to be reset to it instead of pulled. Prefer ordinary merges once `pi` is pushed.

## Make another focused PR

Start from upstream rather than `pi`, so the PR does not include every fork customization. Replace the example branch name and `COMMIT_SHA` before running:

```sh
git fetch upstream
git switch -c fix/descriptive-name upstream/master
git cherry-pick COMMIT_SHA
# Review the diff and run the relevant checks.
git diff --stat upstream/master...HEAD
git log --format=full upstream/master..HEAD
git push -u origin fix/descriptive-name
```

Open the PR from this new branch to `cristianadrielbraun/gofer:master`. Keep its title and summary limited to that change, and follow upstream's [CONTRIBUTING.md](./CONTRIBUTING.md): every fix needs a test that fails without it, and commits and summaries carry no Claude/AI author trailers or generated-by footer. Inspect the commit messages before submitting. Return to `pi` with `git switch pi` afterward.

## Local checkout reminders

`task clean` and `task release:all` delete `dist/`. In this checkout it only holds build outputs, so that is safe, but move anything there that should be kept first.

Keep these notes updated when the branch arrangement or deployment changes. Never put actual passwords, OAuth secrets, or setup tokens in Git.
