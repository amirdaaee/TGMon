import type { NextConfig } from "next";

const minio = (process.env.NEXT_PUBLIC_MINIO_URL ?? "").replace(/\/$/, "");

const nextConfig: NextConfig = {
  async rewrites() {
    if (!minio) {
      return [];
    }
    return [{ source: "/minio/:path*", destination: `${minio}/:path*` }];
  },
};

export default nextConfig;
