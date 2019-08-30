module.exports = {
  publicPath: process.env.PUBLIC_PATH || '/',
  outputDir: process.env.OUTPUT_DIR || 'dist',
  devServer: {
    proxy: {
      '^/api': {
        target: 'http://127.0.0.1:8081'
      }
    }
  }
}
