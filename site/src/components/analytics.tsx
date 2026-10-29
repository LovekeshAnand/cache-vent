"use client";

import { useEffect } from "react";

declare global {
  interface Window {
    gtag?: (...args: unknown[]) => void;
    clarity?: (...args: unknown[]) => void;
  }
}

/** track sends an event to Google Analytics and Microsoft Clarity. */
export function track(event: string, params: Record<string, string> = {}) {
  window.gtag?.("event", event, params);
  window.clarity?.("event", event);
}

/** DownloadTracking reports clicks on any /downloads/ link as a `download` event with the file name. */
export function DownloadTracking() {
  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      const a = (e.target as Element).closest?.("a[href^='/downloads/']");
      if (a) track("download", { file: a.getAttribute("href")!.split("/").pop()!.split("?")[0] });
    };
    document.addEventListener("click", onClick);
    return () => document.removeEventListener("click", onClick);
  }, []);
  return null;
}
