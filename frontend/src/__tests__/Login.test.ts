import "@testing-library/jest-dom";
import { render, fireEvent, screen, waitFor } from "@testing-library/svelte";
import Login from "../routes/Login.svelte";
import * as apiModule from "../lib/api";
import { navigate } from "svelte-routing";
import { setUser, clearUser } from "../stores/userStore";
import { setAuthState } from "../stores/authStore";
import type { UserProfileBasic } from "../lib/api";

jest.mock("../lib/api", () => {
  const originalModule = jest.requireActual("../lib/api");
  return {
    __esModule: true,
    ...originalModule,
    api: {
      login: jest.fn(),
      getOwnUserProfile: jest.fn(),
    },
    saveTokens: jest.fn(),
    clearTokens: jest.fn(),
  };
});

jest.mock("../stores/userStore", () => ({
  setUser: jest.fn() as jest.MockedFunction<
    (userData: UserProfileBasic | null) => void
  >,
  clearUser: jest.fn() as jest.MockedFunction<() => void>,
  user: { subscribe: jest.fn(() => () => {}) },
}));

jest.mock("../stores/authStore", () => ({
  setAuthState: jest.fn() as jest.MockedFunction<(isAuth: boolean) => void>,
  isAuthenticated: { subscribe: jest.fn(() => () => {}) },
}));

const provideRecaptchaToken = (tokenValue: string) => {
  mockRecaptchaToken = tokenValue;
  if (mockRecaptchaCallback) {
    mockRecaptchaCallback(tokenValue);
  }
};

describe("Login.svelte", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockRecaptchaToken = "mock-token-for-test";
    mockRecaptchaCallback = null;
    window.localStorage.clear();
  });

  test("1. Renders the login form correctly", () => {
    render(Login);
    expect(
      screen.getByRole("heading", { name: /Sign in to AY/i })
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/Email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Password/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Log in/i })).toBeInTheDocument();
    expect(
      document.getElementById("recaptcha-container-login")
    ).toBeInTheDocument();
  });

  test("2. Shows error if fields are empty on submit", async () => {
    render(Login);
    const loginButton = screen.getByRole("button", { name: /Log in/i });
    await fireEvent.click(loginButton);
    expect(
      await screen.findByText(/Please enter both email and password./i)
    ).toBeInTheDocument();
  });

  test("3. Shows error if reCAPTCHA is not completed", async () => {
    render(Login);
    (window.grecaptcha.render as jest.Mock).mockImplementationOnce(
      (container, params) => {
        mockRecaptchaCallback = params.callback;
        return 1;
      }
    );
    (window.grecaptcha.getResponse as jest.Mock).mockReturnValueOnce("");

    await fireEvent.input(screen.getByLabelText(/Email/i), {
      target: { value: "test@example.com" },
    });
    await fireEvent.input(screen.getByLabelText(/Password/i), {
      target: { value: "password123" },
    });
    await fireEvent.click(screen.getByRole("button", { name: /Log in/i }));
    expect(
      await screen.findByText(/Please complete the reCAPTCHA verification./i)
    ).toBeInTheDocument();
  });

  test("4. Calls api.login on successful validation and reCAPTCHA", async () => {
    render(Login);
    provideRecaptchaToken("mock-test-recaptcha-token");

    await fireEvent.input(screen.getByLabelText(/Email/i), {
      target: { value: "test@example.com" },
    });
    await fireEvent.input(screen.getByLabelText(/Password/i), {
      target: { value: "password123" },
    });
    await fireEvent.click(screen.getByRole("button", { name: /Log in/i }));

    expect(apiModule.api.login).toHaveBeenCalledTimes(1);
    expect(apiModule.api.login).toHaveBeenCalledWith({
      email: "test@example.com",
      password: "password123",
      recaptchaToken: "mock-test-recaptcha-token",
    });
  });

  test("5. Handles successful login, saves tokens, fetches profile, sets auth state, and navigates", async () => {
    const mockLoginResponse = {
      access_token: "mock-access-token",
      refresh_token: "mock-refresh-token",
    };
    const mockUserProfile = {
      user: {
        id: 1,
        name: "Test User",
        username: "testuser",
        email: "test@example.com",
        gender: "",
        profile_picture: null,
        banner: null,
        date_of_birth: "",
        account_status: "",
        account_privacy: "",
        created_at: "",
        subscribed_to_newsletter: false,
        bio: "",
      } as UserProfileBasic,
    };

    (apiModule.api.login as jest.Mock).mockResolvedValue(mockLoginResponse);
    (apiModule.api.getOwnUserProfile as jest.Mock).mockResolvedValue(
      mockUserProfile
    );

    render(Login);
    provideRecaptchaToken("test-token");

    await fireEvent.input(screen.getByLabelText(/Email/i), {
      target: { value: "test@example.com" },
    });
    await fireEvent.input(screen.getByLabelText(/Password/i), {
      target: { value: "password123" },
    });
    await fireEvent.click(screen.getByRole("button", { name: /Log in/i }));

    await waitFor(() => {
      expect(apiModule.api.login).toHaveBeenCalledTimes(1);
      expect(apiModule.saveTokens).toHaveBeenCalledWith(
        mockLoginResponse.access_token,
        mockLoginResponse.refresh_token
      );
      expect(apiModule.api.getOwnUserProfile).toHaveBeenCalledTimes(1);
      expect(setUser).toHaveBeenCalledWith(mockUserProfile.user);
      expect(setAuthState).toHaveBeenCalledWith(true);
      expect(navigate).toHaveBeenCalledWith("/home", { replace: true });
    });
  });

  test("6. Handles API error on login and displays error message", async () => {
    const errorMessage = "Invalid credentials from API";
    (apiModule.api.login as jest.Mock).mockRejectedValue(
      new apiModule.ApiError(errorMessage, 401)
    );

    render(Login);
    provideRecaptchaToken("test-token");

    await fireEvent.input(screen.getByLabelText(/Email/i), {
      target: { value: "wrong@example.com" },
    });
    await fireEvent.input(screen.getByLabelText(/Password/i), {
      target: { value: "wrongpassword" },
    });
    await fireEvent.click(screen.getByRole("button", { name: /Log in/i }));

    expect(
      await screen.findByText(`Login failed: ${errorMessage}`)
    ).toBeInTheDocument();
    expect(apiModule.clearTokens).toHaveBeenCalled();
    expect(clearUser).toHaveBeenCalled();
    expect(setAuthState).toHaveBeenCalledWith(false);
  });
});
