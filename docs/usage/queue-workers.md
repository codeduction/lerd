# Queue Workers & Framework Workers

Lerd can run framework-defined workers as persistent systemd user services. Workers run inside the project's PHP-FPM container and restart automatically on failure.

## Queue worker

| Command | Description |
|---|---|
| `lerd queue:start` | Start the queue worker for the current project |
| `lerd queue:stop` | Stop the queue worker for the current project |
| `lerd queue start` | Same as `queue:start` (subcommand form) |
| `lerd queue stop` | Same as `queue:stop` (subcommand form) |

Works for any framework that defines a `queue` worker. Laravel has it built-in (`php artisan queue:work`).

## Generic workers (`lerd worker`)

Use this for any other framework-defined worker:

| Command | Description |
|---|---|
| `lerd worker start <name>` | Start a named worker for the current project |
| `lerd worker stop <name>` | Stop a named worker |
| `lerd worker list` | List all workers defined for this project's framework |

Example — start the Symfony Messenger consumer:
```bash
lerd worker start messenger
# Systemd unit: lerd-messenger-myapp.service
# Logs: journalctl --user -u lerd-messenger-myapp -f
```

Workers are defined in framework YAML definitions at `~/.config/lerd/frameworks/`. To add a custom worker to Laravel (e.g. Horizon):
```bash
lerd framework add laravel --from-file horizon.yaml
# or via the UI toggle in the Sites panel
```

---

## Options for `queue:start`

| Flag | Default | Description |
|---|---|---|
| `--queue` | `default` | Queue name to process |
| `--tries` | `3` | Max attempts before marking a job as failed |
| `--timeout` | `60` | Seconds a job may run before timing out |

---

## Redis requirement

If `QUEUE_CONNECTION=redis` is set in the project's `.env`, lerd verifies that `lerd-redis` is running before starting the worker. If it is not, you will see:

```
queue worker requires Redis (QUEUE_CONNECTION=redis in .env) but lerd-redis is not running
Start it first: lerd services start redis
```

---

## Example

```bash
cd ~/Lerd/my-app
lerd queue:start --queue=emails,default --tries=5 --timeout=120
# Systemd unit: lerd-queue-my-app.service
# Logs: journalctl --user -u lerd-queue-my-app -f
```

---

## Auto-restart on config changes

The lerd watcher daemon monitors `.env`, `composer.json`, `composer.lock`, and `.php-version` for every registered site. When any of those files change, it automatically signals `php artisan queue:restart` inside the PHP-FPM container (debounced). This ensures queue workers reload after deploys or PHP version changes without manual intervention.

---

## Web UI control

Queue workers are controllable from the **Sites tab** in the web UI. The amber toggle starts or stops the worker. When a worker is running, a **Queue** log tab appears in the site detail panel alongside PHP-FPM. The amber dot next to the site in the sidebar indicates the worker is active.
