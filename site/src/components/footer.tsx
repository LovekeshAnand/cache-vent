import Link from "next/link";
import { site, version } from "@/lib/site";

export function Footer() {
  const link =
    "rounded text-sm text-muted transition-colors duration-300 ease-vent hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent";
  return (
    <footer className="border-t border-line">
      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-6 py-12 md:flex-row md:items-center">
        <div className="flex items-center gap-3">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src="/logo.png" alt="" width={56} height={28} className="h-6 w-auto" />
          <p className="text-sm text-muted">
            <span className="font-semibold text-ink">{site.name}</span> {version} · MIT licensed · made for people who build things
          </p>
        </div>
        <nav aria-label="Footer" className="flex flex-wrap gap-x-6 gap-y-2 md:ml-auto">
          <Link href="/privacy" className={link}>Privacy</Link>
          <Link href="/license" className={link}>License and terms</Link>
          <a href="/downloads/SHA256SUMS.txt" className={link}>Checksums</a>
          {site.github && <a href={site.github} className={link}>Source code</a>}
        </nav>
      </div>
    </footer>
  );
}
