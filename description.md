# Cache Vent (working name)

> See what every AI and dev tool is storing on your disk, what it is, and what's safe to delete.

_Name is a placeholder. Check GitHub / domain / trademark availability before publishing._

## The problem

AI and developer tools quietly fill disks faster than any software category before them:

- The same 8 GB model is stored in Ollama, LM Studio, the Hugging Face cache, and ComfyUI.
- Every old project keeps its `node_modules`, `.venv`, `target/`, and build caches.
- Agents (Claude Code, Codex, Cursor, Antigravity, …) keep session logs, snapshots, and caches that only grow.
- Docker / WSL virtual disks grow and never shrink.

Existing tools (WinDirStat, WizTree, DaisyDisk) show **how big** a folder is. None of them say **what it is**, **whether you need it**, or **what breaks if you delete it**. So people guess, get scared, and ignore it until the disk is full.

## The core feature

Every folder gets a plain-language identity and a verdict:

```
Hugging Face model cache                                    47.2 GB
  Contains:   Llama-3-8B, Qwen2.5-7B, SDXL base
  Duplicate:  Llama-3-8B also in Ollama, LM Studio          16.0 GB
  Verdict:    ✅ safe to deduplicate (no re-download)
  Last used:  73 days ago
```

The product is **correct explanations**. One wrong "safe to delete" label kills trust, so accuracy beats features and UI.

## What makes it different

1. **Open definitions database.** A public, community-maintained list of what every tool stores, where, and whether it regenerates. The app is easy to copy; the database isn't. See [DEFINITIONS.md](DEFINITIONS.md).
2. **Model deduplication with hardlinks.** Identical model files across apps get stored once. Nothing is re-downloaded and no app notices.
3. **Quarantine, not delete.** Cleaned items are moved to quarantine, auto-purged after 14 days, and can be restored with one command. This removes the fear of cleaning.
4. **Bloat timeline.** Sizes are recorded on each scan: "Cursor grew 12 GB this week."
5. **Agent-native.** CLI with `--json` and an MCP server, so AI agents can ask "what's eating my disk?" and clean up after themselves.

## Who it's for

1. **Primary:** People running local AI (models, ComfyUI, Ollama) and heavy AI-coding-agent users. They have the biggest, most confusing bloat and share tools online.
2. **Secondary:** Any developer with years of old projects.

## Non-goals (for now)

- Browser cache cleaning (it refills within days; CCleaner already does it).
- Cloud-side AI memory (ChatGPT, Claude.ai). There's no public access to it.
- Syncing memory or rules across agents (`AGENTS.md` and existing tools cover this).
- Registry cleaning, "PC optimization," or anything scareware-adjacent.
- A pretty GUI before the scanner is correct.

## Roadmap

| Phase | Ships | Done when |
|---|---|---|
| **0. Validate** | Mockup screenshot posted on r/LocalLLaMA and X | People ask "where can I get it?" |
| **1. Scan (read-only)** | CLI scan, ~15 tool definitions, regenerable/user-data/unknown verdicts, `--json` | Runs on the author's machine and the totals match Explorer |
| **2. Dedupe** | Find identical model files across apps and replace copies with hardlinks (same volume only) | Reclaims space and every app still loads the model |
| **3. Quarantine** | `clean` moves items to quarantine, `restore`, auto-purge after 14 days | Restore puts back byte-identical files |
| **4. Timeline** | Per-scan size history, "grew X this week" | Two scans show a diff |
| **5. Local web UI** | `Cache Vent ui` serves a local page from the same binary | Non-CLI users can use it |
| **6. MCP server** | `Cache Vent mcp` exposes scan / explain / quarantine as tools | Claude Code can call it |
| **Later** | Read-only viewer for local agent memory and rules (`~/.claude`, `.cursor`, `AGENTS.md`) | Only if users ask |

Windows first (the author's platform), with macOS and Linux paths in the definitions from day one.

## Success metrics

- Phase 0: 50+ "I want this" replies or signups.
- Phase 1–2: GitHub stars, and screenshots of "reclaimed X GB" posted by users.
- Definitions contributed by people other than the author. This is the real signal of a moat.
