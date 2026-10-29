import {
  ArrowCounterClockwise,
  CheckCircle,
  CloudSlash,
  FileMagnifyingGlass,
  Lightning,
  LinkBreak,
  LockKey,
  Question,
  ShieldCheck,
  TreeStructure,
  UserCircle,
} from "@phosphor-icons/react/dist/ssr";
import type { ReactNode } from "react";
import { DownloadButton } from "@/components/download-button";
import { InstallTabs } from "@/components/install-tabs";
import { Reveal, WordReveal } from "@/components/motion";
import { author, files, href, mb, site, version } from "@/lib/site";

/* ---------- content ---------- */

const piles = [
  { what: "node_modules in 35 projects", bytes: "10.9 GB", verdict: "safe" },
  { what: "Python virtual environments", bytes: "10.3 GB", verdict: "unknown" },
  { what: "Ollama models", bytes: "7.1 GB", verdict: "safe" },
  { what: "Cargo build output", bytes: "3.3 GB", verdict: "safe" },
  { what: "Docker Desktop virtual disk", bytes: "1.8 GB", verdict: "user" },
  { what: "Claude Code session history", bytes: "830 MB", verdict: "user" },
] as const;

const verdicts = {
  safe: { label: "Safe to free", cls: "bg-safe-soft text-safe" },
  unknown: { label: "Not verified", cls: "bg-mist text-muted" },
  user: { label: "Your data", cls: "bg-vent-soft text-vent-ink" },
} as const;

const features = [
  {
    Icon: FileMagnifyingGlass,
    title: "Every folder explained",
    body: "What it is, which tool made it, what happens if it goes, and a link to the documentation that says so. No guessing from folder names.",
  },
  {
    Icon: ArrowCounterClockwise,
    title: "Undo for 14 days",
    body: "Cleaning renames items into a quarantine on the same drive. It takes a moment, uses no extra space, and one click puts everything back.",
  },
  {
    Icon: TreeStructure,
    title: "Knows which project owns what",
    body: "Ties node_modules, venv, Poetry and pipenv environments to their projects, shows package sizes, and flags environments whose project is gone.",
  },
  {
    Icon: LockKey,
    title: "Private by construction",
    body: "No account, no telemetry, no network requests. The app runs in its own window on your machine and talks only to itself.",
  },
];

const steps = [
  {
    title: "Scan",
    body: "Pick your home folder, every drive, or specific folders. Reading changes nothing on disk and takes a few seconds.",
  },
  {
    title: "Read the verdict",
    body: "Each item is marked safe to free, not verified, or your data, with what happens if it goes and where that claim comes from.",
  },
  {
    title: "Move to quarantine",
    body: "Select what you want gone, or pick everything unused for 30 days. It moves aside at once and can be restored for 14 days.",
  },
];

const finds: { tool: string; what: string; verdict: keyof typeof verdicts | "mixed" }[] = [
  { tool: "Node.js", what: "node_modules per project, npm and Yarn caches", verdict: "safe" },
  { tool: "Python", what: "venv, Poetry and pipenv environments with package sizes", verdict: "mixed" },
  { tool: "Python", what: "pip and uv caches, __pycache__", verdict: "safe" },
  { tool: "Ollama, Hugging Face", what: "Downloaded models", verdict: "safe" },
  { tool: "Rust, Gradle", what: "Cargo target folders, project .gradle caches", verdict: "safe" },
  { tool: "Windows", what: "Your temp folder, older items only", verdict: "safe" },
  { tool: "LM Studio, ComfyUI, SD WebUI", what: "Local models", verdict: "unknown" },
  { tool: "VS Code, Cursor, Claude Desktop", what: "App caches and extensions", verdict: "unknown" },
  { tool: "Claude Code, Codex CLI, Cursor", what: "Session history and workspace storage", verdict: "user" },
  { tool: "Docker Desktop", what: "The virtual disk holding your images and volumes", verdict: "user" },
];

const safety = [
  { Icon: FileMagnifyingGlass, title: "Scanning only reads", body: "Nothing outside Cache Vent's own folder and quarantine is ever written." },
  { Icon: LinkBreak, title: "Links are never followed", body: "Symlinks and junctions are listed, never entered, so nothing outside a folder is touched." },
  { Icon: CheckCircle, title: "No source, no cleaning", body: "Only items backed by a cited source can be selected. Everything else stays not verified." },
  { Icon: UserCircle, title: "Your data stays put", body: "Chat histories, outputs and virtual disks are shown but can never be selected." },
  { Icon: ShieldCheck, title: "Checked again before every move", body: "Home folders, drive roots and anything still in use are refused at the last moment." },
  { Icon: CloudSlash, title: "Cloud folders are off limits", body: "Items inside OneDrive, Dropbox, iCloud or Google Drive folders are never moved." },
];

const speed = [
  { label: "Cache Vent", detail: "all of D:, 1.18 million files", s: 2.53, mem: "41 MB", ours: true },
  { label: "filepath.WalkDir", detail: "same drive, sizes only", s: 8.68, mem: "15 MB", ours: false },
];

const faq: { q: string; a: ReactNode; text: string }[] = [
  {
    q: "Is it safe to run on my main machine?",
    text: "Yes. Scanning only reads. Cleaning moves items into a quarantine on the same drive instead of deleting them, and anything moved can be restored for 14 days. Items marked your data or not verified cannot be selected at all.",
    a: "Yes. Scanning only reads. Cleaning moves items into a quarantine on the same drive instead of deleting them, and anything moved can be restored for 14 days. Items marked your data or not verified cannot be selected at all.",
  },
  {
    q: "What does not verified mean?",
    text: "It means no documented source yet confirms what happens when that folder is removed. Cache Vent still shows its size and what it is, but it will not offer to clean it. Verdicts only change when someone adds a source to the definition.",
    a: "It means no documented source yet confirms what happens when that folder is removed. Cache Vent still shows its size and what it is, but it will not offer to clean it. Verdicts only change when someone adds a source to the definition.",
  },
  {
    q: "Does Cache Vent send anything over the network?",
    text: "No. There is no account, no telemetry and no update check. The desktop window talks only to a local server bound to 127.0.0.1 and protected by a token that changes every launch.",
    a: "No. There is no account, no telemetry and no update check. The desktop window talks only to a local server bound to 127.0.0.1 and protected by a token that changes every launch.",
  },
  {
    q: "Why does Windows show a SmartScreen warning?",
    text: "The build is not code signed yet, so Windows does not recognise the publisher. Installing with the PowerShell command avoids the prompt and checks the file's SHA256 checksum for you. If you downloaded the app in a browser, choose More info, then Run anyway.",
    a: (
      <>
        The build is not code signed yet, so Windows does not recognise the publisher. Installing with the PowerShell command in the{" "}
        <a className="underline underline-offset-4" href="#download">install section</a> avoids the prompt and checks the file&apos;s SHA256 checksum for you. If you
        downloaded the app in a browser, choose <strong>More info</strong>, then <strong>Run anyway</strong>.
      </>
    ),
  },
  {
    q: "The Mac app isn't from an identified developer. How do I open it?",
    text: "Install it with the Terminal command instead of a browser download. Files fetched from Terminal are not flagged by Gatekeeper, and the installer signs Cache Vent.app on your Mac and clears the download flag, so it opens from Launchpad like any other app. No Apple account, admin password or System Settings changes are needed.",
    a: (
      <>
        Install it with the Terminal command instead of a browser download. Files fetched from Terminal aren&apos;t flagged by Gatekeeper, and the installer signs{" "}
        <strong>Cache Vent.app</strong> on your Mac and clears the download flag, so it opens from Launchpad like any other app. No Apple account, admin password or
        System Settings changes are needed. The script is short and readable:{" "}
        <a className="underline underline-offset-4" href="/install.sh">install.sh</a>.
      </>
    ),
  },
  {
    q: "Will removing node_modules or a virtual environment break my project?",
    text: "Only until you reinstall. Each item shows the command that brings it back, such as npm install or uv sync. If you would rather not, restore it from the quarantine in one click.",
    a: (
      <>
        Only until you reinstall. Each item shows the command that brings it back, such as <code className="font-mono text-sm">npm install</code> or{" "}
        <code className="font-mono text-sm">uv sync</code>. If you would rather not, restore it from the quarantine in one click.
      </>
    ),
  },
  {
    q: "Where does the quarantine live, and does it use space?",
    text: "In a .cachevent-quarantine folder on the same drive as the item, so moving is a rename and takes no extra space. The space comes back when items are purged, automatically after 14 days or sooner if you empty it.",
    a: "In a .cachevent-quarantine folder on the same drive as the item, so moving is a rename and takes no extra space. The space comes back when items are purged, automatically after 14 days or sooner if you empty it.",
  },
  {
    q: "Can I delete things straight away?",
    text: "Yes. Select items and choose Delete. You will see every path and its size and confirm before anything is removed. The same rules apply, so your data and unverified items still cannot be deleted.",
    a: "Yes. Select items and choose Delete. You will see every path and its size and confirm before anything is removed. The same rules apply, so your data and unverified items still cannot be deleted.",
  },
  {
    q: "Does it work on macOS and Linux?",
    text: "Yes. macOS 13 Ventura or later on Apple silicon and Intel, and Linux on x64 and ARM64. Windows gets its own native window and the fastest scanner; on macOS and Linux the interface opens in a Chrome, Edge or Brave app window, or your default browser. The Linux build is tested end to end on every change; macOS support is newer and marked as a preview.",
    a: "Yes. macOS 13 Ventura or later on Apple silicon and Intel, and Linux on x64 and ARM64. Windows gets its own native window and the fastest scanner; on macOS and Linux the interface opens in a Chrome, Edge or Brave app window, or your default browser. The Linux build is tested end to end on every change; macOS support is newer and marked as a preview.",
  },
  {
    q: "How is it this fast?",
    text: "On Windows it reads each folder in batches of about 64 KB of entries with sizes included, instead of one system call per file, and walks folders in parallel with a fixed number of workers. It never reads the same folder twice.",
    a: "On Windows it reads each folder in batches of about 64 KB of entries with sizes included, instead of one system call per file, and walks folders in parallel with a fixed number of workers. It never reads the same folder twice.",
  },
  {
    q: "Is it free?",
    text: "Yes. Cache Vent is free and open source under the MIT license, for personal and commercial use.",
    a: "Yes. Cache Vent is free and open source under the MIT license, for personal and commercial use.",
  },
];

/* ---------- pieces ---------- */

function Chip({ v }: { v: keyof typeof verdicts | "mixed" }) {
  if (v === "mixed") return <span className="rounded-md bg-mist px-2 py-0.5 text-xs font-medium text-muted">Per environment</span>;
  return <span className={`rounded-md px-2 py-0.5 text-xs font-medium ${verdicts[v].cls}`}>{verdicts[v].label}</span>;
}

function Heading({ eyebrow, title, body }: { eyebrow: string; title: ReactNode; body?: string }) {
  return (
    <div className="max-w-[680px]">
      <p className="text-sm font-semibold text-vent-ink">{eyebrow}</p>
      <h2 className="mt-3 text-3xl font-semibold tracking-tight md:text-4xl">{title}</h2>
      {body && <p className="mt-4 text-lg text-muted">{body}</p>}
    </div>
  );
}

/** Stairs echoes the banner: a flat orange staircase filling a corner (top right as drawn). */
function Stairs({ className }: { className: string }) {
  return (
    <svg viewBox="0 0 160 160" aria-hidden className={className}>
      <path d="M0 0H160V160H120V120H80V80H40V40H0Z" fill="#fc5e00" />
      <path d="M40 40H80V80H120V120H160V160H120V120H80V80H40Z" fill="#fd8a45" />
    </svg>
  );
}

/* ---------- page ---------- */

export default function Home() {
  const jsonLd = [
    {
      "@context": "https://schema.org",
      "@type": "SoftwareApplication",
      name: site.name,
      applicationCategory: "UtilitiesApplication",
      operatingSystem: "Windows 10, Windows 11, macOS, Linux",
      offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
      description: site.description,
      downloadUrl: new URL(href(files[0]), site.url).toString(),
      license: "https://opensource.org/licenses/MIT",
      codeRepository: site.github,
      author: { "@type": "Person", name: author.name, email: author.email, sameAs: [author.github, author.x, author.linkedin] },
    },
    {
      "@context": "https://schema.org",
      "@type": "FAQPage",
      mainEntity: faq.map((f) => ({ "@type": "Question", name: f.q, acceptedAnswer: { "@type": "Answer", text: f.text } })),
    },
  ];

  return (
    <>
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />

      {/* Hero */}
      <section className="relative overflow-hidden px-4 pb-16 pt-24 sm:px-6 md:pb-24">
        <Stairs className="pointer-events-none absolute right-0 top-0 hidden w-32 md:block lg:w-40" />
        <Stairs className="pointer-events-none absolute bottom-0 left-0 hidden w-32 rotate-180 md:block lg:w-40" />
        <div className="relative mx-auto mt-16 md:mt-24 flex max-w-6xl flex-col items-center text-center">
          <p className="inline-flex items-center gap-2 rounded-full border border-line bg-paper px-3 py-1 text-xs font-medium text-muted sm:text-sm">
            <span className="size-2 rounded-full bg-vent" aria-hidden />
            Free and open source · Windows, macOS, Linux
          </p>
          <h1 className="mt-6 max-w-[680px] bg-linear-to-r from-[#000000] to-[#666666] bg-clip-text pb-1 text-4xl font-semibold tracking-tight text-transparent sm:text-5xl md:text-6xl">
            Get back the gigabytes
            <br className="hidden sm:block" /> your dev tools took
          </h1>
          <p className="mt-6 max-w-[680px] text-lg text-muted">
            Cache Vent finds model weights, node_modules, virtual environments and caches on every drive, explains what each folder is,
            and moves only what is safe into a quarantine you can undo.
          </p>
          <div className="mt-10 flex w-full justify-center">
            <DownloadButton />
          </div>
          <p className="mt-8 text-sm text-muted">
            <Lightning size={16} weight="fill" className="mr-1 inline align-[-3px] text-vent" aria-hidden />
            Scanned <span className="font-mono text-ink">231,033</span> folders across two drives in <span className="font-mono text-ink">2.85 s</span> on a
            developer laptop
          </p>

          <Reveal className="mt-16 w-full">
            <figure className="relative mx-auto max-w-5xl rounded-2xl border border-line bg-mist p-2 shadow-[0_32px_80px_rgba(10,10,10,0.12)]">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src="/screenshot.webp"
                width={1200}
                height={820}
                alt="The Cache Vent window: 28.2 GB safe to free across drives C and D, with a map of node_modules, Python environments, Ollama models and Cargo build output."
                className="w-full rounded-lg border border-line"
              />
              <figcaption className="sr-only">Cache Vent on Windows 11 after scanning every drive.</figcaption>
            </figure>
          </Reveal>
        </div>
      </section>

      {/* Proof numbers */}
      <section aria-label="At a glance" className="px-6">
        <dl className="mx-auto grid max-w-6xl grid-cols-2 gap-px overflow-hidden rounded-2xl border border-line bg-line md:grid-cols-4">
          {[
            ["2.85 s", "to scan every drive on a laptop"],
            ["3.4×", "faster than a standard Go directory walk"],
            ["41 MB", "peak memory on a 1.18 million file drive"],
            ["0", "network requests, accounts or trackers"],
          ].map(([n, l]) => (
            <div key={l} className="bg-paper p-4 sm:p-6">
              <dt className="sr-only">{l}</dt>
              <dd className="font-mono text-3xl font-semibold tracking-tight">{n}</dd>
              <dd className="mt-2 text-sm text-muted">{l}</dd>
            </div>
          ))}
        </dl>
      </section>

      {/* Problem to solution */}
      <section className="px-6 py-24">
        <div className="mx-auto grid max-w-6xl items-start gap-12 md:grid-cols-2 md:gap-16">
          <Reveal>
            <Heading
              eyebrow="The problem"
              title="Every tool keeps its own pile. None of them say which parts matter."
              body="Disk analyzers show you big folders. They can't tell you that 4 GB is a model you can download again in a minute, or the only copy of six months of chat history. Cache Vent can, because every verdict comes from the tool's own documentation."
            />
          </Reveal>
          <Reveal delay={120}>
            <div className="rounded-2xl border border-line bg-paper p-2">
              <p className="px-4 pb-2 pt-3 text-sm font-medium text-muted">Found on one developer laptop</p>
              <ul className="divide-y divide-line rounded-xl bg-mist">
                {piles.map((p) => (
                  <li key={p.what} className="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3">
                    <span className="font-mono text-sm font-semibold tabular-nums">{p.bytes}</span>
                    <span className="text-sm">{p.what}</span>
                    <span className="ml-auto">
                      <Chip v={p.verdict} />
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          </Reveal>
        </div>
      </section>

      {/* Benefits */}
      <section id="features" className="bg-mist px-6 py-24">
        <div className="mx-auto max-w-6xl">
          <Reveal>
            <Heading eyebrow="Why Cache Vent" title="Clean up with the facts in front of you" />
          </Reveal>
          <div className="mt-12 grid gap-4 md:grid-cols-2">
            {features.map(({ Icon, title, body }, i) => (
              <Reveal key={title} delay={i * 80}>
                <article className="h-full rounded-2xl border border-line bg-paper p-6 transition-transform sm:p-8 duration-500 ease-vent hover:-translate-y-1">
                  <span className="grid size-12 place-items-center rounded-xl bg-vent-soft text-vent-ink">
                    <Icon size={24} weight="duotone" aria-hidden />
                  </span>
                  <h3 className="mt-6 text-xl font-semibold tracking-tight">{title}</h3>
                  <p className="mt-2 text-base text-muted">{body}</p>
                </article>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      {/* Tagline */}
      <section aria-label="Our promise" className="px-4 py-24 sm:px-6">
        <WordReveal
          className="mx-auto my-16 max-w-[680px] text-4xl font-semibold tracking-tight sm:text-5xl md:text-6xl"
          lines={["Nothing leaves", "your machine.", "Nothing is deleted", "until you say so."]}
        />
      </section>

      {/* How it works */}
      <section id="how" className="px-6 pb-24">
        <div className="mx-auto max-w-6xl">
          <Reveal>
            <Heading eyebrow="How it works" title="Three steps, about a minute" />
          </Reveal>
          <ol className="mt-12 grid gap-4 md:grid-cols-3">
            {steps.map((s, i) => (
              <Reveal as="li" key={s.title} delay={i * 100} className="rounded-2xl border border-line p-6 sm:p-8">
                <span className="font-mono text-sm font-semibold text-vent-ink">0{i + 1}</span>
                <h3 className="mt-4 text-xl font-semibold tracking-tight">{s.title}</h3>
                <p className="mt-2 text-base text-muted">{s.body}</p>
              </Reveal>
            ))}
          </ol>
        </div>
      </section>

      {/* What it finds */}
      <section className="bg-ink px-4 py-24 text-paper sm:px-6">
        <div className="mx-auto grid max-w-6xl gap-12 md:grid-cols-[2fr_3fr]">
          <Reveal>
            <p className="text-sm font-semibold text-vent">What it finds</p>
            <h2 className="mt-3 text-3xl font-semibold tracking-tight md:text-4xl">The usual suspects, named and measured</h2>
            <p className="mt-4 text-lg text-[#b5b5b5]">
              Tools are described in plain YAML files. Adding one takes no code, only the paths and a source for what happens when they go.
            </p>
          </Reveal>
          <Reveal delay={120}>
            <ul className="divide-y divide-[#313131] rounded-2xl border border-[#313131] bg-[#181818]">
              {finds.map((f) => (
                <li key={f.tool + f.what} className="flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-4 sm:px-6">
                  <span className="w-full text-sm font-semibold sm:w-48">{f.tool}</span>
                  <span className="text-sm text-[#b5b5b5]">{f.what}</span>
                  <span className="ml-auto">
                    <Chip v={f.verdict} />
                  </span>
                </li>
              ))}
            </ul>
          </Reveal>
        </div>
      </section>

      {/* Speed */}
      <section id="speed" className="px-6 py-24">
        <div className="mx-auto max-w-6xl">
          <Reveal>
            <Heading
              eyebrow="Speed"
              title="A whole drive in the time it takes to read this line"
              body="Measured on an Intel Core i7 11850H laptop with an NVMe SSD, median of three warm runs. Cache Vent does more work than the baseline: it matches definitions, finds projects and sizes packages."
            />
          </Reveal>
          <Reveal className="mt-12 rounded-2xl border border-line p-4 sm:p-8">
            <ul className="flex flex-col gap-8">
              {speed.map((r) => (
                <li key={r.label}>
                  <div className="flex flex-wrap items-baseline gap-x-3">
                    <span className="text-base font-semibold">{r.label}</span>
                    <span className="text-sm text-muted">{r.detail}</span>
                    <span className="ml-auto font-mono text-xl font-semibold tabular-nums">{r.s.toFixed(2)} s</span>
                  </div>
                  <div className="mt-3 h-3 rounded-full bg-mist">
                    <div className={`bar h-3 rounded-full ${r.ours ? "bg-vent" : "bg-hatch"}`} style={{ width: `${(r.s / 8.68) * 100}%` }} />
                  </div>
                  <p className="mt-2 text-sm text-muted">Peak memory {r.mem}</p>
                </li>
              ))}
            </ul>
            <dl className="mt-10 grid gap-6 border-t border-line pt-8 sm:grid-cols-3">
              {[
                ["2.85 s", "Every drive, 231,033 folders"],
                ["0.38 s", "Home folder, 19,576 folders"],
                ["0.8 s", "From double click to window"],
              ].map(([n, l]) => (
                <div key={l}>
                  <dt className="text-sm text-muted">{l}</dt>
                  <dd className="mt-1 font-mono text-2xl font-semibold tabular-nums">{n}</dd>
                </div>
              ))}
            </dl>
          </Reveal>
        </div>
      </section>

      {/* Safety */}
      <section className="bg-mist px-6 py-24">
        <div className="mx-auto max-w-6xl">
          <Reveal>
            <Heading eyebrow="Safety" title="Built so it can't cost you anything you need" body="These rules are enforced in code and covered by tests, not left to a settings page." />
          </Reveal>
          <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {safety.map(({ Icon, title, body }, i) => (
              <Reveal key={title} delay={(i % 3) * 80} className="rounded-2xl border border-line bg-paper p-6">
                <Icon size={24} weight="duotone" className="text-safe" aria-hidden />
                <h3 className="mt-4 text-base font-semibold">{title}</h3>
                <p className="mt-1 text-sm text-muted">{body}</p>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      {/* FAQ */}
      <section id="faq" className="px-6 py-24">
        <div className="mx-auto grid max-w-6xl gap-12 md:grid-cols-[2fr_3fr]">
          <Reveal>
            <Heading eyebrow="FAQ" title="Questions people ask before running it" />
          </Reveal>
          <Reveal delay={120}>
            <div className="divide-y divide-line rounded-2xl border border-line">
              {faq.map((f) => (
                <details key={f.q} className="group px-4 sm:px-6">
                  <summary className="flex cursor-pointer list-none items-center gap-4 py-6 text-base font-semibold outline-none focus-visible:text-vent-ink [&::-webkit-details-marker]:hidden">
                    <Question size={20} weight="duotone" className="shrink-0 text-muted" aria-hidden />
                    {f.q}
                    <span
                      aria-hidden
                      className="ml-auto grid size-6 shrink-0 place-items-center rounded-full border border-line text-muted transition-transform duration-500 ease-vent group-open:rotate-45"
                    >
                      +
                    </span>
                  </summary>
                  <p className="pb-6 text-base text-muted sm:pl-10">{f.a}</p>
                </details>
              ))}
            </div>
          </Reveal>
        </div>
      </section>

      {/* Download */}
      <section id="download" className="relative overflow-hidden bg-mist px-4 py-24 sm:px-6">
        <Stairs className="pointer-events-none absolute right-0 top-0 hidden w-32 md:block" />
        <div className="relative mx-auto flex max-w-6xl flex-col items-center">
          <Reveal className="flex flex-col items-center text-center">
            <h2 className="max-w-[680px] text-3xl font-semibold tracking-tight sm:text-4xl md:text-5xl">See what is on your disk in under a minute</h2>
            <p className="mt-4 max-w-[680px] text-lg text-muted">
              Free and MIT licensed. Nothing is deleted without your say, and anything moved can be restored for 14 days.
            </p>
          </Reveal>

          <Reveal className="mt-12 w-full max-w-3xl">
            <InstallTabs />
          </Reveal>

          <Reveal className="mt-4 grid w-full max-w-3xl gap-4">
            <details className="group rounded-2xl border border-line bg-paper">
              <summary className="flex cursor-pointer list-none items-center gap-3 px-4 py-4 text-base font-semibold outline-none focus-visible:text-vent-ink sm:px-6 [&::-webkit-details-marker]:hidden">
                All files and checksums
                <span className="text-sm font-normal text-muted">{files.length} builds, version {version}</span>
                <span aria-hidden className="ml-auto grid size-6 shrink-0 place-items-center rounded-full border border-line text-muted transition-transform duration-500 ease-vent group-open:rotate-45">
                  +
                </span>
              </summary>
              <table className="w-full border-t border-line text-sm">
                <caption className="sr-only">All downloads with sizes and SHA256 checksums</caption>
                <thead className="bg-mist text-muted">
                  <tr>
                    <th scope="col" className="px-4 py-3 text-left font-medium sm:px-6">Platform</th>
                    <th scope="col" className="hidden px-6 py-3 text-left font-medium md:table-cell">SHA256</th>
                    <th scope="col" className="px-4 py-3 text-right font-medium sm:px-6">Size</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-line">
                  {files.map((f) => (
                    <tr key={f.id}>
                      <th scope="row" className="px-4 py-4 text-left font-normal sm:px-6">
                        <a
                          href={href(f)}
                          download
                          className="font-semibold underline decoration-line underline-offset-4 transition-colors duration-300 ease-vent hover:decoration-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent"
                        >
                          {f.os} {f.arch}
                        </a>
                        <span className="mt-1 block break-all font-mono text-xs text-muted">{f.file}</span>
                      </th>
                      <td className="hidden px-6 py-4 font-mono text-xs text-muted md:table-cell" title={f.sha256}>
                        {f.sha256.slice(0, 16)}…{f.sha256.slice(-8)}
                      </td>
                      <td className="px-4 py-4 text-right font-mono tabular-nums text-muted sm:px-6">{mb(f.bytes)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <p className="border-t border-line px-4 py-4 text-sm text-muted sm:px-6">
                Full hashes are in{" "}
                <a className="font-medium text-ink underline decoration-line underline-offset-4 hover:decoration-ink" href="/downloads/SHA256SUMS.txt">
                  SHA256SUMS.txt
                </a>
                . The install commands check them for you. On macOS and Linux, a downloaded file needs <code className="font-mono">chmod +x</code> before it runs; the install command is the easier path.
              </p>
            </details>

            <div className="rounded-2xl border border-line bg-paper p-4 sm:p-6">
              <h3 className="text-base font-semibold">Command line</h3>
              <p className="mt-1 text-sm text-muted">Every build is also a CLI, with stable JSON output for scripts.</p>
              <pre className="mt-4 overflow-x-auto rounded-lg bg-ink p-4 font-mono text-xs text-paper">
                <code>{`cachevent scan --all-drives
cachevent clean --dry-run PATH
cachevent restore ID
cachevent scan --json > report.json`}</code>
              </pre>
            </div>
          </Reveal>
        </div>
      </section>
    </>
  );
}
