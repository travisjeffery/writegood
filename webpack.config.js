module.exports = {
  context: __dirname + "/client",
  entry: "./app.js",
  mode: 'development',

  output: {
    filename: "app.js",
    path: __dirname + "/dist",
  },

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
                  "@babel/plugin-proposal-class-properties"
                ]
              },
            ]
          }
        }
      }
    ]
  }
}
