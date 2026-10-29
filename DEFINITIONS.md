# Definitions

A definition describes one thing a tool stores on disk: where it is, what it is, and what happens if you delete it. The definitions are the heart of the project. **A wrong definition is a bug that can destroy someone's data**, so every one needs a source.

## Format

One YAML file per tool in `definitions/`. A file can hold several entries (e.g. Cursor's cache and its workspace storage are different verdicts).

```yaml
tool: Ollama
homepage: https://ollama.com
entries:
  - id: ollama-models
    name: Ollama models
    paths:
      windows: ["%USERPROFILE%\\.ollama\\models"]
      macos:   ["~/.ollama/models"]
      linux:   ["~/.ollama/models", "/usr/share/ollama/.ollama/models"]
    env_override: OLLAMA_MODELS       # if set, use this path instead
    kind: model
    safety: redownloadable
    if_deleted: Models must be pulled again with `ollama pull`. No settings are lost.
    dedupe: true                       # model weights; safe to hardlink
    verified:
      by: your-github-handle
      on: 2026-10-29
      version: "0.12"
      source: https://github.com/ollama/ollama/blob/main/docs/faq.md
```

For things found anywhere (not at a fixed path), use `pattern` instead of `paths`:

```yaml
  - id: node-modules
    name: node_modules (per project)
    pattern:
      dir_name: node_modules
      siblings: [package-lock.json, pnpm-lock.yaml, yarn.lock]   # only match next to one of these
    kind: dependencies
    safety: redownloadable
    if_deleted: Run `npm install` (or pnpm / yarn) in the project to restore.
```

## Fields

| Field | Required | Meaning |
|---|---|---|
| `id` | yes | Unique, kebab-case, stable forever (history and quarantine refer to it) |
| `name` | yes | What the user sees |
| `paths` or `pattern` | yes | Where to find it. Use `%VAR%` on Windows and `~` elsewhere |
| `pattern.siblings` | no | Match only next to one of these files or folders. Pick something that marks a real project (a lockfile, not just `package.json`) |
| `kind` | yes | `model`, `cache`, `dependencies`, `build-output`, `logs`, `session-history`, `virtual-disk`, `user-data`, `temp` |
| `safety` | yes | See below |
| `if_deleted` | yes | One plain sentence: what the user loses and how to get it back |
| `env_override` | no | An environment variable that relocates the folder |
| `dedupe` | no | `true` only for read-only model weights |
| `clean_older_than_days` | no | Clean only the top-level children untouched for this many days, not the whole folder (temp folders) |
| `verified` | yes, unless `unknown` | Who checked it, when, against which version, and the source. `source` is required for `regenerable` and `redownloadable`. `by: draft` means taken from docs and not yet checked on a real install |

Loading fails on unknown fields, so typos don't pass silently. `go test ./internal/defs` validates every shipped file.

## Safety levels

| Level | Meaning | Example |
|---|---|---|
| `regenerable` | Rebuilt automatically or with one command, offline | `__pycache__`, `target/` |
| `redownloadable` | Comes back, but needs a download (possibly large) | Ollama models, pip cache, `node_modules` |
| `user-data` | Contains something only the user has. **Never suggested for cleaning.** | Claude Code session history, ComfyUI outputs |
| `unknown` | Not yet verified. Shown, never suggested for cleaning | Anything new |

When in doubt, use `unknown`. A conservative label costs a few GB; a wrong one costs someone's work.

## Contributing a definition

1. Find the path in the tool's official docs or source code, and link it in `verified.source`.
2. Check it on a real install and note the tool version.
3. Write `if_deleted` from experience: delete it on a test machine and see what happens.
4. Open a pull request with one tool per PR.

## Starter set (phase 1)

All entries are `by: draft`: sourced from docs, not yet checked on a real install. Anything without an official source is `unknown`.

| Tool | Entries | Status |
|---|---|---|
| Ollama | models | done |
| LM Studio | models (`unknown`: imported models can't be re-downloaded) | done |
| Hugging Face | hub cache | done |
| ComfyUI | models (`unknown`), outputs (`user-data`) | done |
| Stable Diffusion WebUI / Forge | models (`unknown`) | done |
| Claude Code | session history (`user-data`), shell snapshots, todos | done |
| Claude Desktop | app caches | done |
| Codex CLI | sessions (`user-data`) | done |
| Cursor | Cache / CachedData, workspace storage (`user-data`) | done |
| VS Code | Cache / CachedData, extensions | done |
| Antigravity | caches | paths to be researched |
| npm / pnpm / yarn | caches and stores (pnpm `unknown`) | done |
| pip / uv | caches | done |
| Docker Desktop | virtual disk (`user-data`) | done |
| WSL | distro virtual disks | needs wildcard paths |
| Windows | user temp folder (7-day rule) | done |
| Per-project | `node_modules`, `target/`, `.gradle`, `__pycache__` | done |
| Python environments | venv, pipenv, poetry, conda | in `internal/envs`, not YAML |
