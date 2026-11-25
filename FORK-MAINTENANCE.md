# Maintaining my Gofer fork

These notes describe the `pi` branch of [roberto-guapo/gofer](https://github.com/roberto-guapo/gofer): upstream's Gofer plus my own changes, including ones upstream may not accept. Closing a pull request does not delete a change from my fork.

For installation and updates on the Raspberry Pi, start with [Installing my fork on a Raspberry Pi](./docs/raspberry-pi-fork.md). Downloading an upstream release does **not** include my unmerged changes.

## How the fork is arranged

Each of my changes lives on its own branch, based on upstream's `master`. `pi` is built from them by a script: it starts from `upstream/master` and merges every branch listed in [`fork/branches.txt`](./fork/branches.txt), in order. `pi` is never edited by hand; a change to it is a change to one of its branches, followed by a rebuild.

This keeps every change ready to offer upstream, since its branch holds only that change. It also keeps each update to the same steps: fetch upstream and rebuild. When upstream takes a change, its line comes off the list and it's gone from `pi` at the next rebuild.

| Name | Purpose |
| --- | --- |
| `origin` | My fork: `https://github.com/roberto-guapo/gofer.git` |
| `upstream` | Original project: `https://github.com/cristianadrielbraun/gofer.git` |
| `upstream/master` | Local record of upstream's master branch, refreshed by `git fetch upstream`. |
| `master` | A clean copy of upstream's `master` with nothing of mine on it. Never commit to it; bring it up to date instead. |
| `fix/*`, `feature/*` | A change meant for upstream, one per branch, based on `upstream/master`. |
| `fork/*` | A change that stays in my fork, such as the system monitor. |
| `fork/tools` | These notes, the Pi guide, [`fork/rebuild-pi.sh`](./fork/rebuild-pi.sh) and the list of branches `pi` is built from. |
| `pi` | Built by `fork/rebuild-pi.sh`. Deployed to the Pi. Never commit to it directly. |
| `backup/*` | The previous `pi`, kept by each rebuild. Delete them once the new `pi` has been running without problems. |

The older branches `feature/system-monitor`, `feature/calendar-week-start` and `fork-docs` are kept for reference only. Nothing builds from them.

To bring `master`, and my fork's copy of it, up to date with upstream:

```sh
git fetch upstream
git fetch . upstream/master:master
git push origin master
```

The second command only fast-forwards. If it refuses, something was committed to `master`; find it with `git log upstream/master..master` and move it to its own branch before trying again.

## What `pi` adds to upstream

[`fork/branches.txt`](./fork/branches.txt) is the exact list. As of November 24, 2026:

| Branch | Change | Status upstream |
| --- | --- | --- |
| `fork/tools` | These notes, [the Pi installation guide](./docs/raspberry-pi-fork.md) and the rebuild script | Fork only. |
| `fork/system-monitor` | Optional Linux [system monitor](./docs/system-monitor.md) in the sidebar | Fork only. Not submitted. |
| `fix/outlook-subfolder-names` | Show Outlook subfolders by their own name, such as "Conflicts" instead of "Sync Issues/Conflicts" | Has a test. Not yet reported or submitted. |

Kept off `pi` on purpose:

- **`fix/outlook-sender-repair-once`: stop rereading whole Outlook folders for messages without a sender.** Without it, an Outlook folder holding an item that has no sender, such as a draft in Deleted Items, is downloaded in full on every sync. The fix raises the database schema to version 154. Schema versions are a single sequence, so a number upstream hasn't taken could clash with a later upstream one. Add it to the list once upstream accepts it, or after renumbering it to upstream's next free version. Until then, emptying that folder in Outlook on the web stops the repeated downloads.

Earlier changes no longer need maintaining here:

- Mail card action buttons ([#124](https://github.com/cristianadrielbraun/gofer/pull/124)), compose tab order ([#125](https://github.com/cristianadrielbraun/gofer/pull/125)), opening the database on Windows ([#126](https://github.com/cristianadrielbraun/gofer/pull/126)) and the Raspberry Pi setup guides ([#128](https://github.com/cristianadrielbraun/gofer/pull/128)) were merged upstream.
- Keeping messages unread in unread-only views ([#127](https://github.com/cristianadrielbraun/gofer/pull/127)) was declined. Upstream did it differently instead: reading a message marks it read but leaves it in the Unread list until the list is refreshed.
- My first-day-of-week setting was replaced by upstream's, which offers Saturday, Sunday or Monday and also covers date pickers. Upstream saves the setting differently, so a week start chosen with my version is ignored until it is chosen again in Settings › Regional.
- My fix for message frames growing without end around emails that size themselves to the frame was replaced by upstream's own fix, "Stop some HTML emails from growing the reader without end" (`652ec01c`). It uses the same detection and also holds the height when the reader is resized or the message reloads.

## Update `pi` with the author's improvements

Run these on the development computer, from Git Bash or a Linux shell. Commit or stash any work in progress first.

```sh
fork/rebuild-pi.sh
```

The script:

1. fetches upstream;
2. reads the branch list from `fork/tools` (not from the current checkout);
3. builds the new `pi` in a separate worktree next to this one, `../gofer-pi-rebuild`: `upstream/master`, then a merge of each listed branch;
4. regenerates the templ files with the version in `go.mod`, and checks that the code builds;
5. keeps the old `pi` as `backup/pi-before-rebuild-<time>`, moves `pi` to the new build, and moves this checkout with it if `pi` is checked out here.

If anything fails, `pi` is left as it was. Nothing is pushed.

To try a rebuild without touching `pi`, build another branch name with `fork/rebuild-pi.sh --target pi-next`.

Then test the result. Run the Go tests on Linux, such as WSL, the Pi or a Docker build: on Windows, many of them fail for reasons unrelated to Gofer, such as Unix socket paths, symlinks and timer precision.

```sh
git diff --check upstream/master pi
go test ./internal/systemmonitor ./internal/views ./internal/i18n ./internal/mail
go test ./internal/handler -run SystemMonitor
node scripts/test-system-monitor.cjs
```

Run other tests relevant to upstream's changes, or `go test ./...` for a large update. A test that fails here should be checked against `upstream/master` in the same environment before blaming my branches. In the app, check the system monitor and Outlook folders before relying on the new build.

Publish it. A rebuild replaces `pi`'s history, so the push has to replace GitHub's copy; `--force-with-lease` refuses if GitHub's `pi` changed since I last fetched it:

```sh
git push --force-with-lease origin pi
```

Then [update the Pi](./docs/raspberry-pi-fork.md#update-without-losing-my-changes-or-data).

### When a rebuild stops on a conflict

The script stops if a branch conflicts with `upstream/master` or with a branch merged before it, and names the files. Generated `*_templ.go` files never stop it, as they're always regenerated.

1. Open the rebuild worktree it names, `../gofer-pi-rebuild`.
2. Edit each listed file to keep the intended behavior from both sides, and remove the `<<<<<<<`, `=======` and `>>>>>>>` markers.
3. Stage the files with `git add`, then `git commit --no-edit`. Git records the resolution (`rerere`).
4. Run `fork/rebuild-pi.sh` again. It starts over and applies the recorded resolution by itself, now and in later rebuilds, as long as the conflicting lines stay the same.

If the conflict means upstream changed the same code for the same purpose, compare the two. Often the better fix is to update my branch instead: switch to it, rebase it onto `upstream/master` (`git rebase upstream/master`), resolve the conflict there once, and rebuild. A branch that will become a pull request needs that anyway.

## Work on a change

Start every change from upstream, not from `pi`:

```sh
git fetch upstream
git switch -c fix/descriptive-name upstream/master
# Make the change, with a test that fails without it, and commit.
```

Add the branch to `fork/branches.txt` on `fork/tools`, then rebuild:

```sh
git switch fork/tools
# Add the branch name to fork/branches.txt.
git commit -am "Build pi with fix/descriptive-name"
git switch pi
fork/rebuild-pi.sh
```

To change something already on `pi`, commit to its branch and rebuild. The same goes for these notes: they're on `fork/tools`.

### When upstream takes a change

Upstream squashes pull requests, so Git can't tell that my commits are already there. Remove the branch's line from `fork/branches.txt` on `fork/tools`, rebuild, and delete the branch once the rebuilt `pi` works. If upstream implemented it differently, compare the behavior and prefer upstream's version unless mine does something theirs doesn't; then the branch stays, reduced to the difference.

## Open a pull request

A `fix/*` or `feature/*` branch already holds only its change, based on upstream. Bring it up to date and check it:

```sh
git fetch upstream
git switch fix/descriptive-name
git rebase upstream/master
git diff --stat upstream/master...HEAD
git log --format=full upstream/master..HEAD
git push --force-with-lease -u origin fix/descriptive-name
```

Open the PR from that branch to `cristianadrielbraun/gofer:master`. Keep its title and summary limited to that change, and follow upstream's [CONTRIBUTING.md](./CONTRIBUTING.md): every fix needs a test that fails without it, and commits and summaries carry no Claude/AI author trailers or generated-by footer. Inspect the commit messages before submitting. Return to `pi` with `git switch pi` afterward.

## Local checkout reminders

`task clean` and `task release:all` delete `dist/`. In this checkout it only holds build outputs, so that is safe, but move anything there that should be kept first.

Keep these notes updated when the branch arrangement or deployment changes. Never put actual passwords, OAuth secrets, or setup tokens in Git.
