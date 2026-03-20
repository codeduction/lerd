# Web UI

Lerd includes a browser dashboard, served at **`http://127.0.0.1:7073`** by the `lerd-ui` systemd service (started automatically with `lerd install`).

---

## Screenshots

![Sites tab](../assets/screenshots/app-3.png)
*Sites tab — all registered projects with controls*

![Services tab](../assets/screenshots/app-2.png)
*Services tab — start/stop services and copy `.env` values*

![System tab](../assets/screenshots/app-1.png)
*System tab — DNS, nginx, and PHP-FPM health*

---

## Sites tab

Lists all registered projects with their domain, path, PHP version, Node version, and per-site controls:

- **HTTPS toggle** — enable or disable TLS with one click; updates `APP_URL` in `.env` automatically
- **PHP / Node dropdowns** — change the version per site; writes `.php-version` / `.node-version` into the project and regenerates the nginx vhost on the fly
- **Queue toggle** — start or stop the Laravel queue worker for a site; amber when running; click the **logs** link next to the toggle to open the live log drawer for that worker
- **Unlink button** — remove a site from nginx without touching the terminal; for parked sites the directory is left on disk (run `lerd link` to re-register it)
- **Click any row** — opens the live PHP-FPM log drawer at the bottom of the screen

## Services tab

Shows all available services (MySQL, Redis, PostgreSQL, Meilisearch, MinIO, Mailpit, Soketi) with their current status. Start or stop any service with one click; each panel shows the correct `.env` connection values with a one-click copy button.

## System tab

Health check panel for DNS, nginx, PHP-FPM containers, installed Node.js versions, and the autostart toggle.

The **Node.js card** lists all versions installed via fnm and includes an inline install form — enter a version number (e.g. `22`) and click **Install**. This is equivalent to running `lerd node:install <version>` from the terminal.

## Updates

Shows the current and latest version. When an update is available, a notice with the version number is shown alongside an instruction to run `lerd update` in a terminal (the update requires `sudo` for sysctl/sudoers steps and cannot run in the background).
