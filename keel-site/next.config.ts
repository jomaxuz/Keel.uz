import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // Development only. In production Caddy routes `/api/*` to the control
  // plane before Next ever sees it — because this rewrite is baked into the
  // build, so setting CONTROL_ORIGIN in the container has no effect at all.
  // That mistake shipped once: every console login reached localhost inside
  // the site container and came back as "Internal Server Error".
  async rewrites() {
    const control = process.env.CONTROL_ORIGIN ?? "http://localhost:9000";
    return [{ source: "/api/:path*", destination: `${control}/api/:path*` }];
  },
};

export default nextConfig;
