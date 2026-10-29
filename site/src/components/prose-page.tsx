import Link from "next/link";
import type { ReactNode } from "react";

/** A plain reading page (privacy, license) with a way back home. */
export function ProsePage({ title, updated, children }: { title: string; updated: string; children: ReactNode }) {
  return (
    <article className="mx-auto max-w-[680px] px-6 pb-24 pt-24 md:mt-16">
      <Link href="/" className="text-sm font-medium text-muted hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent">
        ← Back to Cache Vent
      </Link>
      <h1 className="mt-6 text-4xl font-semibold tracking-tight">{title}</h1>
      <p className="mt-2 text-sm text-muted">Last updated {updated}</p>
      <div className="mt-10 flex flex-col gap-6 text-base text-ink-2 [&_h2]:mt-4 [&_h2]:text-xl [&_h2]:font-semibold [&_h2]:text-ink">{children}</div>
    </article>
  );
}
