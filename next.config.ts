import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "uploads.mangadex.org",
        pathname: "/**",
      },
      {
        protocol: "https",
        hostname: "lh3.googleusercontent.com",
        pathname: "/**",
      },
      {
        protocol: "http",
        hostname: "25.17.216.66",
        port: "8080",
        pathname: "/**",
      },
      {
        protocol: "http",
        hostname: "localhost",
        port: "8080",
        pathname: "/**",
      },
      {
        protocol: "http",
        hostname: "127.0.0.1",
        port: "8080",
        pathname: "/**",
      },
      {
        protocol: "http",
        hostname: "25.19.136.155",
        port: "8080",
        pathname: "/**",
      },
      {
        protocol: "http",
        hostname: "10.236.15.222",
        port: "8080",
        pathname: "/**",
      },
      {
        protocol: "http",
        hostname: "172.20.10.9",
        port: "8080",
        pathname: "/**",
      },
    ],
    // Disable image optimization to bypass private IP restrictions
    // This allows images from private IPs to load without errors
    unoptimized: true,
  },
};

export default nextConfig;
