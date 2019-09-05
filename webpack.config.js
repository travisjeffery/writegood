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
        test: /\.(js|jsx)$/,
        exclude: /node_modules/,
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
      },
      {
        test: /\.css$/,
        use: ['style-loader', 'css-loader']
      }
    ]
  }
}
