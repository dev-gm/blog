const path = require('path');
const HtmlBundlerPlugin = require('html-bundler-webpack-plugin');

module.exports = {
  mode: 'production',
  output: {
    path: path.join(__dirname, 'dist'),
  },
  plugins: [
    new HtmlBundlerPlugin({
      entry: {
        index: './static/index.html',
      },
      css: {
        inline: true, // inline CSS into HTML
      },
      js: {
        inline: true, // inline JS into HTML
      },
    }),
  ],
  module: {
    rules: [
      {
        test: /\.css$/,
        use: ['css-loader'],
      },
      // inline all assets: images, svg, fonts
      {
        test: /\.(png|jpe?g|webp|svg|woff2?)$/i,
        type: 'asset/inline',
      },
    ],
  },
  performance: false, // disable warning max size
};
