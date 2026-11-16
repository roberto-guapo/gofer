# Maintaining my Gofer fork

These notes belong to the `pi` branch of [roberto-guapo/gofer](https://github.com/roberto-guapo/gofer). They describe my maintained version, including changes that upstream may not accept. Closing a pull request does not delete the changes from my fork.

For installation and updates on the Raspberry Pi, start with [Installing my fork on a Raspberry Pi](./docs/raspberry-pi-fork.md). Downloading an upstream release does **not** include my unmerged changes.

## Branches and remotes

| Name | Purpose |
| --- | --- |
| `origin` | My fork: `https://github.com/roberto-guapo/gofer.git` |
| `upstream` | Original project: `https://github.com/cristianadrielbraun/gofer.git` |
| `pi` | My combined version; merge upstream improvements here and deploy this branch. |
| `upstream/master` | Local record of upstream's master branch, refreshed by `git fetch upstream`. |
| Individual PR branches | Focused changes submitted upstream; keep unrelated work out of them. |

My local `master` and `origin/master` contained the original Raspberry Pi documentation commit when these notes were written. Do not assume either is a clean upstream base. Use a freshly fetched `upstream/master` for a new PR.

## Changes being maintained

Snapshot: November 15, 2026. PR status can change; use the links to check it.

| Change | PR branch | Upstream PR |
| --- | --- | --- |
| Mail card action buttons | `card-action-buttons` | [#124](https://github.com/cristianadrielbraun/gofer/pull/124) |
| Compose keyboard tab order | `compose-tab-order` | [#125](https://github.com/cristianadrielbraun/gofer/pull/125) |
| Windows SQLite paths | `windows-sqlite-path` | [#126](https://github.com/cristianadrielbraun/gofer/pull/126) |
| Preserve unread state in unread-only views | `fix/unread-only-selection` | [#127](https://github.com/cristianadrielbraun/gofer/pull/127) |
| Raspberry Pi and shared setup guides | `raspberry-pi-setup` | [#128](https://github.com/cristianadrielbraun/gofer/pull/128) |

`pi` contains the original combined application work. The PR branches were prepared separately, and #124–#126 had their histories rewritten to remove Claude attribution. Their commit IDs differ from commits already on `pi`; do not merge every PR branch just to make the IDs match.

The Windows change on `pi` is an earlier startup workaround. PR #126 also uses a shared URI helper for migration copies and adds a test. They are not identical implementations. Reconcile them deliberately if bringing the PR version into `pi`.

The reorganized guides from #128 have been merged into `pi`. These fork-only notes are not part of that PR.

## Bring in the author's improvements

The optional [system monitor](./docs/system-monitor.md) is developed on `feature/system-monitor`, branched from `pi`. It is not included in `pi` until that feature branch is merged. Build and deploy the branch containing the feature to use it; an upstream release does not include it.

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

Review and test the result before publishing or installing it. Useful checks for the existing custom changes include:

```sh
git diff --check
node scripts/test-compose-tab-order.cjs
node scripts/test-compose-one-time-from.cjs
node scripts/test-mail-auto-read.cjs
go test ./internal/views ./internal/i18n
```

Run other tests relevant to upstream's changes, and build the release as described in the [Pi instructions](./docs/raspberry-pi-fork.md#build-my-version). These checks are a starting point, not a complete regression suite. In the app, check compose tab order, configured card actions, and unread-only selection before relying on the new build.

```sh
git push -u origin pi
```

Use ordinary merges and pushes for this long-lived branch. Routine upstream updates do not require rebasing or force-pushing. If Git rejects a push because `origin/pi` moved, fetch and inspect those commits before merging them; do not overwrite them blindly.

### If there is a conflict

1. Run `git status` to see the affected files.
2. Edit each file to keep the intended behavior from both versions. Remove the `<<<<<<<`, `=======`, and `>>>>>>>` markers.
3. Stage each resolved file with `git add path/to/file`.
4. Finish with `git commit`, then test before pushing.

To cancel a merge while it is still in progress:

```sh
git merge --abort
```

If upstream accepts or independently implements one of my changes, compare the behavior and resolve overlap during the merge. Do not automatically delete the feature or cherry-pick it again. A squash merge upstream may use different commit IDs for the same work.

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

Open the PR from this new branch to `cristianadrielbraun/gofer:master`. Keep its title and summary limited to that change. Upstream has requested no Claude/AI author trailers or generated-by footer in commits or summaries; inspect the commit messages before submitting. Return to `pi` with `git switch pi` afterward.

## Local checkout reminders

Some PR branches are checked out in worktrees under `dist/`. `git worktree list` shows them. If Git says a branch is already checked out, use that worktree rather than forcing a second checkout.

In this Windows checkout, avoid `task clean` and `task release:all`: their scripts remove `dist/`, which currently contains those worktrees. `task release` builds the single binary without deleting that directory. Build on the Pi or in a dedicated Linux checkout using the installation notes.

Keep these notes updated when the branch arrangement or deployment changes. Never put actual passwords, OAuth secrets, or setup tokens in Git.
