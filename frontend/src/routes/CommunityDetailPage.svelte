<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { link, navigate } from 'svelte-routing';
    import { api, ApiError, CommunityStatusNumbers, type CommunityFullDetailsResponseData, type JoinRequestItem, type UserSummary } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
    import { tick } from 'svelte';
  import { timeAgoProfile } from '../lib/utils/timeAgo';
    // Import ThreadComponent if you were to display posts
    // import ThreadComponent from '../components/ThreadComponent.svelte';
  
    export let id: string; // From route param /community/:id - svelte-routing passes strings
  
    let communityDetailsResponse: CommunityFullDetailsResponseData | null = null;
    let communityIdNum: number = 0;
    let activeTab: 'top' | 'latest' | 'media' | 'about' | 'manage_members' = 'top'; // Default tab
  
    // State for "Manage Members" tab
    let pendingJoinRequests: JoinRequestItem[] = [];
    let isLoadingPendingRequests = false;
    let manageMembersError: string | null = null;
    // TODO: State for listing current members if needed for management beyond just requests
  
    let isLoading = true;
    let error: string | null = null;
    let joinRequestLoading = false;
  
    async function fetchCommunityDetails(communityIdentifier: number) {
      isLoading = true; error = null;
      try {
        communityDetailsResponse = await api.getCommunityDetails(communityIdentifier);
        // If this page has tabs that fetch data, trigger fetch for default tab
        if (communityDetailsResponse?.community) {
            console.log("Fetched community details:", communityDetailsResponse.community);
          if (activeTab === 'manage_members' && isModeratorOrOwner()) {
              fetchPendingRequests(communityDetailsResponse.community.id);
          }
          // Add fetches for other tabs like posts, latest, media when implemented
        }
      } catch (e) {
        console.error("Error fetching community details:", e);
        if (e instanceof ApiError && e.status === 404) error = "Community not found.";
        else error = "Could not load community details.";
      } finally {
        isLoading = false;
      }
    }
  
    // --- Tab Specific Data Fetching ---
    async function fetchPendingRequests(commId: number) {
        isLoadingPendingRequests = true; manageMembersError = null;
        try {
            const response = await api.getCommunityPendingRequests(commId, 1, 50); // Fetch first 50 for now
            pendingJoinRequests = response.requests || [];
        } catch (err) {
            console.error("Error fetching pending join requests:", err);
            manageMembersError = "Could not load join requests.";
        } finally {
            isLoadingPendingRequests = false;
        }
    }
  
  
    $: communityIdNum = parseInt(id, 10); // Convert string param to number
    $: if (communityIdNum && !isNaN(communityIdNum)) {
      fetchCommunityDetails(communityIdNum);
    }
  
    function switchTab(tab: 'top' | 'latest' | 'media' | 'about' | 'manage_members') {
      if (activeTab === tab) return;
      activeTab = tab;
      // Fetch data for the new tab if needed
      if (tab === 'manage_members' && communityDetailsResponse?.community && isModeratorOrOwner()) {
          pendingJoinRequests = []; // Clear previous
          fetchPendingRequests(communityDetailsResponse.community.id);
      }
      // Add logic for other tabs (Top, Latest, Media will fetch threads)
    }
  
    function isModeratorOrOwner(): boolean {
        return communityDetailsResponse?.requester_role === 'owner' || communityDetailsResponse?.requester_role === 'moderator';
    }
  
  
    async function handleRequestToJoinCommunity() {
        if (!communityDetailsResponse?.community || joinRequestLoading) return;
        joinRequestLoading = true; error = null; // Clear main error
        try {
            await api.requestToJoinCommunity(communityDetailsResponse.community.id);
            // Optimistically update or refetch details to show "Pending"
            if (communityDetailsResponse) { // Refetch to get updated status
                fetchCommunityDetails(communityDetailsResponse.community.id);
            }
        } catch (err) {
            if (err instanceof ApiError) error = `Join request failed: ${err.message}`;
            else error = "Could not send join request.";
        } finally {
            joinRequestLoading = false;
        }
    }
  
    async function handleJoinRequest(requestId: number, communityId: number, targetUserId: number, action: 'accept' | 'reject') {
        if (!confirm(`Are you sure you want to ${action} this join request?`)) return;
        manageMembersError = null; isLoadingPendingRequests = true; // Indicate loading
        try {
            const payload = { target_user_id: targetUserId };
            if (action === 'accept') {
                await api.acceptJoinRequest(communityId, payload);
            } else {
                await api.rejectJoinRequest(communityId, payload);
            }
            // Remove from pending list and potentially update member count/list
            pendingJoinRequests = pendingJoinRequests.filter(req => req.request_id !== requestId);
            // TODO: Optionally, update member count on communityDetailsResponse or refetch all
            alert(`Request ${action === 'accept' ? 'accepted' : 'rejected'}.`);
  
        } catch (err) {
            if (err instanceof ApiError) manageMembersError = `Failed to ${action} request: ${err.message}`;
            else manageMembersError = `Could not ${action} request.`;
        } finally {
            isLoadingPendingRequests = false;
        }
    }
  
  </script>
  
  <div class="community-detail-page">
    {#if isLoading && !communityDetailsResponse}
      <!-- Full Page Skeleton Loader for Profile Header -->
      <div class="profile-header-skeleton"> /* ... (copy from ProfilePage.svelte) ... */ </div>
    {:else if error}
      <div class="error-fullpage">{error} <a href="/communities" use:link>Back to Communities</a></div>
    {:else if communityDetailsResponse && communityDetailsResponse.community}
      {@const comm = communityDetailsResponse.community}
      {@const requesterContext = communityDetailsResponse}
  
      <header class="community-detail-header">
        <div class="banner-container">
          {#if comm.banner_url} <img src={comm.banner_url} alt="{comm.name} banner" class="banner-image"/>
          {:else} <div class="banner-placeholder"></div> {/if}
        </div>
        <div class="info-bar">
            <div class="icon-container">
              {#if comm.icon_url} <img src={comm.icon_url} alt="{comm.name} icon" class="community-icon-large"/>
              {:else} <div class="icon-placeholder-large">{comm.name?.charAt(0)?.toUpperCase() ?? 'C'}</div> {/if}
            </div>
            <div class="actions">
                {#if !$currentUserStore}
                    <!-- Guest view: Maybe show login to join -->
                    <a href="/login?redirect=/community/{comm.id}" use:link class="btn btn-primary">Login to join</a>
                {:else if requesterContext.is_joined_by_requester}
                    <button class="btn btn-secondary" disabled>Joined ({requesterContext.requester_role})</button>
                    <!-- TODO: Options: Leave Community, Notifications Settings for this community -->
                {:else if requesterContext.has_pending_request_by_requester}
                    <button class="btn btn-secondary" disabled>Request Pending</button>
                {:else if comm.status === CommunityStatusNumbers.ACTIVE}
                    <button class="btn btn-primary" on:click={handleRequestToJoinCommunity} disabled={joinRequestLoading}>
                        {joinRequestLoading ? 'Joining...' : 'Request to Join'}
                    </button>
                {:else if comm.status === CommunityStatusNumbers.PENDING_APPROVAL}
                    <button class="btn btn-secondary" disabled>Pending Admin Approval</button>
                {/if}
            </div>
        </div>
  
        <div class="community-main-info">
            <h1>{comm.name}</h1>
            {#if comm.categories && comm.categories.length > 0}
              <div class="community-categories">
                  {#each comm.categories as category (category)}
                      <a href="/explore?category={category}" use:link class="category-tag">#{category}</a>
                  {/each}
              </div>
            {/if}
            {#if comm.description}
              <p class="description">{comm.description}</p>
            {/if}
            <div class="meta-info">
              <span>👥 {comm.member_count} member{comm.member_count !== 1 ? 's' : ''}</span>
              <span>📅 Created {timeAgoProfile(comm.created_at)}</span>
            </div>
        </div>
      </header>
  
      <nav class="profile-tabs community-tabs">
          <button class:active={activeTab === 'top'} on:click={() => switchTab('top')}>Top</button>
          <button class:active={activeTab === 'latest'} on:click={() => switchTab('latest')}>Latest</button>
          <button class:active={activeTab === 'media'} on:click={() => switchTab('media')}>Media</button>
          <button class:active={activeTab === 'about'} on:click={() => switchTab('about')}>About</button>
          {#if isModeratorOrOwner()}
              <button class:active={activeTab === 'manage_members'} on:click={() => switchTab('manage_members')}>Manage Members</button>
              <!-- <button>Settings</button> -->
          {/if}
      </nav>
  
      <section class="community-tab-content">
          {#if activeTab === 'about'}
              <div class="about-section">
                  <h4>About {comm.name}</h4>
                  {#if comm.description} <p>{comm.description}</p> {/if}
  
                  {#if comm.creator_summary}
                      <p><strong>Created by:</strong>
                          <a href="/profile/{comm.creator_summary.username}" use:link class="creator-link">
                              {comm.creator_summary.name} (@{comm.creator_summary.username})
                          </a>
                      </p>
                  {/if}
                  <p><strong>Created on:</strong> {timeAgoProfile(comm.created_at)}</p>
  
                  {#if comm.rules && comm.rules.length > 0}
                      <h4>Community Rules</h4>
                      <ol class="rules-list">
                          {#each comm.rules as rule, i (i)}
                              <li>{rule}</li>
                          {/each}
                      </ol>
                  {/if}
  
                  <h4>Moderators</h4>
                  <!-- TODO: List moderators (fetch from GetCommunityMembers with role_filter) -->
                  <p>Moderator list coming soon.</p>
              </div>
  
          {:else if activeTab === 'manage_members'}
              {#if isModeratorOrOwner()}
                  <div class="manage-members-section">
                      <h4>Manage Join Requests</h4>
                      {#if isLoadingPendingRequests} <p>Loading requests...</p>
                      {:else if manageMembersError} <p class="error-text">{manageMembersError}</p>
                      {:else if pendingJoinRequests.length > 0}
                          <ul class="join-request-list">
                          {#each pendingJoinRequests as request (request.request_id)}
                              <li class="join-request-item">
                                  <div class="user-info-join">
                                      {#if request.user.profile_picture_url} <img src={request.user.profile_picture_url} alt={request.user.name} class="avatar-small-join"/>
                                      {:else}<div class="avatar-initials-join">{request.user.name.charAt(0).toUpperCase()}</div>{/if}
                                      <span>{request.user.name} (@{request.user.username})</span>
                                  </div>
                                  <div class="request-actions">
                                      <button class="btn btn-success small" on:click={() => handleJoinRequest(request.request_id, comm.id, request.user.id, 'accept')}>Accept</button>
                                      <button class="btn btn-danger small" on:click={() => handleJoinRequest(request.request_id, comm.id, request.user.id, 'reject')}>Reject</button>
                                  </div>
                              </li>
                          {/each}
                          </ul>
                      {:else}
                          <p>No pending join requests.</p>
                      {/if}
                      <!-- TODO: Section for managing existing members (kick, change role) -->
                  </div>
              {:else}
                   <p>You do not have permission to manage members.</p>
              {/if}
  
  
          {:else if activeTab === 'top' || activeTab === 'latest' || activeTab === 'media'}
              <p>'{activeTab}' threads for this community will appear here.</p>
              <!-- TODO: Fetch and display threads (use ThreadComponent) filtered by community ID and tab type -->
          {/if}
      </section>
  
    {:else}
      <p>Community not found or an error occurred.</p>
    {/if}
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
    .community-detail-page { width: 100%; }
    .community-detail-header {
        border-bottom: 1px solid var(--border-color);
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
        .info-bar { display: flex; justify-content: space-between; align-items: flex-end; padding: 12px 16px; margin-top: -50px; }

        .icon-container {
            .community-icon-large {
                width: 120px;
                height: 120px;
                border-radius: 50%;
                border: 4px solid var(--background);
                object-fit: cover;
                background-color: var(--secondary-text-color);
            }
            .icon-placeholder-large {
                width: 120px; height: 120px; border-radius: 50%;
                border: 4px solid var(--background);
                background-color: var(--secondary-text-color); color: var(--background);
                display: flex; align-items: center; justify-content: center;
                font-size: 3rem; font-weight: bold;
            }
        }

        .actions {
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
        h1 { padding: 0 16px; margin-top: 12px; font-size: 1.8rem; font-weight: bold;}
        .description { padding: 0 16px; margin-top: 4px; color: var(--text-color); font-size: 15px; }
        .meta-info { padding: 8px 16px 16px; display: flex; gap: 12px; color: var(--secondary-text-color); font-size: 15px; }
    }
    .community-feed { padding: 16px; }
    .community-main-info {
        padding: 16px;
        h1 { font-size: 1.8rem; font-weight: 800; margin:0 0 4px; }
        .community-categories {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            margin: 8px 0;
            .category-tag {
                background-color: var(--section-hover-bg);
                color: var(--primary-color);
                padding: 4px 8px;
                border-radius: 12px;
                font-size: 0.8rem;
                font-weight: 500;
                text-decoration: none;
                &:hover {
                    background-color: rgba(var(--primary-color-rgb), 0.2);
                }
            }
        }
        .description { /* ... from before ... */ margin-bottom: 12px; }
        .meta-info { /* ... from before ... */ }
    }
  
  
    .community-tabs {
      display: flex; border-bottom: 1px solid var(--border-color);
      position: sticky; top: 0; /* Make tabs sticky under the potentially non-sticky header */
      background-color: rgba(var(--background-rgb), 0.90); /* More opaque */
      backdrop-filter: blur(10px);
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
  
    .community-tab-content {
        padding: 16px;
    }
  
    .about-section {
        font-size: 15px;
        line-height: 1.6;
        color: var(--text-color);
        h4 { font-size: 1.1rem; font-weight: 700; margin: 1.5rem 0 0.5rem; }
        p { margin-bottom: 0.8rem; }
        .creator-link { color: var(--primary-color); text-decoration: none; &:hover {text-decoration: underline;} }
        .rules-list {
            list-style-type: decimal;
            padding-left: 20px;
            li { margin-bottom: 0.5rem; }
        }
    }
  
    .manage-members-section {
        h4 { font-size: 1.1rem; font-weight: 700; margin: 1.5rem 0 0.5rem; }
    }
    .join-request-list {
        list-style: none; padding: 0; margin: 0;
    }
    .join-request-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 0;
        border-bottom: 1px solid var(--border-color);
        &:last-child { border-bottom: none; }
  
        .user-info-join {
            display: flex; align-items: center; gap: 10px;
            .avatar-small-join, .avatar-initials-join {
                width: 36px; height: 36px; border-radius: 50%;
                background-color: var(--secondary-text-color); color: var(--background);
                display: flex; align-items: center; justify-content: center; font-weight: bold;
                img { width: 100%; height: 100%; object-fit: cover; border-radius: 50%;}
            }
            span { font-weight: 500; }
        }
        .request-actions {
            display: flex; gap: 8px;
            .btn.small { padding: 4px 10px; font-size: 0.85rem; font-weight: 600; border-radius: 12px;}
            .btn-success { background-color: var(--success-color); color: white; border: none; &:hover { opacity: 0.9;}}
            .btn-danger { background-color: var(--error-color); color: white; border: none; &:hover { opacity: 0.9;}}
        }
    }
  
    .feed-status, .empty-feed, .error-text.api-error {
        text-align: center; padding: 20px; color: var(--secondary-text-color); font-size: 14px;
    }

    @keyframes pulse { 0% { background-color: var(--section-hover-bg); } 50% { background-color: var(--border-color); } 100% { background-color: var(--section-hover-bg); } }
    .skeleton-thread { display: flex; padding: 12px 16px; border-bottom: 1px solid var(--border-color); gap: 12px; }
    .skeleton-avatar { width: 40px; height: 40px; border-radius: 50%; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out; flex-shrink: 0; }
    .skeleton-content { flex-grow: 1; display: flex; flex-direction: column; gap: 8px; padding-top: 4px; }
    .skeleton-line { height: 10px; border-radius: 4px; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out;
      &.short { width: 30%; } &.medium { width: 60%; } &.long { width: 90%; }
    }
  </style>