import Link from "next/link";

export default function NotFound() {
  return (
    <section className="mx-auto flex max-w-[680px] flex-col items-center px-6 pb-24 pt-24 text-center md:mt-16">
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img src="/logo.png" alt="" width={112} height={56} className="h-16 w-auto" />
      <p className="mt-8 font-mono text-sm font-semibold text-vent-ink">404</p>
      <h1 className="mt-2 text-4xl font-semibold tracking-tight">Nothing in this vent</h1>
      <p className="mt-4 text-lg text-muted">The page you asked for was moved or never existed. Unlike your files, it can&apos;t be restored from quarantine.</p>
      <Link
        href="/"
        className="mt-10 rounded-xl bg-ink px-3 py-2 text-base font-semibold text-paper transition-transform duration-300 ease-vent hover:-translate-y-0.5 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent focus-visible:ring-offset-2"
      >
        Back to the home page
      </Link>
    </section>
  );
}
