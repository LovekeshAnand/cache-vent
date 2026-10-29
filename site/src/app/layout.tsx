import type { Metadata, Viewport } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Analytics } from "@/components/analytics";
import { Footer } from "@/components/footer";
import { Nav } from "@/components/nav";
import { author, site } from "@/lib/site";
import "./globals.css";

const geistSans = Geist({ variable: "--font-geist-sans", subsets: ["latin"] });
const geistMono = Geist_Mono({ variable: "--font-geist-mono", subsets: ["latin"] });

export const metadata: Metadata = {
  metadataBase: new URL(site.url),
  title: { default: "Cache Vent · Free disk space your dev tools took", template: "%s · Cache Vent" },
  description: site.description,
  applicationName: site.name,
  authors: [{ name: author.name, url: author.github }],
  creator: author.name,
  keywords: ["disk cleanup", "node_modules", "python venv", "ollama models", "hugging face cache", "developer tools", "disk space"],
  alternates: { canonical: "/" },
  openGraph: {
    type: "website",
    siteName: site.name,
    title: "Cache Vent · Free disk space your dev tools took",
    description: site.description,
    url: "/",
  },
  twitter: { card: "summary_large_image", title: "Cache Vent", description: site.description, creator: author.xHandle },
};

export const viewport: Viewport = { themeColor: "#ffffff", colorScheme: "light" };

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" suppressHydrationWarning className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
      <body className="flex min-h-dvh flex-col">
        <script dangerouslySetInnerHTML={{ __html: "document.documentElement.classList.add('js')" }} />
        <a
          href="#main"
          className="fixed left-4 top-4 z-[60] -translate-y-24 rounded-lg bg-ink px-3 py-2 text-sm font-semibold text-paper focus:translate-y-0"
        >
          Skip to content
        </a>
        <Nav />
        <main id="main" className="flex-1">
          {children}
        </main>
        <Footer />
        <Analytics />
      </body>
    </html>
  );
}
