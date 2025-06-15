<script lang="ts">
    import { onDestroy, onMount } from 'svelte';
    import { link, navigate } from 'svelte-routing';
    import { api, ApiError, type UserProfileResponseData, type ThreadData, type FeedResponse, type UserProfileBasic } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
    import ThreadComponent from '../components/ThreadComponent.svelte';
    import { currentPathname } from '../stores/locationStore';
    import { timeAgo, timeAgoProfile } from '../lib/utils/timeAgo';
  import EditProfileModal from '../components/EditProfileModal.svelte';
  import ProfilePicturePreviewModal from '../components/ProfilePicturePreviewModal.svelte';
  
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
  
    let showEditProfileModal = false;
    let showProfilePicPreviewModal = false;
    let previewImageUrl: string | null = null;
  
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

  function openEditProfileModal() {
    if (profileUser?.user) {
        showEditProfileModal = true;
    }
  }

  function openReportModal() {
     console.log("Open Report Modal"); 
  }

  function handleProfileUpdated(event: CustomEvent<UserProfileBasic>) {
      if (profileUser && profileUser.user) {
          profileUser = {
              ...profileUser,
              user: event.detail
          };
      }
      console.log("ProfilePage: Profile updated from modal", event.detail);
  }

  function openProfilePicPreview(imageUrl: string | null) {
      if (imageUrl) {
          previewImageUrl = imageUrl;
          showProfilePicPreviewModal = true;
      }
  }

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
    <div class="profile-header-skeleton">
      <div class="banner-skeleton"></div>
      <div class="profile-info-bar-skeleton">
          <div class="avatar-skeleton large"></div>
          <div class="actions-skeleton">
              <div class="button-skeleton"></div>
          </div>
      </div>
      <div class="details-skeleton">
          <div class="line-skeleton name"></div>
          <div class="line-skeleton handle"></div>
          <div class="line-skeleton bio short"></div>
          <div class="line-skeleton bio long"></div>
          <div class="line-skeleton meta"></div>
          <div class="stats-skeleton">
              <div class="line-skeleton stat"></div>
              <div class="line-skeleton stat"></div>
          </div>
      </div>
  </div>
    {:else if profileError}
    <div class="error-fullpage">{profileError} <a href="/" use:link>Go Home</a></div>
    {:else if profileUser && profileUser.user}
      {@const pUser = profileUser.user}
      <header class="profile-header">
          <div class="banner-container">
              {#if pUser.banner}
                  <img src={pUser.banner} alt="{pUser.username}'s banner" class="banner-image" />
              {:else}
                  <div class="banner-placeholder"></div>
              {/if}
          </div>
          <div class="profile-info-bar">
              <div class="avatar-container" on:click={() => openProfilePicPreview(pUser.profile_picture)} on:keydown={(e) => e.key === 'Enter' && openProfilePicPreview(pUser.profile_picture)} role="button" tabindex="0">
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
              {#if isOwnProfile}
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
  
  
  {#if showEditProfileModal && profileUser?.user}
    <EditProfileModal initialUser={profileUser.user} on:close={() => showEditProfileModal = false} on:updated={handleProfileUpdated} />
  {/if}

  {#if showProfilePicPreviewModal && previewImageUrl}
    <ProfilePicturePreviewModal imageUrl={previewImageUrl} altText="{profileUser?.user?.name || 'User'}'s Profile Picture" on:close={() => showProfilePicPreviewModal = false} />
  {/if}
  
  
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
    top: 0px;
    background-color: rgba(var(--background-rgb), 0.85);
    backdrop-filter: blur(12px);
    z-index: 9;
    overflow-x: auto;
    scrollbar-width: none;
    
    &::-webkit-scrollbar {
      display: none;
    }

    button {
      flex: 1;
      min-width: max-content;
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

  .profile-header-skeleton {
    border-bottom: 1px solid var(--border-color);
    .banner-skeleton { height: 200px; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out; }
    .profile-info-bar-skeleton { display: flex; justify-content: space-between; align-items: flex-start; padding: 12px 16px; margin-top: -60px;
        .avatar-skeleton.large { width: 120px; height: 120px; border-radius: 50%; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out; border: 4px solid var(--background); }
        .actions-skeleton { display: flex; gap: 10px; margin-top: 70px;
            .button-skeleton { width: 100px; height: 36px; border-radius: 9999px; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out; }
        }
    }
    .details-skeleton { padding: 16px; display: flex; flex-direction: column; gap: 10px;
        .line-skeleton { height: 12px; border-radius: 4px; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out;
            &.name { width: 40%; height: 20px; margin-bottom: 4px; }
            &.handle { width: 30%; height: 14px; }
            &.bio.short { width: 70%; }
            &.bio.long { width: 90%; }
            &.meta { width: 50%; }
            &.stat { width: 25%; }
        }
        .stats-skeleton { display: flex; gap: 20px; margin-top: 8px; }
    }
  }

  @keyframes pulse { 0% { background-color: var(--section-hover-bg); } 50% { background-color: var(--border-color); } 100% { background-color: var(--section-hover-bg); } }
  .skeleton-thread { display: flex; padding: 12px 16px; border-bottom: 1px solid var(--border-color); gap: 12px; }
  .skeleton-avatar { width: 40px; height: 40px; border-radius: 50%; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out; flex-shrink: 0; }
  .skeleton-content { flex-grow: 1; display: flex; flex-direction: column; gap: 8px; padding-top: 4px; }
  .skeleton-line { height: 10px; border-radius: 4px; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out;
    &.short { width: 30%; } &.medium { width: 60%; } &.long { width: 90%; }
  }

  /* Responsive styles */
  @media (max-width: 1024px) {
    .profile-info-bar {
      margin-top: -50px;
    }
    
    .avatar-container {
      .profile-avatar-large, 
      .profile-avatar-placeholder-large {
        width: 100px;
        height: 100px;
        font-size: 2.5rem;
      }
    }
    
    .profile-actions {
      margin-top: 50px;
    }
  }
  
  @media (max-width: 900px) {
    .banner-container {
      height: 160px;
    }
    
    .profile-header-skeleton {
      .banner-skeleton {
        height: 160px;
      }
      
      .profile-info-bar-skeleton {
        margin-top: -45px;
        
        .avatar-skeleton.large {
          width: 90px;
          height: 90px;
        }
        
        .actions-skeleton {
          margin-top: 45px;
        }
      }
    }
  }
  
  @media (max-width: 768px) {
    .banner-container {
      height: 140px;
    }
    
    .profile-info-bar {
      margin-top: -40px;
    }
    
    .avatar-container {
      .profile-avatar-large, 
      .profile-avatar-placeholder-large {
        width: 80px;
        height: 80px;
        font-size: 2rem;
        border-width: 3px;
      }
    }
    
    .profile-actions {
      margin-top: 30px;
      
      .btn {
        padding: 6px 12px;
        font-size: 13px;
      }
    }
    
    .profile-details {
      padding: 12px;
      
      .profile-name {
        font-size: 18px;
      }
      
      .profile-bio {
        font-size: 14px;
      }
    }
    
    .profile-tabs button {
      padding: 12px;
      font-size: 14px;
    }
    
    .profile-header-skeleton {
      .banner-skeleton {
        height: 140px;
      }
      
      .profile-info-bar-skeleton {
        margin-top: -40px;
        
        .avatar-skeleton.large {
          width: 80px;
          height: 80px;
        }
        
        .actions-skeleton {
          margin-top: 30px;
          
          .button-skeleton {
            height: 32px;
            width: 80px;
          }
        }
      }
    }
  }
  
  @media (max-width: 576px) {
    .banner-container {
      height: 120px;
    }
    
    .profile-info-bar {
      flex-direction: column;
      align-items: flex-start;
      margin-top: -30px;
    }
    
    .profile-actions {
      margin-top: 10px;
      width: 100%;
      flex-wrap: wrap;
      
      .btn {
        flex: 1;
        min-width: 80px;
        text-align: center;
        padding: 6px 8px;
      }
    }
    
    .private-profile-view, .blocked-view {
      padding: 30px 15px;
      font-size: 1rem;
    }
    
    .feed-status, .empty-feed, .error-text.api-error {
      padding: 15px;
      font-size: 13px;
    }
    
    .profile-header-skeleton {
      .banner-skeleton {
        height: 120px;
      }
      
      .profile-info-bar-skeleton {
        flex-direction: column;
        align-items: flex-start;
        margin-top: -30px;
        
        .actions-skeleton {
          margin-top: 10px;
          width: 100%;
        }
      }
    }
  }
  
  @media (max-width: 480px) {
    .profile-details {
      .profile-name {
        font-size: 17px;
      }
      
      .profile-username {
        font-size: 14px;
        margin-bottom: 8px;
      }
      
      .profile-bio {
        font-size: 13px;
        margin-bottom: 8px;
        line-height: 1.3;
      }
      
      .profile-meta {
        font-size: 13px;
        margin-bottom: 8px;
        flex-wrap: wrap;
      }
      
      .profile-stats {
        font-size: 14px;
        gap: 15px;
      }
    }
    
    .profile-tabs button {
      padding: 10px 8px;
      font-size: 13px;
    }
    
    .avatar-container {
      .profile-avatar-large, 
      .profile-avatar-placeholder-large {
        width: 70px;
        height: 70px;
        font-size: 1.7rem;
      }
    }
  }
  
  @media (max-width: 400px) {
    .banner-container {
      height: 100px;
    }
    
    .profile-info-bar {
      margin-top: -25px;
      padding: 8px 12px;
    }
    
    .avatar-container {
      .profile-avatar-large, 
      .profile-avatar-placeholder-large {
        width: 60px;
        height: 60px;
        font-size: 1.5rem;
        border-width: 2px;
      }
    }
    
    .profile-details {
      padding: 8px;
    }
    
    .profile-tabs {
      button {
        padding: 8px 6px;
        font-size: 12px;
      }
    }
    
    .profile-header-skeleton {
      .banner-skeleton {
        height: 100px;
      }
      
      .profile-info-bar-skeleton {
        margin-top: -25px;
        padding: 8px 12px;
        
        .avatar-skeleton.large {
          width: 60px;
          height: 60px;
          border-width: 2px;
        }
      }
      
      .details-skeleton {
        padding: 8px;
        gap: 8px;
      }
    }
    
    .skeleton-thread {
      padding: 10px 12px;
      gap: 8px;
    }
    
    .skeleton-avatar {
      width: 32px;
      height: 32px;
    }
  }
</style>