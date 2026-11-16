# Optional system monitor

On Linux, including Raspberry Pi OS, open **Settings → Appearance → Show system monitor**. It is off by default. The widget sits above the sidebar footer in mail, contacts, calendar, and settings, using the active theme's colors and fonts. Click **System** to expand the details. Choose **Refresh frequency** beside the monitor setting: 5, 15 (default), 30, or 60 seconds. Slower refreshes reduce collection and browser requests.

This first version is for open and personal installations. It is not exposed to users of a managed installation. Open mode retains its normal behavior: anyone allowed to access the application can use the owner's settings and monitor. Windows and macOS show a disabled setting with a Linux availability note.

## Readings

- **CPU:** total system utilization between samples, excluding idle and I/O wait. The first reading, or the first reading after a viewing gap, shows a dash until the next refresh.
- **Memory:** system RAM in use, calculated using Linux's `MemAvailable` so reclaimable cache is not treated as unavailable memory. Hover over the percentage for used and total bytes.
- **Temperature:** a recognized CPU/SoC thermal sensor in degrees Celsius. Missing or inaccessible sensors show a dash.
- **Data disk:** usage of the filesystem containing the current settings database, including space reserved from ordinary users. It does not add up separate mail-storage volumes. Hover for used and total bytes.
- **Uptime:** system uptime.
- **Gofer memory:** the Gofer process's resident memory (RSS), including resident shared pages; this is not just the Go heap.

Readings describe the Linux environment visible to Gofer. In a container, `/proc`, thermal sensors, and the database filesystem may describe different host/container scopes. This version does not calculate cgroup-relative CPU or memory usage, and does not require privileged container access. Do not grant extra privileges just to populate an unavailable reading.

## Resource use

The browser requests a snapshot at the selected interval while the widget is on screen. It stops when the tab is hidden, the mobile sidebar is closed, or the preference is disabled. Requests time out after five seconds; failed readings are marked unavailable and retried.

The server collects on demand and shares a cache lasting the selected interval across viewers of the same filesystem path. There is no collection ticker, subprocess, new dependency, or history database. The endpoint checks the saved preference before reading system statistics. It returns no data when disabled and is protected by the existing authentication middleware plus owner-mode checks.

## Maintaining the fork

The collector lives in `internal/systemmonitor`, with a separate handler, template, script, and stylesheet. Integration is limited to the shared sidebar, Appearance settings, navigation scripts, stylesheet import, and the two route-registration paths. Keep these integration calls when resolving upstream merge conflicts. Regenerate templates and production assets from the merged sources; do not hand-edit generated templates.

Relevant checks:

```sh
go test ./internal/systemmonitor ./internal/views ./internal/i18n
go test ./internal/handler -run SystemMonitor
node scripts/test-system-monitor.cjs
node scripts/test-client-navigation.cjs
node scripts/test-section-transitions.cjs
task release
```

Run the collector tests on Linux to exercise real `/proc` and filesystem readings. An ARM64 release build verifies Raspberry Pi compilation; actual resource overhead and sensor availability must still be checked on the Pi. The development preview was tested on Linux under WSL, not on Raspberry Pi hardware.
