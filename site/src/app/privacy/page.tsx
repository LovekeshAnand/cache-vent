import type { Metadata } from "next";
import { ProsePage } from "@/components/prose-page";
import { author, site } from "@/lib/site";

export const metadata: Metadata = {
  title: "Privacy",
  description: "The Cache Vent app collects nothing. The website counts downloads without cookies and uses analytics only with your consent.",
  alternates: { canonical: "/privacy" },
};

export default function Privacy() {
  return (
    <ProsePage title="Privacy" updated="29 October 2026">
      <p>
        Short version: the app collects nothing. This website counts downloads without cookies, and uses Google Analytics and Microsoft Clarity only if
        you allow them.
      </p>
      <h2>The app</h2>
      <p>
        Cache Vent runs entirely on your computer. It has no account, no telemetry, no crash reporting and no update check. It reads folder
        names and sizes to build its report, and keeps that report, a short size history and its quarantine in its own folder on your
        machine. None of it is sent anywhere.
      </p>
      <p>
        The desktop window talks to a server inside the app that only listens on 127.0.0.1 and requires a token that changes on every
        launch, so other programs and websites cannot read it. The only time the app opens a web address is when you choose to follow a
        documentation link next to an item.
      </p>
      <h2>Download counts</h2>
      <p>
        When a file under /downloads is fetched, from this page or by the install commands, the site adds one to a daily total for that file. It
        records the date, the file, whether it came from the website or an install script, and the country Cloudflare reports for the request.
        It does not store IP addresses, browser details or cookies, and the totals cannot identify you.
      </p>
      <h2>Analytics, only with your consent</h2>
      <p>
        If you choose Allow in the banner, this site loads Google Analytics and Microsoft Clarity. Google Analytics measures visits and which buttons
        are used. Clarity records how pages are used, such as clicks and scrolling, with page text masked, so we can fix confusing parts. Both set
        cookies and are run by Google and Microsoft under their own privacy policies. If you choose No thanks, or never choose, neither loads. You can
        change your mind any time with Cookie settings at the bottom of every page, which also clears their cookies.
      </p>
      <h2>Hosting</h2>
      <p>
        The site is served by Cloudflare, which processes standard request data such as IP addresses to deliver pages and protect against abuse, as
        described in its own privacy policy.
      </p>
      <h2>Contact</h2>
      <p>
        Questions about privacy go to <a className="underline underline-offset-4" href={`mailto:${author.email}`}>{author.email}</a>, or open an
        issue on <a className="underline underline-offset-4" href={`${site.github}/issues`}>GitHub</a>.
      </p>
    </ProsePage>
  );
}
