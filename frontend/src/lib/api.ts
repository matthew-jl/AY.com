const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api/v1";

// --- Type Definitions ---

export interface HealthResponse {
  status: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
}

export class ApiError extends Error {
  status: number;
  details?: any;

  constructor(message: string, status: number, details?: any) {
    super(message);
    this.name = "ApiError";
    this.status = status; // HTTP status code
    this.details = details;
  }
}

// --- Helper Functions ---

export function saveTokens(accessToken: string, refreshToken: string): void {
  if (typeof window !== "undefined") {
    localStorage.setItem("accessToken", accessToken);
    localStorage.setItem("refreshToken", refreshToken);
  }
}

export function getAccessToken(): string | null {
  if (typeof window !== "undefined") {
    return localStorage.getItem("accessToken");
  }
  return null;
}

export function clearTokens(): void {
  if (typeof window !== "undefined") {
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
  }
}

// --- Generic Fetch Wrapper ---

/**
 * Generic fetch wrapper with JWT handling and improved error management.
 * @param endpoint API endpoint path (e.g., "/users/health")
 * @param options Standard Fetch API options object
 * @returns Promise resolving to the parsed JSON response body
 * @throws {ApiError} If the API request fails
 */
async function apiFetch<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getAccessToken();
  const defaultHeaders = new Headers({
    "Content-Type": "application/json",
    Accept: "application/json",
  });

  const requestHeaders = new Headers(options.headers);
  defaultHeaders.forEach((value, key) => {
    if (!requestHeaders.has(key)) {
      requestHeaders.set(key, value);
    }
  });

  // Best practicenya add Authorization header cuma kalo token exists AND it's
  // not an auth endpoint. Untuk sekarang backend yang handle
  if (token) {
    requestHeaders.set("Authorization", `Bearer ${token}`);
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers: requestHeaders,
    });
  } catch (networkError) {
    console.error("Network Error:", networkError);
    throw new Error("Network error occurred while fetching data.");
  }

  if (!response.ok) {
    let errorDetails: any = null;
    let errorMessage = `API request failed with status ${response.status}`;

    try {
      errorDetails = await response.json();
      errorMessage =
        errorDetails?.error || errorDetails?.message || errorMessage;
    } catch (e) {
      try {
        const textError = await response.text();
        if (textError) {
          errorMessage = `${errorMessage}: ${textError.substring(0, 100)}`; // Limit length
        }
      } catch (textE) {
        // Ignore if reading text also fails
      }
      console.warn("Could not parse error response body as JSON.", e);
    }

    throw new ApiError(errorMessage, response.status, errorDetails);
  }

  if (
    response.status === 204 ||
    response.headers.get("content-length") === "0"
  ) {
    return null as T;
  }

  try {
    // Parse successful response
    const data = await response.json();
    return data as T;
  } catch (parsingError) {
    console.error("Error parsing successful response:", parsingError);
    throw new Error("Failed to parse successful API response.");
  }
}

// --- API Method Interfaces ---
export interface RegisterRequestData {
  name: string;
  username: string;
  email: string;
  password: string;
  gender: string;
  date_of_birth: string; // YYYY-MM-DD
  security_question: string;
  security_answer: string;
  recaptchaToken: string;
}

export interface LoginRequestData {
  email: string;
  password: string;
  recaptchaToken: string;
}

export interface VerifyEmailRequestData {
  email: string;
  code: string;
}

export interface GetSecurityQuestionRequestData {
  email: string;
}

export interface GetSecurityQuestionResponseData {
  security_question: string;
}

export interface ResetPasswordRequestData {
  email: string;
  security_answer: string;
  new_password: string;
}

// --- API Methods ---
export const api = {
  getHealth: (): Promise<HealthResponse> =>
    apiFetch<HealthResponse>("/users/health", { method: "GET" }),

  login: (credentials: LoginRequestData): Promise<AuthResponse> =>
    apiFetch<AuthResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify(credentials),
    }),

  register: (userData: RegisterRequestData): Promise<void> =>
    apiFetch<void>("/auth/register", {
      method: "POST",
      body: JSON.stringify(userData),
    }),

  verifyEmail: (verificationData: VerifyEmailRequestData): Promise<void> =>
    apiFetch<void>("/auth/verify", {
      method: "POST",
      body: JSON.stringify(verificationData),
    }),

  getSecurityQuestion: (
    data: GetSecurityQuestionRequestData
  ): Promise<GetSecurityQuestionResponseData> =>
    apiFetch<GetSecurityQuestionResponseData>(
      "/auth/forgot-password/question",
      {
        method: "POST",
        body: JSON.stringify(data),
      }
    ),

  resetPassword: (data: ResetPasswordRequestData): Promise<void> =>
    apiFetch<void>("/auth/forgot-password/reset", {
      method: "POST",
      body: JSON.stringify(data),
    }),
};
