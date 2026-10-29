# Cache Vent website

The landing page and download host for Cache Vent: Next.js (static export), TypeScript and Tailwind, served by a Cloudflare Worker as static assets. No server code runs.

## Commands

```sh
npm install
npm run dev          # local development
npm run downloads    # build all six binaries, checksums, icons and the install scripts (needs Go)
npm run build        # downloads + static site in out/
npm run preview      # build, then serve out/ with wrangler exactly as Cloudflare will
npm run deploy       # build, then wrangler deploy
```

## Before deploying

The site lives at https://cachevent.kelados.in (the default). To build for another address:

```sh
NEXT_PUBLIC_SITE_URL=https://cachevent.example npm run deploy
```

Optionally set `NEXT_PUBLIC_GITHUB_URL` to show source links. Run `npx wrangler login` once first.

## Download counts

A small Worker (`worker/index.ts`) runs only for `/downloads/*`. It serves the file and adds one to a daily total in the `cache-vent` D1 database: file, source (`sh`/`ps1` install command or `web` link), and country. No IP addresses, user agents or cookies. HEAD requests and resumed downloads aren't counted.

```sh
npm run stats          # last 30 days: by file, source, country and day
npm run stats -- 7     # last 7 days
```

Raw SQL: `npx wrangler d1 execute cache-vent --remote --command "SELECT * FROM downloads ORDER BY day DESC"`.

## Google Analytics and Microsoft Clarity

Both are opt-in: nothing loads until a visitor chooses Allow in the consent banner, and Cookie settings in the footer withdraws it. Set the IDs at build time; with neither set, there's no banner and no scripts.

```sh
NEXT_PUBLIC_GA_ID=G-XXXXXXXXXX NEXT_PUBLIC_CLARITY_ID=abcdefghij npm run deploy
```

Events sent after consent: `download` (with the file) and `copy_install_command`.

## What is served

| Path | |
|---|---|
| `/downloads/*` | Windows app (x64, ARM64), macOS and Linux binaries (Apple silicon, Intel, x64, ARM64), `SHA256SUMS.txt`, icons |
| `/install.sh` | macOS and Linux installer, from `install/install.sh` with the site address filled in |
| `/install.ps1` | Windows installer, from `install/install.ps1` |

Each file is under Cloudflare's 25 MiB static asset limit. Security headers are in `public/_headers`.

After deploying, run the **Platform check** workflow with the live address to install and test it on real Macs and Linux.
