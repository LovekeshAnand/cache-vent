"use client";

import Link from "next/link";
import Script from "next/script";
import { useEffect, useSyncExternalStore } from "react";

// Set at build time. With neither set, no banner shows and nothing loads.
const GA_ID = process.env.NEXT_PUBLIC_GA_ID ?? "";
const CLARITY_ID = process.env.NEXT_PUBLIC_CLARITY_ID ?? "";
const enabled = Boolean(GA_ID || CLARITY_ID);

type Consent = "granted" | "denied" | null;
const KEY = "analytics-consent";

declare global {
  interface Window {
    gtag?: (...args: unknown[]) => void;
    clarity?: (...args: unknown[]) => void;
  }
}

function read(): Consent {
  try {
    const v = localStorage.getItem(KEY);
    return v === "granted" || v === "denied" ? v : null;
  } catch {
    return "denied"; // storage blocked: never ask, never load
  }
}

function subscribe(onChange: () => void) {
  window.addEventListener("storage", onChange);
  window.addEventListener(KEY, onChange);
  return () => {
    window.removeEventListener("storage", onChange);
    window.removeEventListener(KEY, onChange);
  };
}

function save(v: Consent) {
  try {
    if (v) localStorage.setItem(KEY, v);
    else localStorage.removeItem(KEY);
  } catch {}
  window.dispatchEvent(new Event(KEY));
}

/** track sends an event to whichever analytics loaded (only after consent). */
export function track(event: string, params: Record<string, string> = {}) {
  window.gtag?.("event", event, params);
  window.clarity?.("event", event);
}

/**
 * Analytics loads Google Analytics and Microsoft Clarity only after the visitor allows
 * it, and tracks download clicks. Downloads are also counted on the server, without
 * cookies, for everyone (see worker/index.ts).
 */
export function Analytics() {
  // Server render: undecided, so the banner markup waits for the client.
  const consent = useSyncExternalStore(subscribe, read, () => "denied" as Consent);

  useEffect(() => {
    if (consent !== "granted") return;
    const onClick = (e: MouseEvent) => {
      const a = (e.target as Element).closest?.("a[href^='/downloads/']");
      if (a) track("download", { file: a.getAttribute("href")!.split("/").pop()!.split("?")[0] });
    };
    document.addEventListener("click", onClick);
    return () => document.removeEventListener("click", onClick);
  }, [consent]);

  if (!enabled) return null;

  return (
    <>
      {consent === "granted" && GA_ID && (
        <>
          <Script src={`https://www.googletagmanager.com/gtag/js?id=${GA_ID}`} strategy="afterInteractive" />
          <Script id="ga" strategy="afterInteractive">
            {`window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments)}window.gtag=gtag;gtag('js',new Date());gtag('config','${GA_ID}');`}
          </Script>
        </>
      )}
      {consent === "granted" && CLARITY_ID && (
        <Script id="clarity" strategy="afterInteractive">
          {`(function(c,l,a,r,i,t,y){c[a]=c[a]||function(){(c[a].q=c[a].q||[]).push(arguments)};t=l.createElement(r);t.async=1;t.src="https://www.clarity.ms/tag/"+i;y=l.getElementsByTagName(r)[0];y.parentNode.insertBefore(t,y)})(window,document,"clarity","script","${CLARITY_ID}");`}
        </Script>
      )}
      {consent === null && (
        <section
          aria-label="Analytics consent"
          className="fixed inset-x-4 bottom-4 z-50 mx-auto max-w-md rounded-2xl border border-line bg-paper p-4 shadow-[0_16px_48px_rgba(10,10,10,0.16)] sm:inset-x-auto sm:left-6 sm:p-6 motion-safe:animate-[menu-in_500ms_var(--ease-vent)_both]"
        >
          <h2 className="text-base font-semibold">Can we measure visits?</h2>
          <p className="mt-2 text-sm text-muted">
            Google Analytics and Microsoft Clarity show us which parts of this page help. They set cookies, and Clarity records how the page is used with
            text masked. Nothing loads unless you allow it. The app itself never sends anything.{" "}
            <Link href="/privacy" className="font-medium text-ink underline decoration-line underline-offset-4 hover:decoration-ink">
              Privacy
            </Link>
          </p>
          <div className="mt-4 flex gap-2">
            <button
              type="button"
              onClick={() => save("granted")}
              className="flex-1 rounded-lg bg-ink px-3 py-2 text-sm font-semibold text-paper transition-transform duration-300 ease-vent active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent focus-visible:ring-offset-2"
            >
              Allow
            </button>
            <button
              type="button"
              onClick={() => save("denied")}
              className="flex-1 rounded-lg border border-line px-3 py-2 text-sm font-semibold transition-colors duration-300 ease-vent hover:bg-mist active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent"
            >
              No thanks
            </button>
          </div>
        </section>
      )}
    </>
  );
}

/** Reopens the choice. Withdrawing consent clears the analytics cookies and reloads. */
export function ConsentSettings({ className }: { className: string }) {
  if (!enabled) return null;
  function reset() {
    const was = read();
    save(null);
    if (was === "granted") {
      for (const c of document.cookie.split(";")) {
        const name = c.split("=")[0].trim();
        if (/^(_ga|_gid|_clck|_clsk|CLID|MUID)/.test(name)) {
          for (const domain of ["", location.hostname, "." + location.hostname.split(".").slice(-2).join(".")]) {
            document.cookie = `${name}=; Max-Age=0; path=/${domain ? "; domain=" + domain : ""}`;
          }
        }
      }
      location.reload();
    }
  }
  return (
    <button type="button" onClick={reset} className={className}>
      Cookie settings
    </button>
  );
}
