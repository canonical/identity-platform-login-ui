module.exports = {
  output: 'export',
  distDir: 'dist',
  experimental:  {
    esmExternals: 'loose'
  },
  transpilePackages: ['@canonical/react-components'],
  "images":{
    "unoptimized": true
  },
  assetPrefix: "./"
}
