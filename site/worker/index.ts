// Runs only for /downloads/* (see run_worker_first in wrangler.jsonc); every other
// path is served straight from the static assets. It serves the file and adds one to
// that day's count. Nothing about the visitor is stored except the country.

interface Env {
  ASSETS: Fetcher;
  DB: D1Database;
}

const counted = /^\/downloads\/(CacheVent-windows-(x64|arm64)\.exe|cachevent-(macos|linux)-(x64|arm64))$/;

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const res = await env.ASSETS.fetch(request);
    const url = new URL(request.url);
    const m = url.pathname.match(counted);
    // Count full downloads only: not HEAD, not resumed or chunked ranges after the first byte.
    const range = request.headers.get("Range");
    if (m && res.status === 200 && request.method === "GET" && (!range || range.startsWith("bytes=0-"))) {
      const via = ["sh", "ps1"].includes(url.searchParams.get("via") ?? "") ? url.searchParams.get("via")! : "web";
      const country = (request.cf?.country as string | undefined) ?? "XX";
      const day = new Date().toISOString().slice(0, 10);
      ctx.waitUntil(
        env.DB.prepare(
          "INSERT INTO downloads (day, file, via, country, count) VALUES (?, ?, ?, ?, 1) ON CONFLICT (day, file, via, country) DO UPDATE SET count = count + 1",
        )
          .bind(day, m[1], via, country)
          .run()
          .catch(() => {}), // a failed count must never break a download
      );
    }
    return res;
  },
} satisfies ExportedHandler<Env>;
