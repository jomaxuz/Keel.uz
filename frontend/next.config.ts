import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // StrictMode double-mounts effects in dev, which makes 2GIS MapGL create and
  // destroy two WebGL contexts back-to-back → "Failed to obtain WebGL context".
  // Disabled so the map initialises exactly once.
  reactStrictMode: false,
  output: "standalone",
  // Allow Next dev assets to be requested from tunnel origins (localhost.run,
  // cloudflared) when previewing the site over HTTPS.
  allowedDevOrigins: ["*.lhr.life", "*.trycloudflare.com", "*.ngrok-free.app"],
  images: {
    // Menu images are served from the Go backend `/uploads/*` route.
    remotePatterns: [
      { protocol: "http", hostname: "localhost" },
      { protocol: "http", hostname: "127.0.0.1" },
    ],
  },
  // Proxy API + uploads to the Go backend so the browser only ever talks to the
  // Next origin. This means a single tunnel (cloudflared/ngrok) exposes the whole
  // app over HTTPS with no mixed-content or CORS issues.
  async rewrites() {
    const backend = process.env.BACKEND_ORIGIN ?? "http://localhost:8080";
    return [
      { source: "/api/:path*", destination: `${backend}/api/:path*` },
      { source: "/uploads/:path*", destination: `${backend}/uploads/:path*` },
    ];
  },
};

export default nextConfig;
