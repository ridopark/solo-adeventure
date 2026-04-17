import type { NextConfig } from "next";

const isStaticExport = process.env.STATIC_EXPORT === "1";

const config: NextConfig = {
  reactStrictMode: true,
  output: isStaticExport ? "export" : "standalone",
  images: { unoptimized: true },
};

export default config;
