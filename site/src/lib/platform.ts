"use client";

import { useSyncExternalStore } from "react";
import { site } from "@/lib/site";

export type Platform = "windows" | "macos" | "linux" | "mobile";

function detect(): Platform {
  const ua = navigator.userAgent.toLowerCase();
  // iPadOS reports itself as a Mac; touch points give it away.
  if (/iphone|ipad|ipod|android/.test(ua) || (ua.includes("mac") && navigator.maxTouchPoints > 1)) return "mobile";
  if (ua.includes("mac")) return "macos";
  if (ua.includes("linux") || ua.includes("x11")) return "linux";
  return "windows";
}

const noop = () => () => {};

/** The visitor's platform. Server render and first paint assume Windows, the most common. */
export function usePlatform(): Platform {
  return useSyncExternalStore(noop, detect, () => "windows" as Platform);
}

export const installCommand = {
  windows: `irm ${site.url}/install.ps1 | iex`,
  unix: `curl -fsSL ${site.url}/install.sh | sh`,
};
