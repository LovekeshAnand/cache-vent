"use client";

import { AppleLogo, LinuxLogo, WindowsLogo } from "@phosphor-icons/react";
import { useState, type KeyboardEvent } from "react";
import { CopyCommand } from "@/components/copy-command";
import { installCommand, usePlatform, type Platform } from "@/lib/platform";
import { byId, href } from "@/lib/site";

type Tab = Exclude<Platform, "mobile">;

const tabs: { id: Tab; label: string; Icon: typeof WindowsLogo }[] = [
  { id: "windows", label: "Windows", Icon: WindowsLogo },
  { id: "macos", label: "macOS", Icon: AppleLogo },
  { id: "linux", label: "Linux", Icon: LinuxLogo },
];

const link = "font-medium text-ink underline decoration-line underline-offset-4 hover:decoration-ink";

function Body({ tab }: { tab: Tab }) {
  if (tab === "windows")
    return (
      <>
        <CopyCommand command={installCommand.windows} label="Install command for Windows PowerShell" />
        <ol className="mt-4 flex list-decimal flex-col gap-2 pl-6 text-sm text-muted">
          <li>Open PowerShell from the Start menu and paste the command.</li>
          <li>It installs for your user only, adds Cache Vent to the Start menu and opens it. No admin rights needed.</li>
          <li>
            Prefer a file? <a className={link} href={href(byId("windows-x64"))} download>Download the app</a> and run it. The build isn&apos;t code
            signed yet, so SmartScreen asks first: choose <strong>More info</strong>, then <strong>Run anyway</strong>.
          </li>
        </ol>
      </>
    );
  if (tab === "macos")
    return (
      <>
        <CopyCommand command={installCommand.unix} label="Install command for macOS Terminal" />
        <ol className="mt-4 flex list-decimal flex-col gap-2 pl-6 text-sm text-muted">
          <li>Open Terminal (Spotlight, then type Terminal) and paste the command.</li>
          <li>
            It puts <strong>Cache Vent.app</strong> in Applications, signs it on your Mac and clears the download flag, so it opens like any app. No Apple
            account, no admin password.
          </li>
          <li>Open Cache Vent from Launchpad or Spotlight. The first scan may ask to read Documents, Desktop or Downloads; allow it to include them.</li>
          <li>
            The interface opens in Chrome, Edge or Brave if one is installed, otherwise in your default browser. macOS 13 Ventura or later, Apple silicon and Intel.
          </li>
        </ol>
      </>
    );
  return (
    <>
      <CopyCommand command={installCommand.unix} label="Install command for Linux terminal" />
      <ol className="mt-4 flex list-decimal flex-col gap-2 pl-6 text-sm text-muted">
        <li>Paste the command into a terminal. It installs to ~/.local/bin and adds Cache Vent to your applications menu. No sudo.</li>
        <li>Start it from the menu or run <code className="font-mono">cachevent ui</code>. It opens in Chrome, Edge, Chromium or Brave, or your default browser.</li>
        <li>x64 and ARM64. To remove it, run the same command with <code className="font-mono">sh -s -- --uninstall</code> at the end.</li>
      </ol>
    </>
  );
}

/** Install instructions per OS, opened on the visitor's own. Arrow keys move between tabs. */
export function InstallTabs() {
  const platform = usePlatform();
  const [picked, setPicked] = useState<Tab | null>(null);
  const tab: Tab = picked ?? (platform === "mobile" ? "windows" : platform);

  function onKey(e: KeyboardEvent) {
    const i = tabs.findIndex((t) => t.id === tab);
    const next = e.key === "ArrowRight" ? i + 1 : e.key === "ArrowLeft" ? i - 1 : null;
    if (next === null) return;
    const t = tabs[(next + tabs.length) % tabs.length];
    setPicked(t.id);
    document.getElementById(`tab-${t.id}`)?.focus();
  }

  return (
    <div className="rounded-2xl border border-line bg-paper p-2">
      <div role="tablist" aria-label="Install on" onKeyDown={onKey} className="grid grid-cols-3 gap-1 rounded-xl bg-mist p-1">
        {tabs.map(({ id, label, Icon }) => (
          <button
            key={id}
            id={`tab-${id}`}
            role="tab"
            type="button"
            aria-selected={tab === id}
            aria-controls="install-panel"
            tabIndex={tab === id ? 0 : -1}
            onClick={() => setPicked(id)}
            className="inline-flex items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-semibold text-muted transition-colors duration-300 ease-vent hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent aria-selected:bg-paper aria-selected:text-ink aria-selected:shadow-[0_1px_2px_rgba(10,10,10,0.08)]"
          >
            <Icon size={16} weight="fill" aria-hidden />
            {label}
          </button>
        ))}
      </div>
      <div id="install-panel" role="tabpanel" aria-labelledby={`tab-${tab}`} className="p-4 sm:p-6">
        <Body tab={tab} />
      </div>
    </div>
  );
}
