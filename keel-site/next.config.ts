import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // The dashboard talks to the control plane; in production Caddy serves both
  // under keel.uz so the browser stays single-origin.
  async rewrites() {
    const control = process.env.CONTROL_ORIGIN ?? "http://localhost:9000";
    return [{ source: "/api/:path*", destination: `${control}/api/:path*` }];
  },
};

export default nextConfig;
