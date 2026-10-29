-- Daily download counts. No IP addresses, user agents or cookies: only what was
-- downloaded, how (install script or website) and the country Cloudflare reports.
CREATE TABLE downloads (
  day     TEXT    NOT NULL, -- YYYY-MM-DD, UTC
  file    TEXT    NOT NULL, -- e.g. cachevent-macos-arm64
  via     TEXT    NOT NULL, -- sh, ps1 or web
  country TEXT    NOT NULL, -- two-letter code, or XX
  count   INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (day, file, via, country)
);
