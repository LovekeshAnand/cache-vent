"use client";

import { useEffect, useRef, useState, type ElementType, type ReactNode } from "react";

/** Reveal rises and fades its content in once, when it first enters the viewport. */
export function Reveal({
  as: Tag = "div",
  className = "",
  children,
  delay = 0,
}: {
  as?: ElementType;
  className?: string;
  children: ReactNode;
  delay?: number;
}) {
  const ref = useRef<HTMLElement>(null);
  const [shown, setShown] = useState(false);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const io = new IntersectionObserver(
      ([e]) => {
        if (e.isIntersecting) {
          setShown(true);
          io.disconnect();
        }
      },
      { rootMargin: "0px 0px -10% 0px" },
    );
    io.observe(el);
    return () => io.disconnect();
  }, []);
  return (
    <Tag ref={ref} data-shown={shown} className={`reveal ${className}`} style={delay ? { transitionDelay: `${delay}ms` } : undefined}>
      {children}
    </Tag>
  );
}

/**
 * WordReveal lights each word from muted to full ink as it crosses a line 40%
 * above the bottom of the viewport, in reading order. Words stay lit.
 */
export function WordReveal({ lines, className = "" }: { lines: string[]; className?: string }) {
  const ref = useRef<HTMLParagraphElement>(null);
  useEffect(() => {
    const words = ref.current?.querySelectorAll<HTMLElement>(".word");
    if (!words) return;
    const io = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) {
            (e.target as HTMLElement).dataset.on = "true";
            io.unobserve(e.target);
          }
        }
      },
      { rootMargin: "0px 0px -40% 0px" },
    );
    words.forEach((w) => io.observe(w));
    return () => io.disconnect();
  }, []);
  return (
    <p ref={ref} className={className}>
      {lines.map((line, i) => (
        <span key={i} className="block">
          {line.split(" ").map((w, j) => (
            <span key={j} className="word">
              {w}{" "}
            </span>
          ))}
        </span>
      ))}
    </p>
  );
}
