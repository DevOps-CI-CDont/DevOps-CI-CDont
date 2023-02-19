/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  env: {
    NEXT_PUBLIC_CORS_ORIGIN: "http://0.0.0.0:6001",
    NEXT_PUBLIC_API_URL: "http://localhost:8080"
  },
  output:'standalone'
}

/* module.exports = {
  nextConfig,
  env: {
    NEXT_PUBLIC_CORS_ORIGIN: "http://localhost:6001",
    NEXT_PUBLIC_API_URL: "http://127.0.0.1:8080"
  }
} */

module.exports = nextConfig
