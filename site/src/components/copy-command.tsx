"use client";

import { Check, Copy } from "@phosphor-icons/react";
import { useRef, useState } from "react";

/** A one-line terminal command with a copy button. Scrolls sideways on narrow screens. */
export function CopyCommand({ command, label }: { command: string; label: string }) {
  const [copied, setCopied] = useState(false);
  const code = useRef<HTMLElement>(null);

  async function copy() {
    try {
      await navigator.clipboard.writeText(command);
    } catch {
      // Clipboard blocked: select the text so Ctrl+C or Cmd+C works.
      const r = document.createRange();
      if (code.current) r.selectNodeContents(code.current);
      getSelection()?.removeAllRanges();
      getSelection()?.addRange(r);
      return;
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="flex w-full items-center gap-2 rounded-xl bg-ink p-2 text-paper">
      <span aria-hidden className="pl-2 font-mono text-sm text-[#9b9b9b]">
        $
      </span>
      <code ref={code} aria-label={label} className="min-w-0 flex-1 overflow-x-auto whitespace-nowrap py-2 font-mono text-sm [scrollbar-width:none]">
        {command}
      </code>
      <button
        type="button"
        onClick={copy}
        className="inline-flex shrink-0 items-center gap-2 rounded-lg bg-[#272727] px-3 py-2 text-sm font-semibold transition-colors duration-300 ease-vent hover:bg-[#313131] active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vent"
      >
        {copied ? <Check size={16} weight="bold" aria-hidden /> : <Copy size={16} weight="bold" aria-hidden />}
        <span aria-live="polite">{copied ? "Copied" : "Copy"}</span>
      </button>
    </div>
  );
}
