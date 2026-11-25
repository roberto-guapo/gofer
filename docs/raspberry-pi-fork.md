# Installing my fork on a Raspberry Pi

This is the deployment companion to [my fork-maintenance notes](../FORK-MAINTENANCE.md). It installs the `pi` branch, including my unmerged changes. The [general Raspberry Pi guide](./raspberry-pi.md) instead starts from an upstream release archive.

Commands below run in Bash on the Pi, not Windows PowerShell. The example layout is:

| Location | Contents |
| --- | --- |
| `~/src/gofer` | Git checkout and build tools' generated files |
| `~/gofer` | Installed binary and runtime data |
| `~/gofer/data` | Default database, mail, keys, and other application data |
| `/etc/gofer.env` | Service settings, including OAuth credentials if configured |
| `/etc/systemd/system/gofer.service` | Startup service |

Keep source and runtime data separate. If an installation already exists, inspect its service and environment file and adapt these paths before updating it. These paths are examples, not a record of the current Pi's actual configuration. A custom `GOFER_DB_PATH` or external storage location also needs to be backed up.

## Requirements and source checkout

Use 64-bit Raspberry Pi OS and a regular user account. Check the architecture:

```sh
uname -m
```

It should print `aarch64`; the binary target is `linux/arm64`. Install Git, curl, and CA certificates if needed. Install the ARM64 version of [Go](https://go.dev/doc/install) at the version required by `go.mod` or a compatible newer version. The current branch requires Go 1.27.1. Install [Task v3](https://taskfile.dev/docs/installation) as well.

After the development computer has pushed the `pi` branch:

```sh
mkdir -p ~/src
git clone --branch pi https://github.com/roberto-guapo/gofer.git ~/src/gofer
cd ~/src/gofer
git branch --show-current
git log -1 --oneline
go version
task --version
```

For an existing clone, use it instead of cloning over it. Confirm that its `origin` is my fork and that the checked-out branch is `pi`.

### Install the asset-generation tools

Install templ at the version used by this checkout:

```sh
cd ~/src/gofer
go install github.com/a-h/templ/cmd/templ@"$(go list -m -f '{{.Version}}' github.com/a-h/templ)"
export PATH="$(go env GOPATH)/bin:$HOME/.local/bin:$PATH"
```

Install the [Tailwind standalone CLI](https://tailwindcss.com/blog/standalone-cli) for Linux ARM64. This snapshot uses v4.2.4, matching the repository's `Dockerfile`. Recheck that version after upstream updates. Use the regular ARM64 asset for Raspberry Pi OS; the musl asset in the Dockerfile is for Alpine Linux.

```sh
mkdir -p ~/.local/bin
curl -fL https://github.com/tailwindlabs/tailwindcss/releases/download/v4.2.4/tailwindcss-linux-arm64 -o ~/.local/bin/tailwindcss
chmod +x ~/.local/bin/tailwindcss
templ version
tailwindcss --help
```

Keep the Go bin directory and `~/.local/bin` on `PATH` in future build sessions. Node.js is needed for the JavaScript regression tests in the maintenance notes, but not for this release build: the Taskfile runs esbuild through Go.

## Build my version

From the Pi's source checkout:

```sh
cd ~/src/gofer
git status --short
task release
```

`task release` generates templates, CSS, and minified JavaScript, then builds `dist/gofer` with embedded assets. A plain `go build` does not replace those generation steps. Keep the previous installation running until the new build succeeds. The first build downloads dependencies and may take a while on a Pi.

If generated tracked files change, inspect them before the next Git update; do not discard unexplained changes. Confirm that the templ version still matches the checkout.

## First installation

For a new installation only, after a successful build:

```sh
mkdir -p ~/gofer
install -m 755 ~/src/gofer/dist/gofer ~/gofer/gofer
git -C ~/src/gofer rev-parse HEAD > ~/gofer/BUILD-COMMIT
```

For an existing installation, use the backup-and-update procedure below instead.

Choose how to reach Gofer before starting the service:

- [Tailscale HTTPS](./tailscale.md): private HTTPS access from your connected devices. Use its native-binary settings in `/etc/gofer.env`, with the actual Serve URL as `GOFER_BASE_URL`.
- [Private LAN HTTP](./lan-access.md): use the Pi's private IP and the correct network range. A DHCP reservation keeps the address stable. Browser traffic is unencrypted with this option.
- Another HTTPS reverse proxy: follow the [deployment precautions](./setup.md#deployment-precautions).

Follow [running Gofer as a Linux service](./linux-service.md). Replace `youruser` in the service with the result of `whoami`, and confirm that `WorkingDirectory` and `ExecStart` point to `~/gofer`, not `~/src/gofer`. Keep `/etc/gofer.env` readable by root only. Use `personal` authentication for one profile, or `managed` for separate users.

Open the configured address, find the first-run token in `journalctl -u gofer`, and complete setup. See [Gmail and Google Workspace](./gmail.md) for Google accounts. Store OAuth credentials in the service environment file, never in the source repository.

Check the installation with:

```sh
systemctl status gofer
journalctl -u gofer -n 50 --no-pager
curl -fsS http://127.0.0.1:8090/readyz
cat ~/gofer/BUILD-COMMIT
```

Adjust the health-check port if changed. Check **Settings → About** and the fork's own features, such as the system monitor. The source commit record identifies what was installed even though it is not an upstream release.

## Update without losing my changes or data

First rebuild `pi`, test it, and push it from the development computer using the [maintenance workflow](../FORK-MAINTENANCE.md#update-pi-with-the-authors-improvements). On the Pi, update from my fork:

```sh
cd ~/src/gofer
git status --short
git fetch origin
git reset --keep origin/pi
task release
```

`pi` is rebuilt rather than added to, so its history changes with each update and `git pull` can't follow it. `git reset --keep` moves the checkout to the new `pi` instead. It refuses, and changes nothing, if a file it would replace has local changes. This checkout is only used for building, so `git status --short` should print nothing. If it lists files, inspect them before updating; don't discard unexplained changes. Run the build only after the reset succeeds. Recheck tool versions if the build requirements changed.

After the new build succeeds, stop Gofer and make a matching backup of the installed binary, data, configuration, and service. The following subshell stops on the first error. If it fails after stopping the service, inspect the failure before proceeding; do not replace the binary without a successful backup.

```sh
(
  set -eu
  FORK_BACKUP="$HOME/gofer-backup-$(date +%Y%m%d-%H%M%S)"
  mkdir -m 700 "$FORK_BACKUP"
  sudo systemctl stop gofer
  cp -a "$HOME/gofer" "$FORK_BACKUP/runtime"
  sudo cp -p /etc/gofer.env "$FORK_BACKUP/gofer.env"
  sudo cp -p /etc/systemd/system/gofer.service "$FORK_BACKUP/gofer.service"
  install -m 755 "$HOME/src/gofer/dist/gofer" "$HOME/gofer/gofer.new"
  mv "$HOME/gofer/gofer.new" "$HOME/gofer/gofer"
  git -C "$HOME/src/gofer" rev-parse HEAD > "$HOME/gofer/BUILD-COMMIT"
  sudo systemctl start gofer
  printf 'Backup: %s\n' "$FORK_BACKUP"
)
```

Check the service logs, readiness endpoint, browser sign-in, and custom features again. Migrations can delay readiness. Copy the backup off the Pi as well; a backup on the same card or disk does not protect against that device failing.

An older binary may not open a database migrated by a newer one. To roll back, stop the service, preserve the failed runtime directory separately, restore the **matching backup's entire runtime directory** and configuration, and restore the service file if it changed. Run `sudo systemctl daemon-reload` if restoring the unit, then start Gofer and check it again. Restoring a backup also loses changes made since that backup, so preserve the newer data before restoring.

Do not replace this custom build with an upstream archive or the default upstream Docker image if the goal is to retain my patches. An upstream update notification is a reminder to merge and rebuild, not a replacement for this workflow.

## What has and has not been verified

These instructions were checked against this branch's Taskfile, Dockerfile, build scripts, and configuration examples. The installation and update commands have not been executed on the actual Raspberry Pi as part of writing these notes. Record any machine-specific adjustments here after confirming them, without recording secrets.
