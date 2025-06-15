import "@testing-library/jest-dom";
import { jest } from "@jest/globals";

global.grecaptcha = {
  render: jest.fn((container, params) => {
    if (params.callback) {
      global.mockRecaptchaCallback = params.callback;
    }
    if (params.sitekey === "ERROR_SITE_KEY") {
      if (params["error-callback"]) params["error-callback"]();
    }
    return Math.random();
  }),
  reset: jest.fn(),
  getResponse: jest.fn(
    () => global.mockRecaptchaToken || "mock-recaptcha-token"
  ),
};

global.mockRecaptchaToken = "initial-mock-recaptcha-token";
global.mockRecaptchaCallback = null;

global.onloadRecaptchaCallback = jest.fn();

const localStorageMock = (function () {
  let store = {};
  return {
    getItem(key) {
      return store[key] || null;
    },
    setItem(key, value) {
      store[key] = value.toString();
    },
    removeItem(key) {
      delete store[key];
    },
    clear() {
      store = {};
    },
  };
})();
Object.defineProperty(window, "localStorage", { value: localStorageMock });

jest.mock("svelte-routing", () => ({
  ...jest.requireActual("svelte-routing"),
  navigate: jest.fn(),
  link: jest.fn(),
}));
