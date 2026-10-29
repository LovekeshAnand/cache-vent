import type { NextConfig } from "next";

// A static export: `next build` writes plain files to out/, which the Cloudflare
// Worker serves as static assets (see wrangler.jsonc).
const nextConfig: NextConfig = {
  output: "export",
  images: { unoptimized: true },
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
