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

export interface UserProfileResponse {
  id: number;
  name: string;
  username: string;
  email: string;
  gender: string;
  profile_picture: string | null;
  banner: string | null;
  date_of_birth: string;
  account_status: string;
  account_privacy: string;
  created_at: string; // Expect ISO string from backend JSON
}

export interface MediaMetadata {
  id: number;
  uploader_user_id: number;
  supabase_path: string;
  bucket_name: string;
  mime_type: string;
  file_size: number; // JS uses number for int64
  public_url: string;
  created_at: string; // ISO String
}

export interface ThreadData {
  id: number;
  user_id: number;
  content: string;
  parent_thread_id?: number; // Optional
  reply_restriction: string; // e.g., "EVERYONE", "FOLLOWING"
  scheduled_at?: string | null; // ISO String or null
  posted_at: string; // ISO String
  community_id?: number; // Optional
  is_advertisement: boolean;
  media_ids: number[];
  created_at: string; // ISO String
  // --- Frontend-specific state ---
  author?: UserProfileResponse | null; // Hydrated author info
  media?: MediaMetadata[]; // Hydrated media info
  is_liked?: boolean;
  is_bookmarked?: boolean;
  is_reposted?: boolean; // Add later
  like_count: number;
  reply_count: number;
  repost_count: number;
}

export interface FeedResponse {
  threads: ThreadData[];
  has_more: boolean;
}

export interface ErrorResponse {
  error?: string;
  message?: string;
  [key: string]: unknown; // Allow additional fields
}

export class ApiError extends Error {
  status: number;
  details?: ErrorResponse;

  constructor(message: string, status: number, details?: ErrorResponse) {
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
    throw new Error(
      "Network error occurred while fetching data: " + networkError
    );
  }

  if (!response.ok) {
    let errorDetails: ErrorResponse | null = null;
    let errorMessage = `API request failed with status ${response.status}`;

    try {
      errorDetails = await response.json();
      errorMessage =
        errorDetails?.error || errorDetails?.message || errorMessage;
    } catch {
      try {
        const textError = await response.text();
        if (textError) {
          errorMessage = `${errorMessage}: ${textError.substring(0, 100)}`; // Limit length
        }
      } catch {
        throw new Error("Could not parse error response body as JSON.");
      }
    }

    throw new ApiError(
      errorMessage,
      response.status,
      errorDetails || undefined
    );
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
    throw new Error("Failed to parse successful API response: " + parsingError);
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

export interface CreateThreadRequestData {
  content: string;
  parent_thread_id?: number;
  reply_restriction?: string; // Map frontend choice to backend enum string
  scheduled_at?: string | null; // ISO 8601 format string
  community_id?: number;
  media_ids?: number[];
}

export interface UploadMediaResponseData {
  media: MediaMetadata; // Matches backend
}

export interface GetMediaMetadataRequestData {
  media_id: number;
}

export interface InteractThreadRequestData {
  thread_id: number; // User ID inferred from token on backend
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

  getUserProfile: (): Promise<UserProfileResponse> =>
    apiFetch<UserProfileResponse>("/users/profile", {
      // Uses auth token implicitly
      method: "GET",
    }),

  // Upload: Takes FormData, returns parsed JSON response
  uploadMedia: (formData: FormData): Promise<UploadMediaResponseData> => {
    const token = getAccessToken();
    const headers = new Headers(); // No need to set Content-Type for FormData, browser does it
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }
    // Use raw fetch for FormData upload
    return fetch(`${API_BASE_URL}/media/upload`, {
      method: "POST",
      headers: headers,
      body: formData,
    })
      .then(async (response) => {
        if (!response.ok) {
          let errorDetails: ErrorResponse | null = null;
          let errorMessage = `Media upload failed with status ${response.status}`;
          try {
            errorDetails = await response.json();
            errorMessage =
              errorDetails?.error || errorDetails?.message || errorMessage;
          } catch (e) {
            errorDetails = e as ErrorResponse;
          }
          throw new ApiError(
            errorMessage,
            response.status,
            errorDetails || undefined
          );
        }
        if (
          response.status === 204 ||
          response.headers.get("content-length") === "0"
        ) {
          throw new ApiError(
            "Media upload returned no content",
            response.status
          );
        }
        return response.json();
      })
      .catch((networkError) => {
        throw new Error(
          "Network error occurred during media upload:" + networkError
        );
      });
  },
  getMediaMetadata: (
    req: GetMediaMetadataRequestData
  ): Promise<MediaMetadata> =>
    apiFetch<MediaMetadata>(`/media/${req.media_id}/metadata`, {
      method: "GET",
    }),

  // Thread Methods
  createThread: (
    data: CreateThreadRequestData
  ): Promise<ThreadData> => // Expect backend to return created thread
    apiFetch<ThreadData>("/threads", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  getThread: (
    threadId: number
  ): Promise<ThreadData> => // Expect backend to return hydrated thread
    apiFetch<ThreadData>(`/threads/${threadId}`, { method: "GET" }),
  deleteThread: (threadId: number): Promise<void> =>
    apiFetch<void>(`/threads/${threadId}`, { method: "DELETE" }),
  likeThread: (threadId: number): Promise<void> =>
    apiFetch<void>(`/threads/${threadId}/like`, { method: "POST" }),
  unlikeThread: (threadId: number): Promise<void> =>
    apiFetch<void>(`/threads/${threadId}/like`, { method: "DELETE" }),
  bookmarkThread: (threadId: number): Promise<void> =>
    apiFetch<void>(`/threads/${threadId}/bookmark`, { method: "POST" }),
  unbookmarkThread: (threadId: number): Promise<void> =>
    apiFetch<void>(`/threads/${threadId}/bookmark`, { method: "DELETE" }),

  // Feed Method Placeholder (adjust endpoint/params as needed)
  getFeedThreads: (
    page: number = 1,
    limit: number = 20,
    type: "foryou" | "following" = "foryou"
  ): Promise<FeedResponse> =>
    apiFetch<FeedResponse>(
      `/threads/feed?type=${type}&page=${page}&limit=${limit}`,
      { method: "GET" }
    ),
};
