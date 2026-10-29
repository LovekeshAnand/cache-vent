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
      sibling: package.json            # only match next to this file
    kind: dependencies
    safety: regenerable
    if_deleted: Run `npm install` (or pnpm / yarn) in the project to restore.
```

## Fields

| Field | Required | Meaning |
|---|---|---|
| `id` | yes | Unique, kebab-case, stable forever (history and quarantine refer to it) |
| `name` | yes | What the user sees |
| `paths` or `pattern` | yes | Where to find it. Use `%VAR%` on Windows and `~` elsewhere |
| `kind` | yes | `model`, `cache`, `dependencies`, `build-output`, `logs`, `session-history`, `virtual-disk`, `user-data` |
| `safety` | yes | See below |
| `if_deleted` | yes | One plain sentence: what the user loses and how to get it back |
| `env_override` | no | An environment variable that relocates the folder |
| `dedupe` | no | `true` only for read-only model weights |
| `verified` | yes | Who checked it, when, against which version, and the source |

## Safety levels

| Level | Meaning | Example |
|---|---|---|
| `regenerable` | Rebuilt automatically or with one command, offline | `node_modules`, `__pycache__`, `target/` |
| `redownloadable` | Comes back, but needs a download (possibly large) | Ollama models, pip cache |
| `user-data` | Contains something only the user has. **Never suggested for cleaning.** | Claude Code session history, ComfyUI outputs |
| `unknown` | Not yet verified. Shown, never suggested for cleaning | Anything new |

When in doubt, use `unknown`. A conservative label costs a few GB; a wrong one costs someone's work.

## Contributing a definition

1. Find the path in the tool's official docs or source code, and link it in `verified.source`.
2. Check it on a real install and note the tool version.
3. Write `if_deleted` from experience: delete it on a test machine and see what happens.
4. Open a pull request with one tool per PR.

## Starter set (phase 1)

All paths still need verifying before they ship.

| Tool | Entries |
|---|---|
| Ollama | models |
| LM Studio | models |
| Hugging Face | hub cache |
| ComfyUI | models, outputs (`user-data`) |
| Stable Diffusion WebUI / Forge | models |
| Claude Code | session history (`user-data`), shell snapshots, todos |
| Claude Desktop | app caches |
| Codex CLI | sessions (`user-data`) |
| Cursor | Cache / CachedData, workspace storage |
| VS Code | Cache / CachedData, extensions |
| Antigravity | caches (paths to be researched) |
| npm / pnpm / yarn | global caches and stores |
| pip / uv | caches |
| Docker Desktop / WSL | virtual disks |
| Per-project | `node_modules`, `.venv`, `target/`, `.gradle`, `__pycache__` |
