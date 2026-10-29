// Download report from the live counter: npm run stats [days]   (add --local for wrangler dev)
import { execFileSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const wrangler = join(dirname(fileURLToPath(import.meta.url)), "..", "node_modules", "wrangler", "bin", "wrangler.js");

const args = process.argv.slice(2);
const where = args.includes("--local") ? "--local" : "--remote";
const days = Number(args.find((a) => /^\d+$/.test(a)) ?? 30);
const since = new Date(Date.now() - days * 864e5).toISOString().slice(0, 10);

function query(sql) {
  // Run wrangler's script with node directly: no shell, so the SQL stays one argument on Windows too.
  const out = execFileSync(process.execPath, [wrangler, "d1", "execute", "cache-vent", where, "--json", "--command", sql], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "inherit"],
  });
  return JSON.parse(out)[0].results;
}

const total = query("SELECT COALESCE(SUM(count), 0) AS n FROM downloads")[0].n;
console.log(`\nAll time: ${total} downloads\n\nLast ${days} days, since ${since}:`);
const q = (by) => `SELECT ${by}, SUM(count) AS downloads FROM downloads WHERE day >= '${since}' GROUP BY ${by} ORDER BY downloads DESC`;
console.log("\nBy file");
console.table(query(q("file")));
console.log("By source (sh = macOS/Linux install command, ps1 = Windows install command, web = download link)");
console.table(query(q("via")));
console.log("By country");
console.table(query(q("country") + " LIMIT 15"));
console.log("By day");
console.table(query(`SELECT day, SUM(count) AS downloads FROM downloads WHERE day >= '${since}' GROUP BY day ORDER BY day`));
