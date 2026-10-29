import { EnvelopeSimple, GithubLogo, LinkedinLogo, XLogo } from "@phosphor-icons/react/dist/ssr";
import Link from "next/link";
import { ConsentSettings } from "@/components/analytics";
import { author, site, version } from "@/lib/site";

const social = [
  { href: author.github, label: "GitHub", Icon: GithubLogo },
  { href: author.x, label: "X", Icon: XLogo },
  { href: author.linkedin, label: "LinkedIn", Icon: LinkedinLogo },
  { href: `mailto:${author.email}`, label: "Email", Icon: EnvelopeSimple },
];

export function Footer() {
  const link =
    "rounded text-sm text-muted transition-colors duration-300 ease-vent hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent";
  return (
    <footer className="border-t border-line">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-12 sm:px-6 md:grid-cols-[3fr_2fr_2fr]">
        <div>
          <Link href="/" className="inline-flex items-center gap-2 rounded-lg text-base font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src="/logo.png" alt="" width={56} height={28} className="h-6 w-auto" />
            {site.name}
          </Link>
          <p className="mt-3 max-w-sm text-sm text-muted">
            See what your dev tools keep on disk and free what is safe, with an undo. Free and open source under the MIT license.
          </p>
          <p className="mt-6 text-sm text-muted">
            Built by{" "}
            <a href={author.github} className="font-medium text-ink underline decoration-line underline-offset-4 hover:decoration-ink">
              {author.name}
            </a>
          </p>
          <ul className="mt-3 flex gap-2" aria-label={`${author.name} elsewhere`}>
            {social.map(({ href, label, Icon }) => (
              <li key={label}>
                <a
                  href={href}
                  aria-label={label}
                  title={label}
                  {...(href.startsWith("http") ? { target: "_blank", rel: "noopener noreferrer me" } : {})}
                  className="grid size-10 place-items-center rounded-xl border border-line text-muted transition-colors duration-300 ease-vent hover:border-ink hover:text-ink active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent"
                >
                  <Icon size={20} weight="fill" aria-hidden />
                </a>
              </li>
            ))}
          </ul>
        </div>

        <nav aria-label="Project">
          <h2 className="text-sm font-semibold">Project</h2>
          <ul className="mt-3 flex flex-col gap-2">
            <li><a href={site.github} className={link}>Source code</a></li>
            <li><a href={`${site.github}/issues`} className={link}>Report a problem</a></li>
            <li><a href={`${site.github}/releases`} className={link}>Releases</a></li>
            <li><a href="/downloads/SHA256SUMS.txt" className={link}>Checksums</a></li>
          </ul>
        </nav>

        <nav aria-label="Legal and contact">
          <h2 className="text-sm font-semibold">About</h2>
          <ul className="mt-3 flex flex-col gap-2">
            <li><Link href="/privacy" className={link}>Privacy</Link></li>
            <li><Link href="/license" className={link}>License and terms</Link></li>
            <li><a href={`${site.github}/security/policy`} className={link}>Security</a></li>
            <li><ConsentSettings className={link} /></li>
            <li><a href={`mailto:${author.email}`} className={link}>{author.email}</a></li>
          </ul>
        </nav>
      </div>
      <div className="border-t border-line">
        <p className="mx-auto max-w-6xl px-4 py-6 text-xs text-muted sm:px-6">
          © 2026 {author.name} · {site.name} {version} · Analytics only if you allow it
        </p>
      </div>
    </footer>
  );
}
