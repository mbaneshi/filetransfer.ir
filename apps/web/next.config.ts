import type { NextConfig } from "next";

// Same-origin /api so the HttpOnly ft_session cookie works without CORS.
const API = process.env.API_INTERNAL_URL ?? "http://localhost:18474";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${API}/api/:path*` }];
  },
};

export default nextConfig;
