import { GRecaptcha } from "./grecaptcha"; // Assuming grecaptcha.d.ts is in the same folder or path is correct

declare global {
  // For properties set on the global object in jest.setup.js for tests
  let mockRecaptchaToken: string | null;
  let mockRecaptchaCallback: ((token: string) => void) | null;
  // For window properties (JSDOM environment)
  interface Window {
    grecaptcha: GRecaptcha; // From your grecaptcha.d.ts
    onloadRecaptchaCallback?: () => void;
  }
}

// Export something to make it a module, if it's not already
export {};
