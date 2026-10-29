// Builds the Cache Vent binaries the site serves, from the Go module one folder up.
// Writes public/downloads/* and SHA256SUMS.txt, and src/generated/downloads.json
// (file names, sizes and hashes) for the page. Needs Go on PATH.
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { copyFileSync, mkdirSync, readFileSync, rmSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const site = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repo = resolve(site, "..");
const out = join(site, "public", "downloads");

const targets = [
  { id: "windows-x64", os: "Windows", arch: "x64", goos: "windows", goarch: "amd64", file: "CacheVent-windows-x64.exe", app: true },
  { id: "windows-arm64", os: "Windows", arch: "ARM64", goos: "windows", goarch: "arm64", file: "CacheVent-windows-arm64.exe", app: true },
  { id: "macos-arm64", os: "macOS", arch: "Apple silicon", goos: "darwin", goarch: "arm64", file: "cachevent-macos-arm64" },
  { id: "macos-x64", os: "macOS", arch: "Intel", goos: "darwin", goarch: "amd64", file: "cachevent-macos-x64" },
  { id: "linux-x64", os: "Linux", arch: "x64", goos: "linux", goarch: "amd64", file: "cachevent-linux-x64" },
  { id: "linux-arm64", os: "Linux", arch: "ARM64", goos: "linux", goarch: "arm64", file: "cachevent-linux-arm64" },
];

rmSync(out, { recursive: true, force: true });
mkdirSync(out, { recursive: true });

// The release version lives in VERSION at the repo root; CACHEVENT_VERSION overrides it.
const version = process.env.CACHEVENT_VERSION || readFileSync(join(repo, "VERSION"), "utf8").trim();
// The install scripts download from this site, so they need its real address (override for staging).
const base = (process.env.NEXT_PUBLIC_SITE_URL || "https://cachevent.kelados.in").replace(/\/$/, "");
const files = targets.map((t) => {
  // Windows builds are the desktop app: no console, the UI opens on double click.
  const ldflags = (t.app ? "-s -w -H windowsgui -X main.defaultCmd=ui" : "-s -w") + ` -X main.version=${version}`;
  const dest = join(out, t.file);
  execFileSync("go", ["build", "-trimpath", "-ldflags", ldflags, "-o", dest, "./cmd/cachevent"], {
    cwd: repo,
    stdio: "inherit",
    env: { ...process.env, GOOS: t.goos, GOARCH: t.goarch, CGO_ENABLED: "0" },
  });
  const sha256 = createHash("sha256").update(readFileSync(dest)).digest("hex");
  const { id, os, arch, file } = t;
  return { id, os, arch, file, bytes: statSync(dest).size, sha256 };
});

// Icons the installers put next to the app (macOS bundle, Linux launcher).
copyFileSync(join(repo, "assets", "cachevent.icns"), join(out, "cachevent.icns"));
copyFileSync(join(repo, "assets", "cachevent-256.png"), join(out, "cachevent.png"));
for (const f of ["install.sh", "install.ps1"]) {
  const src = readFileSync(join(site, "install", f), "utf8");
  writeFileSync(join(site, "public", f), src.replaceAll("__BASE__", base).replaceAll("__VERSION__", version));
}

writeFileSync(join(out, "SHA256SUMS.txt"), files.map((f) => `${f.sha256}  ${f.file}\n`).join(""));
mkdirSync(join(site, "src", "generated"), { recursive: true });
writeFileSync(join(site, "src", "generated", "downloads.json"), JSON.stringify({ version, files }, null, 2) + "\n");
console.log(files.map((f) => `${f.file}  ${(f.bytes / 1048576).toFixed(1)} MB`).join("\n"));
