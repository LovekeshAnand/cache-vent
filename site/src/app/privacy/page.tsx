import type { Metadata } from "next";
import { ProsePage } from "@/components/prose-page";

export const metadata: Metadata = {
  title: "Privacy",
  description: "Cache Vent collects nothing. The app runs offline and this website sets no cookies.",
  alternates: { canonical: "/privacy" },
};

export default function Privacy() {
  return (
    <ProsePage title="Privacy" updated="29 October 2026">
      <p>Short version: Cache Vent collects nothing, and this website sets no cookies.</p>
      <h2>The app</h2>
      <p>
        Cache Vent runs entirely on your computer. It has no account, no telemetry, no crash reporting and no update check. It reads folder
        names and sizes to build its report, and keeps that report, a short size history and its quarantine in its own folder on your
        machine. None of it is sent anywhere.
      </p>
      <p>
        The desktop window talks to a server inside the app that only listens on 127.0.0.1 and requires a token that changes on every
        launch, so other programs and websites cannot read it.
      </p>
      <p>
        The only time the app opens a web address is when you choose to follow a documentation link next to an item. That link opens in
        your browser.
      </p>
      <h2>This website</h2>
      <p>
        This site is a set of static files served by Cloudflare. It uses no analytics, no cookies and no third party scripts. Cloudflare
        processes standard request data such as IP addresses to deliver the pages and protect against abuse, as described in its own
        privacy policy.
      </p>
      <h2>Contact</h2>
      <p>Questions about privacy can be raised as an issue on the project&apos;s source repository.</p>
    </ProsePage>
  );
}
