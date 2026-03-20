# Node

## Commands

| Command | Description |
|---|---|
| `lerd node:install <version>` | Install a Node.js version globally via fnm |
| `lerd node:uninstall <version>` | Uninstall a Node.js version via fnm |
| `lerd isolate:node <version>` | Pin Node version for cwd — writes `.node-version`, runs `fnm install` |

---

## Usage

`lerd install` places shims for `node`, `npm`, and `npx` in `~/.local/share/lerd/bin/`, which is added to your `PATH`. You use them exactly as you normally would — lerd picks the right version automatically:

```bash
node --version
npm install
npx tsc --init
```

---

## Version resolution

1. `.nvmrc` in the project root
2. `.node-version` in the project root
3. `package.json` — `engines.node` field
4. Global default in `~/.config/lerd/config.yaml`

To pin a project to a specific version:

```bash
cd ~/Lerd/my-app
lerd isolate:node 20
# writes .node-version and installs Node 20 via fnm
```

To install a version without pinning a project:

```bash
lerd node:install 22
```

---

## fnm

Node version management is handled by [fnm](https://github.com/Schniz/fnm), which is bundled and installed automatically. The `node`, `npm`, and `npx` shims in `~/.local/share/lerd/bin/` invoke the correct version via fnm for each project.
