import downloads from "@/generated/downloads.json";

export const site = {
  name: "Cache Vent",
  // Set NEXT_PUBLIC_SITE_URL at build time to the deployed domain (used for canonical and social cards).
  url: process.env.NEXT_PUBLIC_SITE_URL ?? "https://cachevent.kelados.in",
  // Set NEXT_PUBLIC_GITHUB_URL to show source links; they are hidden while it is empty.
  github: process.env.NEXT_PUBLIC_GITHUB_URL ?? "",
  description:
    "Cache Vent finds what AI and developer tools keep on your disk, explains each folder, and moves only what is safe into a quarantine you can undo. Free, open source, offline.",
};

export type Download = (typeof downloads.files)[number];
export const version = downloads.version;
export const files: Download[] = downloads.files;
export const byId = (id: string) => files.find((f) => f.id === id)!;
export const href = (f: Download) => `/downloads/${f.file}`;
export const mb = (bytes: number) => (bytes / 1048576).toFixed(1) + " MB";
