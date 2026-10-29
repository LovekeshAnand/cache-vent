"use client";

import { AppleLogo, DownloadSimple, LinuxLogo, ShareNetwork, WindowsLogo } from "@phosphor-icons/react";
import { useState } from "react";
import { CopyCommand } from "@/components/copy-command";
import { installCommand, usePlatform } from "@/lib/platform";
import { byId, href, mb, site } from "@/lib/site";

const button =
  "group inline-flex items-center gap-2 rounded-xl bg-ink px-3 py-2 text-base font-semibold text-paper shadow-[0_8px_24px_rgba(10,10,10,0.18)] transition-transform duration-300 ease-vent hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent focus-visible:ring-offset-2";

const more = (
  <a href="#download" className="font-medium text-ink underline decoration-line underline-offset-4 hover:decoration-ink">
    other platforms
  </a>
);

/**
 * The one primary action, matched to the visitor: the Windows app, the install command on
 * macOS and Linux (unsigned builds install cleanly from Terminal), or sharing the link from a phone.
 */
export function DownloadButton({ note = true }: { note?: boolean }) {
  const platform = usePlatform();
  const [shared, setShared] = useState("");

  if (platform === "macos" || platform === "linux") {
    const Icon = platform === "macos" ? AppleLogo : LinuxLogo;
    return (
      <div className="flex w-full max-w-xl flex-col items-center gap-3">
        <p className="inline-flex items-center gap-2 text-base font-semibold">
          <Icon size={20} weight="fill" aria-hidden />
          Install on {platform === "macos" ? "macOS" : "Linux"}: paste this into Terminal
        </p>
        <CopyCommand command={installCommand.unix} label={`Install command for ${platform === "macos" ? "macOS" : "Linux"}`} />
        {note && (
          <p className="text-sm text-muted">
            {platform === "macos" ? "Adds Cache Vent to Applications, no Apple account or admin needed" : "Adds Cache Vent to your applications menu, no sudo"} ·{" "}
            {more}
          </p>
        )}
      </div>
    );
  }

  if (platform === "mobile") {
    async function share() {
      const url = `${site.url}/#download`;
      try {
        if (navigator.share) await navigator.share({ title: "Cache Vent", text: "Install Cache Vent on my computer", url });
        else {
          await navigator.clipboard.writeText(url);
          setShared("Link copied. Open it on your computer.");
        }
      } catch {
        /* the visitor closed the share sheet */
      }
    }
    return (
      <div className="flex flex-col items-center gap-3">
        <button type="button" onClick={share} className={button}>
          <ShareNetwork size={20} weight="bold" aria-hidden />
          Send the link to your computer
        </button>
        <p className="text-sm text-muted" aria-live="polite">
          {shared || "Cache Vent is a desktop app for Windows, macOS and Linux."}
        </p>
      </div>
    );
  }

  const f = byId("windows-x64");
  return (
    <div className="flex flex-col items-center gap-3">
      <a href={href(f)} download className={button}>
        <WindowsLogo size={20} weight="fill" aria-hidden />
        Download for Windows
        <DownloadSimple size={16} weight="bold" aria-hidden className="transition-transform duration-300 ease-vent group-hover:translate-y-0.5" />
      </a>
      {note && (
        <p className="text-sm text-muted">
          Windows 10 and 11 · {mb(f.bytes)} · no installer, no account · {more}
        </p>
      )}
    </div>
  );
}
