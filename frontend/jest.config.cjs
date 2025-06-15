module.exports = {
  transform: {
    "^.+\\.svelte$": [
      "svelte-jester",
      {
        preprocess: true,
      },
    ],
    "^.+\\.ts$": "babel-jest",
    "^.+\\.(js|mjs)$": "babel-jest",
  },
  moduleFileExtensions: ["js", "mjs", "ts", "svelte"],
  setupFilesAfterEnv: ["<rootDir>/jest.setup.js"],
  testEnvironment: "jest-environment-jsdom",
  moduleNameMapper: {
    "\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$":
      "<rootDir>/__mocks__/fileMock.js",
    "\\.(css|scss)$": "<rootDir>/__mocks__/styleMock.js",
    "^../lib/(.*)$": "<rootDir>/src/lib/$1",
    "^../stores/(.*)$": "<rootDir>/src/stores/$1",
  },
  transformIgnorePatterns: [
    "/node_modules/(?!svelte-routing|other-esm-svelte-lib).+\\.(js|mjs|svelte)$",
  ],
};
