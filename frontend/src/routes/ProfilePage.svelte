<script lang="ts">
    import { onDestroy, onMount } from 'svelte';
    import { link, navigate } from 'svelte-routing';
    import { api, ApiError, type UserProfileResponseData, type ThreadData, type FeedResponse } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
    import ThreadComponent from '../components/ThreadComponent.svelte';
  import { currentPathname } from '../stores/locationStore';
  import { timeAgo, timeAgoProfile } from '../lib/utils/timeAgo';
  
    let profileUser: UserProfileResponseData | null = null;
    let profileThreads: ThreadData[] = [];
    let activeTab: 'posts' | 'replies' | 'likes' | 'media' = 'posts';
    let isLoadingProfile = true;
    let isLoadingThreads = false;
    let profileError: string | null = null;
    let threadsError: string | null = null;
  
    let currentThreadsPage = 1;
    let hasMoreThreads = true;
    let threadsSentinel: Element;
    let threadsObserver: IntersectionObserver;
  
  
    let isOwnProfile = false;
    let usernameFromUrl: string | null = null;
  
    let pathUnsubscribe: (() => void) | null = null;

    function getUsernameFromPath(path: string | null): string | null {
        if (!path) return null;
        const parts = path.split('/');
        // Assuming route is /profile/:username, username is the 3rd part (index 2)
        if (parts.length >= 3 && parts[1] === 'profile') {
        return parts[2];
        }
        return null;
    }

    // --- Data Fetching ---
    async function fetchProfileData(username: string | null) {
        if (!username) {
            profileError = "Username not found in URL.";
            isLoadingProfile = false;
            return;
        }
      isLoadingProfile = true;
      profileError = null;
      profileUser = null;
      profileThreads = [];
      activeTab = 'posts';
      currentThreadsPage = 1;
      hasMoreThreads = true;
  
      console.log(`Fetching profile for: ${username}`);
      try {
        isOwnProfile = $currentUserStore?.username === username;

        profileUser = await api.getUserProfileByUsername(username);
        console.log("Profile data:", profileUser);
        if (profileUser?.user) {
            fetchProfileThreads(profileUser.user.username, activeTab, 1, true);
        } else if (!profileUser) {
            profileError = "User not found.";
        }
        } catch (err) {
        console.error("Error fetching profile data:", err);
        if (err instanceof ApiError) {
            profileError = err.status === 404 ? "User not found." : `Error: ${err.message}`;
        } else if (err instanceof Error) { profileError = err.message; }
        else { profileError = "An unexpected error occurred."; }
        profileUser = null;
        } finally {
        isLoadingProfile = false;
        }
    }
  
    async function fetchProfileThreads(username: string, type: 'posts' | 'replies' | 'likes' | 'media', page = 1, reset = false) {
      if (isLoadingThreads || (!hasMoreThreads && !reset)) return;
      isLoadingThreads = true;
      if (reset) {
          profileThreads = [];
          threadsError = null;
          currentThreadsPage = 1;
          hasMoreThreads = true;
      }
  
      try {
        const response: FeedResponse = await api.getUserThreads(username, type, page);
        if (response && response.threads) {
          profileThreads = reset ? response.threads : [...profileThreads, ...response.threads];
          currentThreadsPage = page;
          hasMoreThreads = response.has_more ?? (response.threads.length === 10);
        } else {
          hasMoreThreads = false;
        }
      } catch (err) {
        console.error(`Error fetching ${type} for ${username}:`, err);
        threadsError = `Could not load ${type}.`;
        hasMoreThreads = false;
      } finally {
        isLoadingThreads = false;
      }
    }
  
  let socialActionLoading = false;
  async function handleFollow() {
    if (!profileUser?.user || socialActionLoading) return;
    socialActionLoading = true;
    try {
      if (profileUser.is_followed_by_requester) {
        await api.unfollowUser(profileUser.user.username);
        profileUser.is_followed_by_requester = false;
        profileUser.follower_count--;
      } else {
        await api.followUser(profileUser.user.username);
        profileUser.is_followed_by_requester = true;
        if (!profileUser.follower_count) {
            profileUser.follower_count = 1;
        } else if (profileUser.follower_count > 0) {
            profileUser.follower_count++;
        }
      }
      profileUser = { ...profileUser };
    } catch (err) { console.error("Follow/Unfollow error:", err);}
    finally { socialActionLoading = false; }
  }

  async function handleBlock() {
    if (!profileUser?.user || socialActionLoading) return;
    if (!confirm(`Are you sure you want to ${profileUser.is_blocked_by_requester ? 'unblock' : 'block'} @${profileUser.user.username}?`)) return;

    socialActionLoading = true;
    try {
      if (profileUser.is_blocked_by_requester) {
        await api.unblockUser(profileUser.user.username);
        profileUser.is_blocked_by_requester = false;
      } else {
        await api.blockUser(profileUser.user.username);
        profileUser.is_blocked_by_requester = true;
        if(profileUser.is_followed_by_requester) {
            profileUser.is_followed_by_requester = false;
            profileUser.follower_count--;
        }
      }
      profileUser = { ...profileUser };
    } catch (err) { console.error("Block/Unblock error:", err);}
    finally { socialActionLoading = false; }
  }
  function openEditProfileModal() { console.log("Open Edit Profile Modal"); }
  function openReportModal() { console.log("Open Report Modal"); }


  onMount(() => {
    const initialUsername = getUsernameFromPath(window.location.pathname);
    if (initialUsername) {
        usernameFromUrl = initialUsername;
        fetchProfileData(usernameFromUrl);
    }

    pathUnsubscribe = currentPathname.subscribe(newPath => {
        if (newPath) {
            const newUsername = getUsernameFromPath(newPath);
            if (newUsername && newUsername !== usernameFromUrl) {
                console.log(`Path store changed, new username: ${newUsername}`);
                usernameFromUrl = newUsername;
                fetchProfileData(usernameFromUrl);
            } else if (newUsername === null && newPath.startsWith('/profile/')) {
                profileError = "Invalid profile URL.";
                isLoadingProfile = false;
                profileUser = null;
            }
        }
    });

    // Setup Intersection Observer
    threadsObserver = new IntersectionObserver(handleThreadIntersect, { threshold: 0.1 });
    if (threadsSentinel) threadsObserver.observe(threadsSentinel);
  });

  onDestroy(() => {
    if (pathUnsubscribe) pathUnsubscribe();
    if (threadsObserver && threadsSentinel) threadsObserver.unobserve(threadsSentinel);
  });


  // --- Infinite Scroll for Threads ---
  function handleThreadIntersect(entries: IntersectionObserverEntry[]) {
      const entry = entries[0];
      if (entry.isIntersecting && hasMoreThreads && !isLoadingThreads && usernameFromUrl && profileUser?.user) {
          fetchProfileThreads(profileUser.user.username, activeTab, currentThreadsPage + 1);
      }
  }

  function switchTab(tab: 'posts' | 'replies' | 'likes' | 'media') {
    if (!usernameFromUrl || activeTab === tab || !profileUser?.user) return; 
    activeTab = tab;
    fetchProfileThreads(profileUser.user.username, tab, 1, true);
  }

  function handleThreadDelete(event: CustomEvent<{ id: number }>) {
      profileThreads = profileThreads.filter(t => t.id !== event.detail.id);
  }


  </script>
  
  <div class="profile-page-container">
    {#if isLoadingProfile && !profileUser}
      <p>Loading profile...</p> <!-- TODO: Skeleton Loader for profile header -->
    {:else if profileError}
      <div class="error-fullpage">{profileError} <a href="/" use:link>Go Home</a></div>
    {:else if profileUser && profileUser.user}
      {@const pUser = profileUser.user} <!-- Alias for easier access -->
      <header class="profile-header">
          <div class="banner-container">
              {#if pUser.banner}
                  <img src={pUser.banner} alt="{pUser.username}'s banner" class="banner-image" />
              {:else}
                  <div class="banner-placeholder"></div>
              {/if}
          </div>
          <div class="profile-info-bar">
              <div class="avatar-container">
                  {#if pUser.profile_picture}
                      <img src={pUser.profile_picture} alt="{pUser.username}'s profile" class="profile-avatar-large" />
                  {:else}
                      <div class="profile-avatar-placeholder-large">{pUser.name?.charAt(0)?.toUpperCase() ?? '?'}</div>
                  {/if}
              </div>
              <div class="profile-actions">
                  {#if isOwnProfile}
                      <button class="btn btn-secondary" on:click={openEditProfileModal}>Edit profile</button>
                  {:else}
                      <!-- TODO: Message button, More options (block/report) -->
                      <button class="btn" class:btn-primary={!profileUser.is_followed_by_requester} class:btn-secondary={profileUser.is_followed_by_requester} on:click={handleFollow} disabled={socialActionLoading}>
                          {profileUser.is_followed_by_requester ? 'Following' : 'Follow'}
                      </button>
                       <button class="btn btn-secondary" on:click={handleBlock} disabled={socialActionLoading}>
                          {profileUser.is_blocked_by_requester ? 'Unblock' : 'Block'}
                      </button>
                       <button class="btn btn-secondary" on:click={openReportModal} disabled={socialActionLoading}>Report</button>
                  {/if}
              </div>
          </div>
  
          <div class="profile-details">
              <h1 class="profile-name">{pUser.name}</h1>
              <p class="profile-username">@{pUser.username}</p>
              {#if pUser.bio}
                  <p class="profile-bio">{pUser.bio}</p>
              {/if}
              <div class="profile-meta">
                  <!-- TODO: Location, Website icons/links -->
                  <span>📅 Joined {timeAgoProfile(pUser.created_at)}</span>
              </div>
              <div class="profile-stats">
                  <a href="/profile/{pUser.username}/following" use:link class="stat-link">
                      <strong>{profileUser.following_count}</strong> Following
                  </a>
                   <a href="/profile/{pUser.username}/followers" use:link class="stat-link">
                      <strong>{profileUser.follower_count}</strong> Followers
                  </a>
              </div>
          </div>
      </header>
  
      <!-- Tabs for Profile Content -->
      {#if profileUser.is_blocking_requester && !isOwnProfile}
          <p class="blocked-view">You cannot view @{pUser.username}'s full profile because they have blocked you.</p>
      {:else if pUser.account_privacy === 'private' && !isOwnProfile && !profileUser.is_followed_by_requester}
          <p class="private-profile-view">This account is private. Follow them to see their posts.</p>
      {:else}
          <nav class="profile-tabs">
              <button class:active={activeTab === 'posts'} on:click={() => switchTab('posts')}>Posts</button>
              <button class:active={activeTab === 'replies'} on:click={() => switchTab('replies')}>Replies</button>
              {#if isOwnProfile || pUser.account_privacy === 'public' || profileUser.is_followed_by_requester}
                  <button class:active={activeTab === 'likes'} on:click={() => switchTab('likes')}>Likes</button>
              {/if}
              <button class:active={activeTab === 'media'} on:click={() => switchTab('media')}>Media</button>
          </nav>
  
          <section class="profile-feed">
              {#if isLoadingThreads && profileThreads.length === 0}
                  <!-- Initial Tab Load Skeleton -->
                  {#each { length: 3 } as _} <div class="skeleton-thread">...</div> {/each}
              {:else if threadsError}
                  <p class="error-text api-error">{threadsError}</p>
              {:else if profileThreads.length > 0}
                  {#each profileThreads as thread (thread.id)}
                      <ThreadComponent {thread} on:delete={handleThreadDelete} />
                  {/each}
              {:else if !isLoadingThreads}
                  <p class="empty-feed">@{pUser.username} hasn't posted any {activeTab} yet.</p>
              {/if}
              <!-- Sentinel for thread infinite scroll -->
              <div class="feed-status" bind:this={threadsSentinel}>
                  {#if isLoadingThreads && profileThreads.length > 0} <p>Loading more...</p> {/if}
                  {#if !hasMoreThreads && profileThreads.length > 0 && !isLoadingThreads} <p>You've reached the end.</p> {/if}
              </div>
          </section>
      {/if}
  
    {:else}
      <p>Profile loading or something went wrong...</p>
    {/if}
  </div>
  
  
  <!-- Modals to be implemented -->
  <!-- {#if showEditProfileModal} <EditProfileModal on:close ... /> {/if} -->
  <!-- {#if showReportModal} <ReportModal on:close ... /> {/if} -->
  
  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .profile-page-container {
      width: 100%;
      min-height: 100vh;
    }
  
    .error-fullpage {
        display: flex; flex-direction: column; align-items: center; justify-content: center;
        height: 60vh; text-align: center; color: var(--error-color);
        a { margin-top: 1rem; color: var(--primary-color); }
    }
  
    .profile-header {
      border-bottom: 1px solid var(--border-color);
    }
  
    .banner-container {
      height: 200px;
      background-color: var(--secondary-text-color);
      .banner-image {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
      .banner-placeholder {
          width: 100%; height: 100%;
          background: linear-gradient(45deg, var(--secondary-text-color), var(--border-color));
      }
    }
  
    .profile-info-bar {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      padding: 12px 16px;
      margin-top: -60px;
    }
  
    .avatar-container {
      .profile-avatar-large {
        width: 120px;
        height: 120px;
        border-radius: 50%;
        border: 4px solid var(--background);
        object-fit: cover;
        background-color: var(--secondary-text-color);
      }
      .profile-avatar-placeholder-large {
          width: 120px; height: 120px; border-radius: 50%;
          border: 4px solid var(--background);
          background-color: var(--secondary-text-color); color: var(--background);
          display: flex; align-items: center; justify-content: center;
          font-size: 3rem; font-weight: bold;
      }
    }
  
    .profile-actions {
      display: flex;
      gap: 10px;
      margin-top: 70px;
       .btn {
          padding: 8px 16px;
          border-radius: 9999px;
          font-weight: bold;
          font-size: 14px;
          cursor: pointer;
          border: 1px solid var(--border-color);
          background-color: transparent;
          color: var(--text-color);
          transition: background-color 0.2s ease;
           &:hover { background-color: var(--section-hover-bg); }
  
           &.btn-primary {
              background-color: var(--follow-button-bg);
              color: var(--follow-button-text);
              border-color: var(--follow-button-border);
               &:hover { background-color: var(--follow-button-hover-bg); }
           }
           &.btn-secondary {
              
           }
       }
    }
  
    .profile-details {
      padding: 16px;
      .profile-name {
        font-size: 20px;
        font-weight: 800;
        margin: 0;
      }
      .profile-username {
        font-size: 15px;
        color: var(--secondary-text-color);
        margin-bottom: 12px;
      }
      .profile-bio {
        font-size: 15px;
        line-height: 1.4;
        margin-bottom: 12px;
        white-space: pre-wrap;
      }
      .profile-meta {
        font-size: 15px;
        color: var(--secondary-text-color);
        display: flex;
        gap: 12px;
        margin-bottom: 12px;
        span { display: flex; align-items: center; gap: 4px; }
      }
      .profile-stats {
        display: flex;
        gap: 20px;
        font-size: 15px;
        .stat-link {
            color: var(--secondary-text-color);
            text-decoration: none;
             &:hover { text-decoration: underline; }
             strong { color: var(--text-color); font-weight: bold; }
        }
      }
    }
  
    .private-profile-view, .blocked-view {
        text-align: center;
        padding: 40px 20px;
        color: var(--secondary-text-color);
        font-size: 1.1rem;
        border-top: 1px solid var(--border-color);
    }
  
  
    .profile-tabs {
      display: flex;
      border-bottom: 1px solid var(--border-color);
      position: sticky;
      top: 57px;
      background-color: rgba(var(--background-rgb), 0.85);
      backdrop-filter: blur(12px);
      z-index: 9;
  
      button {
        flex: 1;
        padding: 16px;
        background: none; border: none;
        color: var(--secondary-text-color);
        font-weight: bold; font-size: 15px;
        cursor: pointer; position: relative;
        transition: background-color 0.2s ease;
        &:hover { background-color: var(--section-hover-bg); }
        &.active {
          color: var(--text-color);
          &::after {
            content: ''; position: absolute; bottom: 0; left: 0; right: 0;
            height: 4px; background-color: var(--primary-color); border-radius: 2px;
          }
        }
      }
    }
  
    .profile-feed {
      
    }
  
    .feed-status, .empty-feed, .error-text.api-error {
        text-align: center; padding: 20px; color: var(--secondary-text-color); font-size: 14px;
    }
    .skeleton-thread {}
  
  </style>