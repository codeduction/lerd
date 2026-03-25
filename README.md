# Lerd

**A Herd-like local PHP development environment for Linux** — Podman-native, rootless, zero system dependencies.

Lerd bundles Nginx, PHP-FPM, and optional services (MySQL, Redis, PostgreSQL, and more) as rootless Podman containers, giving you automatic `.test` domain routing, per-project PHP/Node version isolation, and one-command TLS — all without touching your system's PHP or web server. Laravel-first, with built-in support for Symfony, WordPress, and any PHP framework via YAML definitions.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/geodro/lerd/main/install.sh | bash
```

## Documentation

**[geodro.github.io/lerd](https://geodro.github.io/lerd/)**

- [Requirements](https://geodro.github.io/lerd/getting-started/requirements/)
- [Installation](https://geodro.github.io/lerd/getting-started/installation/)
- [Quick Start](https://geodro.github.io/lerd/getting-started/quick-start/)
- [Services](https://geodro.github.io/lerd/usage/services/)
- [Command Reference](https://geodro.github.io/lerd/reference/commands/)
- [Changelog](https://geodro.github.io/lerd/changelog/)

## License

MIT
