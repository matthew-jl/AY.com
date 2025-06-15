module.exports = {
  presets: [
    [
      "@babel/preset-env",
      {
        targets: {
          node: "current", // Or your desired browser targets
        },
      },
    ],
    // Add '@babel/preset-typescript' if you're using 'babel-jest' for TS and not 'ts-jest'
    // If you use ts-jest, you might not need babel-jest for TS files.
    // For simplicity with the above jest.config, let's assume babel-jest handles TS too.
    "@babel/preset-typescript",
  ],
};
