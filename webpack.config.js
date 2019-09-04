module.exports = {
  context: __dirname + "/client",
  entry: "./index.js",
  mode: 'development',

  output: {
    filename: "index.js",
    path: __dirname + "/dist",
  },

  devtool: 'inline-source-map',

  module: {
    rules: [
      {
        test: /\.m?js$/,
        exclude: /(node_modules|bower_components)/,
        use: {
          loader: 'babel-loader',
          options: {
            presets: [
              '@babel/preset-env',
              '@babel/preset-react',
              {
                "plugins": [
                  "@babel/plugin-proposal-class-properties",
                ]
              },
            ]
          }
        }
      }
    ]
  }
}
