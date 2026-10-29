import { readFileSync } from "node:fs";
import { join } from "node:path";
import type { Metadata } from "next";
import { ProsePage } from "@/components/prose-page";

export const metadata: Metadata = {
  title: "License and terms",
  description: "Cache Vent is free software under the MIT license.",
  alternates: { canonical: "/license" },
};

// Read at build time, so the page always matches the LICENSE file in the repository.
const license = readFileSync(join(process.cwd(), "..", "LICENSE"), "utf8");

export default function License() {
  return (
    <ProsePage title="License and terms" updated="29 October 2026">
      <p>
        Cache Vent is free and open source. You may use, copy, modify and distribute it, for personal or commercial use, under the MIT
        license below. The software is provided as is, without warranty. Cache Vent moves files into a quarantine rather than deleting
        them, but you remain responsible for what you choose to remove.
      </p>
      <pre className="overflow-x-auto whitespace-pre-wrap rounded-2xl border border-line bg-mist p-6 font-mono text-sm text-ink-2">{license}</pre>
    </ProsePage>
  );
}
