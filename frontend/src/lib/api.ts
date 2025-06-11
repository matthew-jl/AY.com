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

export interface UserProfileBasic {
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
  created_at: string;
  subscribed_to_newsletter: boolean;
  bio: string;
}

export interface UserProfileResponseData {
  user: UserProfileBasic | null;
  follower_count: number;
  following_count: number;
  is_followed_by_requester: boolean;
  is_blocked_by_requester: boolean;
  is_blocking_requester: boolean;
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
  author?: UserProfileBasic | null; // Hydrated author info
  media?: MediaMetadata[]; // Hydrated media info
  is_liked_by_current_user?: boolean;
  is_bookmarked_by_current_user?: boolean;
  is_reposted_by_current_user?: boolean; // Add later
  like_count: number;
  reply_count: number;
  repost_count: number;
  bookmark_count: number;
}

export interface FeedResponse {
  threads: ThreadData[];
  has_more: boolean;
}

export interface SearchUsersApiResponse {
  users: UserProfileBasic[];
  has_more: boolean;
}

export interface SearchThreadsApiResponse {
  threads: ThreadData[];
  has_more: boolean;
}

export interface GetTrendingHashtagsApiResponse {
  trending_hashtags: TrendingHashtagItem[];
}

export interface TrendingHashtagItem {
  tag: string;
  count: number;
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

export function getRefreshToken(): string | null {
  if (typeof window !== "undefined") {
    return localStorage.getItem("refreshToken");
  }
  return null;
}

export async function refreshAccessToken(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;
  try {
    const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!response.ok) return false;
    const data = await response.json();
    saveTokens(data.access_token, data.refresh_token);
    return true;
  } catch {
    return false;
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
  options: RequestInit = {},
  retry = true
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

  // If unauthorized, try refresh
  if (response.status === 401 && retry) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      // Try again with new token
      return apiFetch<T>(endpoint, options, false);
    } else {
      clearTokens();
      throw new ApiError("Session expired. Please log in again.", 401);
    }
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
  subscribed_to_newsletter: boolean;
  profile_picture_url?: string | null;
  banner_url?: string | null;
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

export interface UpdateUserProfileRequestData {
  name?: string | null;
  bio?: string | null;
  current_password?: string | null;
  new_password?: string | null;
  gender?: string | null;
  profile_picture_url?: string | null;
  banner_url?: string | null;
  date_of_birth?: string | null;
  account_privacy?: "public" | "private" | null;
  subscribed_to_newsletter?: boolean | null;
}

export interface ResendVerificationRequestData {
  email: string;
}

export interface CreateThreadRequestData {
  content: string;
  parent_thread_id?: number | null;
  reply_restriction?: string;
  scheduled_at?: string | null;
  community_id?: number | null;
  media_ids?: number[];
  categories?: string[];
}

export interface UploadMediaResponseData {
  media: MediaMetadata;
}

export interface GetMediaMetadataRequestData {
  media_id: number;
}

export interface InteractThreadRequestData {
  thread_id: number;
}

export interface SocialUserListItem {
  user_summary: UserProfileBasic;
  is_followed_by_requester: boolean;
}

export interface SocialListResponseData {
  users: SocialUserListItem[];
  has_more: boolean;
}

export interface NotificationData {
  id: number;
  user_id: number;
  type: string; // "new_follower", "like", "mention", "reply" etc.
  message: string;
  is_read: boolean;
  entity_id: string; // ID of related entity (thread_id, follower_user_id)
  actor_id?: number | null;
  // Optional hydrated actor details
  actor_name?: string;
  actor_username?: string;
  actor_profile_picture?: string | null;
  created_at: string;
}

export interface GetNotificationsApiResponse {
  notifications: NotificationData[];
  has_more: boolean;
}

export interface GetUnreadNotificationCountApiResponse {
  count: number;
}

export interface GetWhoToFollowApiResponse {
  users: UserProfileBasic[];
}

export interface UserSummary {
  id: number;
  name: string;
  username: string;
  profile_picture_url: string | null;
}

export interface MessageData {
  id: number;
  chat_id: number;
  sender_id: number;
  content: string;
  type: string; // "text", "image", "video", "gif"
  media_items?: MediaMetadata[];
  sent_at: string;
  is_deleted: boolean;
  sender_summary?: UserSummary | null;
}

export interface ChatData {
  id: number;
  type: string; // "direct", "group"
  name?: string | null;
  creator_id: number;
  created_at: string;
  updated_at: string;
  participants: UserSummary[];
  last_message?: MessageData | null;
  // Frontend specific
  display_name?: string | null;
  display_avatar?: string | null;
  unread_count?: number;
}

export interface GetOrCreateDirectChatRequestData {
  other_user_id: number;
}

export interface SendMessageRequestData {
  content: string;
  media_ids?: number[];
}

export interface GetMessagesApiResponse {
  messages: MessageData[];
  has_more: boolean;
}

export interface GetUserChatsApiResponse {
  chats: ChatData[];
  has_more: boolean;
}

export interface CreateGroupChatRequestData {
  name: string;
  initial_participant_ids: number[];
}

export interface AddParticipantRequestData {
  target_user_id: number;
}

export const CommunityStatusNumbers = {
  COMMUNITY_STATUS_UNSPECIFIED: 0,
  PENDING_APPROVAL: 1,
  ACTIVE: 2,
  REJECTED: 3,
  BANNED: 4,
} as const;
// Type for the keys of CommunityStatusNumbers, e.g., "ACTIVE", "PENDING_APPROVAL"
export type CommunityStatusString = keyof typeof CommunityStatusNumbers;
// Type for the values (the numbers)
export type CommunityStatusCode =
  (typeof CommunityStatusNumbers)[CommunityStatusString];

export interface CommunityDetailsData {
  id: number;
  name: string;
  description: string;
  creator_id: number;
  creator_summary: UserSummary; // Hydrated creator info
  icon_url: string | null;
  banner_url: string | null;
  categories: string[];
  rules: string[];
  status: CommunityStatusCode;
  created_at: string;
  member_count: number;
}

export interface CommunityFullDetailsResponseData {
  community: CommunityDetailsData;
  is_joined_by_requester: boolean;
  requester_role: string; // "member", "moderator", "owner", "pending_join", "none"
  has_pending_request_by_requester: boolean;
}

export interface CreateCommunityRequestData {
  name: string;
  description: string;
  // creator_id is set by backend from JWT
  icon_url?: string | null;
  banner_url?: string | null;
  categories?: string[];
  rules?: string[];
}

export interface CommunityListItem {
  id: number;
  name: string;
  description_snippet: string;
  icon_url: string | null;
  status: CommunityStatusCode;
  member_count: number;
  is_joined_by_requester?: boolean;
  has_pending_request_by_requester?: boolean;
  categories?: string[];
}

export interface ListCommunitiesApiResponse {
  communities: CommunityListItem[];
  has_more: boolean;
}

export interface JoinRequestItem {
  request_id: number;
  community_id: number;
  community_name?: string; // Needs hydration for user's list
  user: UserSummary; // User who requested
  status: string;
  requested_at: string;
}

export interface GetUserJoinRequestsApiResponse {
  requests: JoinRequestItem[];
  has_more: boolean;
}

export interface HandleJoinRequestPayload {
  target_user_id: number;
}

export interface HandleCommunityJoinRequestPayload {
  target_user_id: number;
}

export interface UpdateMemberRoleRequestData {
  target_user_id: number;
  new_role: "member" | "moderator";
}

export interface CommunityMemberDetails {
  user: UserSummary;
  role: string; // "member", "moderator", "owner"
  joined_at: string;
}

export interface GetCommunityMembersApiResponse {
  members: CommunityMemberDetails[];
  has_more: boolean;
}

export interface GetTopCommunityMembersApiResponse {
  users: UserProfileBasic[];
}

export interface GetCommunityThreadsRequestData {
  community_id: number;
  requester_user_id?: number | null;
  sort_type?: "latest" | "top_posts";
  thread_type_filter?: "all" | "media_only";
  page?: number;
  limit?: number;
}

export interface AISuggestionRequest {
  text: string;
}

export interface AISuggestionResponse {
  predicted_class_index: number; // 0, 1, 2, 3
  predicted_category_name: string; // "World", "Sports"
  original_text_snippet?: string;
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

  resendVerificationCode: (
    data: ResendVerificationRequestData
  ): Promise<void> =>
    apiFetch<void>("/auth/verify/resend", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  getOwnUserProfile: (): Promise<UserProfileResponseData> =>
    apiFetch<UserProfileResponseData>("/users/me/profile", { method: "GET" }),

  getUserProfileByUsername: (
    username: string
  ): Promise<UserProfileResponseData> =>
    apiFetch<UserProfileResponseData>(`/profiles/${username}`, {
      method: "GET",
    }),
  updateUserProfile: (
    data: UpdateUserProfileRequestData
  ): Promise<UserProfileBasic> =>
    apiFetch<UserProfileBasic>("/users/me/profile", {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  followUser: (username: string): Promise<void> =>
    apiFetch<void>(`/profiles/${username}/follow`, { method: "POST" }),
  unfollowUser: (username: string): Promise<void> =>
    apiFetch<void>(`/profiles/${username}/follow`, { method: "DELETE" }),
  blockUser: (username: string): Promise<void> =>
    apiFetch<void>(`/profiles/${username}/block`, { method: "POST" }),
  unblockUser: (username: string): Promise<void> =>
    apiFetch<void>(`/profiles/${username}/block`, { method: "DELETE" }),

  getFollowers: (
    username: string,
    page: number = 1,
    limit: number = 20
  ): Promise<SocialListResponseData> =>
    apiFetch<SocialListResponseData>(
      `/profiles/${username}/followers?page=${page}&limit=${limit}`,
      { method: "GET" }
    ),
  getFollowing: (
    username: string,
    page: number = 1,
    limit: number = 20
  ): Promise<SocialListResponseData> =>
    apiFetch<SocialListResponseData>(
      `/profiles/${username}/following?page=${page}&limit=${limit}`,
      { method: "GET" }
    ),

  // Upload: Takes FormData, returns parsed JSON response
  uploadMedia: (formData: FormData): Promise<UploadMediaResponseData> => {
    const token = getAccessToken();
    const headers = new Headers();
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }
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

  getUserThreads: (
    username: string,
    type: "posts" | "replies" | "likes" | "media",
    page: number = 1,
    limit: number = 10
  ): Promise<FeedResponse> =>
    apiFetch<FeedResponse>(
      `/profiles/${username}/threads?type=${type}&page=${page}&limit=${limit}`,
      { method: "GET" }
    ),

  searchUsers: (
    query: string,
    page: number = 1,
    limit: number = 10
  ): Promise<SearchUsersApiResponse> =>
    apiFetch<SearchUsersApiResponse>(
      `/search/users?q=${encodeURIComponent(
        query
      )}&page=${page}&limit=${limit}`,
      { method: "GET" }
    ),

  searchThreads: (
    query: string,
    page: number = 1,
    limit: number = 10
  ): Promise<SearchThreadsApiResponse> =>
    apiFetch<SearchThreadsApiResponse>(
      `/search/threads?q=${encodeURIComponent(
        query
      )}&page=${page}&limit=${limit}`,
      { method: "GET" }
    ),

  getTrendingHashtags: (
    limit: number = 10
  ): Promise<GetTrendingHashtagsApiResponse> =>
    apiFetch<GetTrendingHashtagsApiResponse>(
      `/trending/hashtags?limit=${limit}`,
      { method: "GET" }
    ),

  getWhoToFollow: (limit: number = 3): Promise<GetWhoToFollowApiResponse> =>
    apiFetch<GetWhoToFollowApiResponse>(
      `/suggestions/who-to-follow?limit=${limit}`,
      { method: "GET" }
    ),

  getBookmarkedThreads: (
    page: number = 1,
    limit: number = 20
  ): Promise<FeedResponse> =>
    apiFetch<FeedResponse>(`/threads/bookmarked?page=${page}&limit=${limit}`, {
      method: "GET",
    }),

  getNotifications: (
    page: number = 1,
    limit: number = 20,
    unreadOnly: boolean = false
  ): Promise<GetNotificationsApiResponse> =>
    apiFetch<GetNotificationsApiResponse>(
      `/notifications?page=${page}&limit=${limit}&unread_only=${unreadOnly}`,
      { method: "GET" }
    ),

  markNotificationAsRead: (notificationId: number): Promise<void> =>
    apiFetch<void>(`/notifications/read/${notificationId}`, { method: "POST" }), // Assuming POST, as it's an action

  markAllNotificationsAsRead: (): Promise<void> =>
    apiFetch<void>("/notifications/read/all", { method: "POST" }),

  getUnreadNotificationCount:
    (): Promise<GetUnreadNotificationCountApiResponse> =>
      apiFetch<GetUnreadNotificationCountApiResponse>(
        "/notifications/unread_count",
        { method: "GET" }
      ),

  getUserChats: (
    page: number = 1,
    limit: number = 20
  ): Promise<GetUserChatsApiResponse> =>
    apiFetch<GetUserChatsApiResponse>(`/messages?page=${page}&limit=${limit}`, {
      method: "GET",
    }),

  deleteChat: (chatId: number): Promise<void> =>
    apiFetch<void>(`/messages/chat/${chatId}`, { method: "DELETE" }),

  getOrCreateDirectChat: (
    data: GetOrCreateDirectChatRequestData
  ): Promise<ChatData> =>
    apiFetch<ChatData>("/messages/direct", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  getMessagesForChat: (
    chatId: number,
    page: number = 1,
    limit: number = 30
  ): Promise<GetMessagesApiResponse> =>
    apiFetch<GetMessagesApiResponse>(
      `/messages/chat/${chatId}?page=${page}&limit=${limit}`,
      { method: "GET" }
    ),

  sendMessageToChat: (
    chatId: number,
    data: SendMessageRequestData
  ): Promise<MessageData> =>
    apiFetch<MessageData>(`/messages/chat/${chatId}`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  deleteMessage: (chatId: number, messageId: number): Promise<void> =>
    apiFetch<void>(`/messages/chat/${chatId}/message/${messageId}`, {
      method: "DELETE",
    }),

  createGroupChat: (data: CreateGroupChatRequestData): Promise<ChatData> =>
    apiFetch<ChatData>("/messages/group", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  addParticipantToGroup: (
    chatId: number,
    data: AddParticipantRequestData
  ): Promise<void> =>
    apiFetch<void>(`/messages/group/${chatId}/participants`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  removeParticipantFromGroup: (
    chatId: number,
    targetUserId: number
  ): Promise<void> =>
    apiFetch<void>(`/messages/group/${chatId}/participants/${targetUserId}`, {
      method: "DELETE",
    }),

  createCommunity: (
    data: CreateCommunityRequestData
  ): Promise<CommunityDetailsData> =>
    apiFetch<CommunityDetailsData>("/communities", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  getCommunityDetails: (
    communityId: number
  ): Promise<CommunityFullDetailsResponseData> =>
    apiFetch<CommunityFullDetailsResponseData>(`/communities/${communityId}`, {
      method: "GET",
    }),

  listCommunities: (
    filterType: "ALL_PUBLIC" | "JOINED_BY_USER" | "CREATED_BY_USER",
    userIdContext?: number | null, // Required for JOINED or CREATED
    page: number = 1,
    limit: number = 20,
    searchQuery?: string,
    categories?: string[]
  ): Promise<ListCommunitiesApiResponse> => {
    let queryParams = `filter_type=${filterType}&page=${page}&limit=${limit}`;
    if (userIdContext) queryParams += `&user_id_context=${userIdContext}`;
    if (searchQuery)
      queryParams += `&search_query=${encodeURIComponent(searchQuery)}`;
    if (categories && categories.length > 0)
      queryParams += `&categories=${categories.join(",")}`;
    return apiFetch<ListCommunitiesApiResponse>(`/communities?${queryParams}`, {
      method: "GET",
    });
  },

  getJoinedCommunities: (
    userId: number,
    page: number = 1,
    limit: number = 100
  ): Promise<ListCommunitiesApiResponse> =>
    api.listCommunities("JOINED_BY_USER", userId, page, limit),

  requestToJoinCommunity: (communityId: number): Promise<void> =>
    apiFetch<void>(`/communities/${communityId}/join`, { method: "POST" }),

  getUserJoinRequests: (
    page: number = 1,
    limit: number = 20
  ): Promise<GetUserJoinRequestsApiResponse> =>
    apiFetch<GetUserJoinRequestsApiResponse>(
      `/users/community-join-requests?page=${page}&limit=${limit}`,
      { method: "GET" }
    ),

  acceptJoinRequest: (
    communityId: number,
    data: HandleJoinRequestPayload
  ): Promise<void> =>
    apiFetch<void>(`/communities/${communityId}/requests/accept`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  rejectJoinRequest: (
    communityId: number,
    data: HandleJoinRequestPayload
  ): Promise<void> =>
    apiFetch<void>(`/communities/${communityId}/requests/reject`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  getCommunityPendingRequests: (
    communityId: number,
    page: number = 1,
    limit: number = 20
  ): Promise<GetUserJoinRequestsApiResponse> =>
    apiFetch<GetUserJoinRequestsApiResponse>(
      `/communities/${communityId}/requests?page=${page}&limit=${limit}`,
      { method: "GET" }
    ),

  getCommunityMembers: (
    communityId: number,
    page: number = 1,
    limit: number = 25,
    role_filter?: "all" | "member" | "moderator" | "owner"
  ): Promise<GetCommunityMembersApiResponse> => {
    let url = `/communities/${communityId}/members?page=${page}&limit=${limit}`;
    if (role_filter && role_filter !== "all") {
      url += `&role=${role_filter}`;
    }
    return apiFetch<GetCommunityMembersApiResponse>(url, { method: "GET" });
  },

  getTopCommunityMembers: (
    communityId: number,
    limit: number = 3
  ): Promise<GetTopCommunityMembersApiResponse> =>
    apiFetch<GetTopCommunityMembersApiResponse>(
      `/communities/${communityId}/top-members?limit=${limit}`,
      { method: "GET" }
    ),

  updateMemberRole: (
    communityId: number,
    data: UpdateMemberRoleRequestData
  ): Promise<void> =>
    apiFetch<void>(`/communities/${communityId}/members/role`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  getCommunityThreads: (
    communityId: number,
    params: {
      requesterUserId?: number | null;
      sortType?: "latest" | "top";
      filterMediaOnly?: boolean;
      page?: number;
      limit?: number;
    }
  ): Promise<FeedResponse> => {
    let url = `/communities/${communityId}/threads?page=${
      params.page || 1
    }&limit=${params.limit || 10}`;
    if (params.sortType) {
      // Backend needs to support sort_type query param
      url += `&sort=${params.sortType}`;
    }
    if (params.filterMediaOnly) {
      // Backend needs to support type=media filter
      url += "&type=media";
    }
    return apiFetch<FeedResponse>(url, { method: "GET" });
  },

  suggestCategory: (data: AISuggestionRequest): Promise<AISuggestionResponse> =>
    apiFetch<AISuggestionResponse>("/ai/suggest-category", {
      method: "POST",
      body: JSON.stringify(data),
    }),
};
