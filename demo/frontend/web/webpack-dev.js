const merge = require('webpack-merge')
const common = require('./webpack-common.js')

module.exports = merge(common, {
  mode: 'development',
  devtool: 'inline-source-map',
  devServer: {
    compress: true,
    disableHostCheck: true, // That solved it,
    historyApiFallback: true,
    contentBase: './dist',
    hot: true
  }
})
