# Lerd — TODO

Features to implement, roughly in priority order.

---

- [X] **Xdebug** — `lerd xdebug on/off` toggle per PHP version; rebuilds FPM image with Xdebug installed and configured
- [ ] **Wildcard TLS cert** — generate `*.test` at install time so every site is auto-HTTPS without running `lerd secure`
- [X] **Shell shims** — `php`, `composer`, `node` on `$PATH` that resolve to the project-local version automatically
- [X] **`lerd open`** — open the current site in the default browser
- [X] **Queue worker** — `lerd queue:start / stop` to run `artisan queue:work` as a managed systemd user service per site
- [X] **`lerd db`** shortcuts — `lerd db:import`, `lerd db:export` wrappers around mysql/psql inside the service containers
- [X] **`lerd fetch`** — pre-pull/build PHP images in the background so first use of a new version isn't slow
- [ ] **Custom PHP extensions** — `lerd php:ext add <ext>` adds an extension to the FPM image and rebuilds
- [X] **`lerd share`** — expose a local site publicly via an ngrok/Expose tunnel
- [ ] **Per-site env vars** — inject a `~/.config/lerd/env/<domain>.env` file into the FPM container at runtime
