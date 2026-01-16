const isDev = process.env.DEV === "true";

module.exports = {
    basePath: isDev ? '/ui' : '',
    output: 'export',
    distDir: 'dist',
    experimental: {
        esmExternals: 'loose'
    },
    transpilePackages: ['@canonical/react-components'],
    "images": {
        "unoptimized": true
    },
    assetPrefix: "./",
    async rewrites() {
        return isDev ? [
            {
                source: "/api/:path*",
                destination: "http://localhost:4455/api/:path*",
                basePath: false,
            },
            {
                source: "/self-service/:path*",
                destination: "http://localhost:4455/api/kratos/self-service/:path*",
                basePath: false,
            },
            {
                source: "/ui/:path*",
                destination: "http://localhost:4455/ui/:path*",
                basePath: false,
            },
            {
                source: "/.well-known/webauthn.js",
                destination: "http://localhost:4433/.well-known/ory/webauthn.js",
                basePath: false,
            }
        ] : []
    },
    async redirects() {
        return isDev ? [
            {
                source: "/",
                destination: "/manage_details",
                permanent: false,
            },
        ] : []
    }
}
