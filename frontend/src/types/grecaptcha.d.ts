interface GRecaptcha {
  render(
    container: string | HTMLElement,
    parameters: {
      sitekey: string;
      theme?: "light" | "dark";
      size?: "normal" | "compact";
      tabindex?: number;
      callback?: (token: string) => void; // Called on success
      "expired-callback"?: () => void; // Called on expiration
      "error-callback"?: () => void; // Called on network error/config issue
    }
  ): number; // Returns a widget ID
  reset(widgetId?: number): void;
  getResponse(widgetId?: number): string; // Gets the token programmatically
}

// Declare the global variable that Google's script adds
declare global {
  interface Window {
    grecaptcha: GRecaptcha;
    // Callback function for async script loading
    onloadRecaptchaCallback?: () => void;
  }
}

export {};
