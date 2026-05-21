# Changelog

## v0.0.22 - 2026-05-21

- Tweak the threshold for "Experience" label detection again...
- Separate various containers into tabs.
- Move the usage instructions from `README.md` to a "Help" tab.

## v0.0.21 — 2026-05-14

- Add online counter.
- Make ping stats easier to read.
- Lower the threshold for "Experience" label detection.

## v0.0.20 — 2026-05-08

- Add the CHANGELOG link to updater pop-up.

## v0.0.19 — 2026-05-08

- Add autoupdater.
- New `updater/` module for self-update binary replacement.
- Server-side update check and execute endpoints (`/api/update/check`, `/api/update/execute`).
- Autorelease workflow updated to build and include `updater.exe` asset.

## v0.0.18 — 2026-05-08

- Add experimental Relic support.
- New `settings/` package for configuration storage and preset management.
- Proxy ping stats support.
- Preset switching with server restart.

## v0.0.17 — 2026-05-06

- Add RTT/packet loss stats.
- New `ping/` package for network diagnostics.
- Minor CSS changes.
- Major UI refactor of static assets.

## v0.0.16 — 2026-05-01

- Don't create a new Windows callback for each screenshot. This should fix long session panics.
- Refactored screenshot capture logic.
- Improved server and main startup flow.

## v0.0.15 — 2026-04-30

- Add session exp / duration tracking.
- Extended exp cache with session statistics.
- UI updates for session display.

## v0.0.14 — 2026-04-29

- Fix background healthchecks, this time for real.
- Corrected keepalive logic in the frontend.

## v0.0.13 — 2026-04-29

- Don't die when tab goes to background.
- Fixed visibility-related issues in the frontend polling loop.

## v0.0.12 — 2026-04-28

- Improve lifetime management.
- Replaced system tray with keepalive-based server lifecycle.
- Removed `tray/` and `assets/` packages.
- Embedded favicon directly in the server.
- Browser auto-opens on server start.

## v0.0.11 — 2026-04-28

- Logging: append instead of truncating.

## v0.0.10 — 2026-04-27

- Add crash logging.

## v0.0.9 — 2026-04-27

- Automatic version check.
- Frontend fetches latest GitHub release and shows update notification.
- Version endpoint added to server.
- Build script passes version via `-ldflags`.

## v0.0.8 — 2026-04-27

- Expose the favicon on the webpage as well.
- Added `favicon.ico` asset.
- Cleaned up tray icon embedding.

## v0.0.7 — 2026-04-27

- Auto-reset exp counter on XP decrease.

## v0.0.6 — 2026-04-27

- Auto-close webpage when server is gone.
- Keepalive mechanism between frontend and server.

## v0.0.5 — 2026-04-27

- Simplify the logic of exp tracker.
- Major refactor of exp cache and server handler.
- UI improvements for exp display.

## v0.0.4 — 2026-04-22

- Exp: add separator commas for number formatting.

## v0.0.3 — 2026-04-22

- Fix exp/h counters.
- Add a note on re-focusing the game window.
- Exp reader improvements.

## v0.0.2 — 2026-04-22

- Fix first run issues.
- Create accounts directory if missing.
- Return `[]` instead of `nil` in JSON responses.

## v0.0.1 — 2026-04-22

- Initial release.
- Account switcher with registry snapshot/restore.
- Experience tracker with OCR-based screen reading.
- Web-based UI with embedded static assets.
- GitHub Actions autorelease workflow.
