"use client";

import { GithubLogo } from "@phosphor-icons/react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { site } from "@/lib/site";

const links = [
  { href: "/#features", label: "Features" },
  { href: "/#how", label: "How it works" },
  { href: "/#speed", label: "Speed" },
  { href: "/#faq", label: "FAQ" },
];

export function Nav() {
  const [open, setOpen] = useState(false);
  useEffect(() => {
    document.body.style.overflow = open ? "hidden" : "";
    const esc = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    window.addEventListener("keydown", esc);
    return () => window.removeEventListener("keydown", esc);
  }, [open]);

  return (
    <header className="fixed inset-x-0 top-4 z-50 flex justify-center px-4">
      <nav
        aria-label="Main"
        className="flex w-full max-w-3xl items-center gap-2 rounded-2xl border border-line/80 bg-paper/75 p-2 shadow-[0_8px_32px_rgba(10,10,10,0.06)] backdrop-blur-xl md:w-auto"
      >
        <Link
          href="/"
          aria-label="Cache Vent home"
          className="flex items-center gap-2 rounded-lg px-2 py-1 text-base font-semibold tracking-tight outline-none focus-visible:ring-2 focus-visible:ring-vent"
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src="/logo.png" alt="" width={56} height={28} className="h-6 w-auto" />
          {site.name}
        </Link>
        <ul className="ml-auto hidden items-center md:flex">
          {links.map((l) => (
            <li key={l.href}>
              <Link
                href={l.href}
                className="rounded-lg px-3 py-2 text-sm font-medium text-muted transition-colors duration-300 ease-vent hover:bg-mist hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent"
              >
                {l.label}
              </Link>
            </li>
          ))}
        </ul>
        <a
          href={site.github}
          target="_blank"
          rel="noopener noreferrer"
          aria-label="Cache Vent on GitHub"
          title="Source code on GitHub"
          className="ml-auto grid size-10 place-items-center rounded-lg text-ink transition-colors duration-300 ease-vent hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent md:ml-1"
        >
          <GithubLogo size={20} weight="fill" aria-hidden />
        </a>
        <Link
          href="/#download"
          className="rounded-lg bg-ink px-3 py-2 text-sm font-semibold text-paper transition-transform duration-300 ease-vent hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent focus-visible:ring-offset-2"
        >
          Download
        </Link>
        <button
          type="button"
          aria-label={open ? "Close menu" : "Open menu"}
          aria-expanded={open}
          aria-controls="mobile-menu"
          onClick={() => setOpen(!open)}
          className="relative grid size-10 place-items-center rounded-lg hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent md:hidden"
        >
          <span
            className={`absolute h-0.5 w-4 rounded-full bg-ink transition-transform duration-500 ease-vent ${open ? "rotate-45" : "-translate-y-1"}`}
          />
          <span
            className={`absolute h-0.5 w-4 rounded-full bg-ink transition-transform duration-500 ease-vent ${open ? "-rotate-45" : "translate-y-1"}`}
          />
        </button>
      </nav>

      <div
        id="mobile-menu"
        hidden={!open}
        className="fixed inset-0 -z-10 bg-paper/95 px-6 pt-24 backdrop-blur-2xl md:hidden"
      >
        <ul className="flex flex-col gap-2">
          {[...links, { href: "/#download", label: "Download" }].map((l, i) => (
            <li
              key={l.href}
              className="motion-safe:animate-[menu-in_500ms_var(--ease-vent)_both]"
              style={{ animationDelay: `${Math.min(i * 60, 400)}ms` }}
            >
              <Link
                href={l.href}
                onClick={() => setOpen(false)}
                className="block rounded-xl px-3 py-3 text-3xl font-semibold tracking-tight focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent"
              >
                {l.label}
              </Link>
            </li>
          ))}
        </ul>
        <a
          href={site.github}
          className="mt-8 inline-flex items-center gap-2 rounded-xl px-3 py-3 text-base font-semibold text-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent"
        >
          <GithubLogo size={20} weight="fill" aria-hidden />
          Source code on GitHub
        </a>
      </div>
    </header>
  );
}
