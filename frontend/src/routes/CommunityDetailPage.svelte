<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { link, navigate } from 'svelte-routing';
    import { api, ApiError, CommunityStatusNumbers, type CommunityFullDetailsResponseData, type CommunityMemberDetails, type JoinRequestItem, type ThreadData, type UpdateMemberRoleRequestData, type UserProfileBasic, type UserSummary } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
    import { tick } from 'svelte';
    import { timeAgoProfile } from '../lib/utils/timeAgo';
    import UserCard from '../components/UserCard.svelte';
    import UserCardSummary from '../components/UserCardSummary.svelte';
    import ThreadComponent from '../components/ThreadComponent.svelte';
  
    export let id: string; // From route param /community/:id
  
    let communityDetailsResponse: CommunityFullDetailsResponseData | null = null;
    let communityIdNum: number = 0;
    type CommunityDetailTab = 'top' | 'latest' | 'media' | 'about' | 'manage_members';
    let activeTab: CommunityDetailTab = 'top';
  
    // Members Modal (click on members count)
    let showMembersModal = false;
    let membersModalTab: 'members' | 'moderators' = 'members';
    let membersList: CommunityMemberDetails[] = [];
    let moderatorsList: CommunityMemberDetails[] = [];
    let isLoadingMembers = false;
    let membersModalError: string | null = null;
    let memberSearchQuery = ''; // For searching within the modal

    // State for Thread tabs ("Top", "Latest", "Media")
    let communityThreads: ThreadData[] = [];
    let topPostsThreads: ThreadData[] = [];
    let isLoadingCommunityThreads = false;
    let communityThreadsError: string | null = null;
    let currentCommunityThreadsPage = 1;
    let hasMoreCommunityThreads = true;
    let communityThreadsSentinel: Element;
    let communityThreadsObserver: IntersectionObserver;
    // State for Top Tab specifically if it has different content
    let topMembersForDisplay: UserProfileBasic[] = []; // Top 3 members

    // State for "Manage Members" tab
    let pendingJoinRequests: JoinRequestItem[] = [];
    let isLoadingPendingRequests = false;
    let manageMembersError: string | null = null;
    // TODO: State for listing current members if needed for management beyond just requests
  
    let isLoading = true;
    let error: string | null = null;
    let joinRequestLoading = false;
    let roleUpdateLoadingForUser: number | null = null;
  
    async function fetchCommunityDetails(communityIdentifier: number) {
      isLoading = true; error = null;
      try {
        communityDetailsResponse = await api.getCommunityDetails(communityIdentifier);
        // If this page has tabs that fetch data, trigger fetch for default tab
        if (communityDetailsResponse?.community) {
            console.log("Fetched community details:", communityDetailsResponse.community);
            // Initial fetch for the default tab (Top)
            switchTab(activeTab, true);
        }
      } catch (e) {
        console.error("Error fetching community details:", e);
        if (e instanceof ApiError && e.status === 404) error = "Community not found.";
        else error = "Could not load community details.";
      } finally {
        isLoading = false;
      }
    }

    async function fetchThreadsForCurrentTab(page = 1, resetList = false) {
        if (!communityDetailsResponse?.community || isLoadingCommunityThreads || (!hasMoreCommunityThreads && !resetList)) return;

        isLoadingCommunityThreads = true;
        if (resetList) {
            communityThreads = [];
            topPostsThreads = [];
            currentCommunityThreadsPage = 1;
            hasMoreCommunityThreads = true;
            communityThreadsError = null;
        }

        const commId = communityDetailsResponse.community.id;
        let sortType: 'latest' | 'top' | undefined = undefined;
        let filterMediaOnly = false;

        if (activeTab === 'latest') sortType = 'latest';
        else if (activeTab === 'top') sortType = 'top'; // Backend needs to handle "top" sort
        else if (activeTab === 'media') filterMediaOnly = true;
        else { isLoadingCommunityThreads = false; return; } // Not a thread tab

        try {
        const response = await api.getCommunityThreads(commId, {
            requesterUserId: $currentUserStore?.id,
            page: currentCommunityThreadsPage,
            limit: 10, // Or your preferred limit
            sortType: sortType,
            filterMediaOnly: filterMediaOnly,
        });
        if (response && response.threads) {
            communityThreads = resetList ? response.threads : [...communityThreads, ...response.threads];

            if (activeTab === 'top') {
                topPostsThreads = [...communityThreads].sort((a, b) => (b.like_count ?? 0) - (a.like_count ?? 0));
            }

            currentCommunityThreadsPage = page;
            hasMoreCommunityThreads = response.has_more;
        } else { hasMoreCommunityThreads = false; }
        } catch (err) {
        console.error(`Error fetching ${activeTab} threads:`, err);
        communityThreadsError = `Could not load ${activeTab} threads.`;
        hasMoreCommunityThreads = false;
        } finally {
        isLoadingCommunityThreads = false;
        }
    }

    async function fetchTopMembers(commId: number) {
        try {
            const response = await api.getTopCommunityMembers(commId, 3); // Fetches top 3
            topMembersForDisplay = response.users || [];
            console.log("Fetched top community members:", topMembersForDisplay);
        } catch (err) {
            console.error("Error fetching top members:", err);
        }
    }

    async function fetchCommunityMembersForModal(commId: number, role: 'member' | 'moderator' | 'all' = 'all') {
        isLoadingMembers = true; membersModalError = null;
        try {
            // Backend GetCommunityMembersRequest takes string role_filter
            let roleFilterForApi = role;
            if (role === 'member') roleFilterForApi = 'member'; // Ensure exact match with backend if specific

            const response = await api.getCommunityMembers(commId, 1, 50, roleFilterForApi); // Fetch up to 50 for now
            console.log(`Fetched community ${role} members:`, response);
            if (role === 'member') membersList = response.members || [];
            else if (role === 'moderator') moderatorsList = response.members || [];
            else { // For 'all', split them as needed
                membersList = (response.members || []).filter(m => m.role === 'member');
                moderatorsList = (response.members || []).filter(m => m.role === 'moderator' || m.role === 'owner'); // Include owner in moderators
            }
        } catch (err) {
            console.error(`Error fetching community ${role}:`, err);
            membersModalError = `Could not load ${role}.`;
        } finally { isLoadingMembers = false; }
    }

    function openMembersModal() {
    if (!communityDetailsResponse?.community) return;
    membersModalTab = 'members';
    membersList = []; moderatorsList = [];
    // fetchCommunityMembersForModal(communityDetailsResponse.community.id, 'member'); // Fetch members
    // fetchCommunityMembersForModal(communityDetailsResponse.community.id, 'moderator'); // Fetch moderators
    fetchCommunityMembersForModal(communityDetailsResponse.community.id, 'all');
    showMembersModal = true;
  }
  function closeMembersModal() { showMembersModal = false; memberSearchQuery = ''; }
  function switchMembersModalTab(tab: 'members' | 'moderators') { membersModalTab = tab; }

  async function handleRoleChange(targetUserId: number, currentRole: string) {
    if (!communityDetailsResponse?.community || !$currentUserStore || roleUpdateLoadingForUser === targetUserId) return;
    const newRole = currentRole === 'member' ? 'moderator' : 'member';
    if (!confirm(`Are you sure you want to ${newRole === 'moderator' ? 'promote' : 'demote'} this user to ${newRole}?`)) return;

    roleUpdateLoadingForUser = targetUserId;
    manageMembersError = null; // Clear previous specific error
    try {
        const payload: UpdateMemberRoleRequestData = { target_user_id: targetUserId, new_role: newRole };
        await api.updateMemberRole(communityDetailsResponse.community.id, payload);

        // Optimistically update local list or refetch
        const updateUserInList = (list: CommunityMemberDetails[]) =>
            list.map(m => m.user.id === targetUserId ? {...m, role: newRole} : m)
              .filter(m => !(newRole === 'member' && currentRole === 'moderator' && m.user.id === targetUserId && membersModalTab === 'moderators') &&
                           !(newRole === 'moderator' && currentRole === 'member' && m.user.id === targetUserId && membersModalTab === 'members'));


        if (currentRole === 'member' && newRole === 'moderator') { // Promoted
            const userToMove = membersList.find(m => m.user.id === targetUserId);
            if (userToMove) {
                membersList = membersList.filter(m => m.user.id !== targetUserId);
                moderatorsList = [{...userToMove, role: newRole}, ...moderatorsList];
            }
        } else if (currentRole === 'moderator' && newRole === 'member') { // Demoted
            const userToMove = moderatorsList.find(m => m.user.id === targetUserId);
            if (userToMove) {
                moderatorsList = moderatorsList.filter(m => m.user.id !== targetUserId);
                membersList = [{...userToMove, role: newRole}, ...membersList];
            }
        }
        // Force reactivity if needed
        membersList = [...membersList];
        moderatorsList = [...moderatorsList];

        alert(`User role updated to ${newRole}.`);
    } catch (err) {
        if (err instanceof ApiError) manageMembersError = `Role update failed: ${err.message}`;
        else manageMembersError = "Could not update role.";
    } finally {
        roleUpdateLoadingForUser = null;
    }
  }

  $: filteredModalMembers = membersModalTab === 'members'
    ? membersList.filter(m => m.user.name.toLowerCase().includes(memberSearchQuery.toLowerCase()) || m.user.username.toLowerCase().includes(memberSearchQuery.toLowerCase()))
    : moderatorsList.filter(m => m.user.name.toLowerCase().includes(memberSearchQuery.toLowerCase()) || m.user.username.toLowerCase().includes(memberSearchQuery.toLowerCase()));
  
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
  
    function switchTab(tab: CommunityDetailTab, forceFetch = false) {
        if (activeTab === tab && !forceFetch) return;
        activeTab = tab;
        communityThreads = []; topPostsThreads = [];
        communityThreadsError = null; currentCommunityThreadsPage = 1; hasMoreCommunityThreads = true; // Reset thread list
        manageMembersError = null; // Clear manage members error

        if (tab === 'manage_members' && communityDetailsResponse?.community && isModeratorOrOwner()) {
            pendingJoinRequests = [];
            fetchPendingRequests(communityDetailsResponse.community.id);
        } else if (tab === 'about') {
            // Data is already in communityDetailsResponse
        } else if (tab === 'top' || tab === 'latest' || tab === 'media') {
            if (communityDetailsResponse?.community) {
                fetchThreadsForCurrentTab(1, true); // Fetch page 1, reset list
                if (tab === 'top') {
                    fetchTopMembers(communityDetailsResponse.community.id);
                }
            }
        }
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

    onMount(() => {
        // Convert string ID from prop to number for API calls
        const numId = parseInt(id, 10);
        if (!isNaN(numId)) {
        communityIdNum = numId;
        fetchCommunityDetails(communityIdNum); // This will trigger fetch for default tab (top)
        } else {
            error = "Invalid Community ID.";
            isLoading = false;
        }
    });

    function handleThreadDeleteFromCommunity(event: CustomEvent<{ id: number }>) {
        communityThreads = communityThreads.filter(t => t.id !== event.detail.id);
    }

    // TODO: Handle infinite scroll for community threads
  
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
                <button class="meta-link" on:click={openMembersModal} aria-label="View members">
                    👥 {comm.member_count} member{comm.member_count !== 1 ? 's' : ''}
                </button>
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
  
              {:else if activeTab === 'top'}
              {#if topMembersForDisplay.length > 0}
                  <div class="top-members-preview">
                    <h4>Top Members</h4>
                    <div class="user-cards-row">
                        {#each topMembersForDisplay as user (user.id)}
                            <div class="user-card-container">
                                <UserCard user={user} />
                            </div>
                        {/each}
                    </div>
                  </div>
              {/if}
              <!-- Display topPostsThreads for 'top' tab -->
              {#if isLoadingCommunityThreads && topPostsThreads.length === 0}
                {#each { length: 3 } as _} <div class="skeleton-thread"> <div class="skeleton-avatar"></div> <div class="skeleton-content"> <div class="skeleton-line short"></div> <div class="skeleton-line long"></div> <div class="skeleton-line medium"></div> </div> </div> {/each}
              {:else if communityThreadsError} <p class="error-text api-error">{communityThreadsError}</p>
              {:else if topPostsThreads.length > 0}
                  {#each topPostsThreads as thread (thread.id)} <ThreadComponent {thread} on:delete={handleThreadDeleteFromCommunity} /> {/each}
              {:else if !isLoadingCommunityThreads} <p class="empty-feed">No top posts in this community yet.</p> {/if}
              <!-- Sentinel for infinite scroll (applies to underlying communityThreads) -->
              <div class="feed-status">
                {#if isLoadingCommunityThreads && communityThreads.length > 0} <p>Loading more...</p> {/if}
                {#if !hasMoreCommunityThreads && communityThreads.length > 0 && !isLoadingCommunityThreads} <p>You've reached the end.</p> {/if}
              </div>
  
          {:else if activeTab === 'latest'}
              <!-- Display communityThreads for 'latest' tab -->
              {#if isLoadingCommunityThreads && communityThreads.length === 0}
                {#each { length: 3 } as _} <div class="skeleton-thread"> <div class="skeleton-avatar"></div> <div class="skeleton-content"> <div class="skeleton-line short"></div> <div class="skeleton-line long"></div> <div class="skeleton-line medium"></div> </div> </div> {/each}
              {:else if communityThreadsError} <p class="error-text api-error">{communityThreadsError}</p>
              {:else if communityThreads.length > 0}
                  {#each communityThreads as thread (thread.id)} <ThreadComponent {thread} on:delete={handleThreadDeleteFromCommunity} /> {/each}
              {:else if !isLoadingCommunityThreads} <p class="empty-feed">No latest threads in this community yet.</p> {/if}
              <div class="feed-status">
                {#if isLoadingCommunityThreads && communityThreads.length > 0} <p>Loading more...</p> {/if}
                {#if !hasMoreCommunityThreads && communityThreads.length > 0 && !isLoadingCommunityThreads} <p>You've reached the end.</p> {/if}
              </div>
  
          {:else if activeTab === 'media'}
              <!-- Display media from communityThreads -->
              {#if isLoadingCommunityThreads && communityThreads.length === 0}
                {#each { length: 3 } as _} <div class="skeleton-thread"> <div class="skeleton-avatar"></div> <div class="skeleton-content"> <div class="skeleton-line short"></div> <div class="skeleton-line long"></div> <div class="skeleton-line medium"></div> </div> </div> {/each}
              {:else if communityThreadsError} <p class="error-text api-error">{communityThreadsError}</p>
              {:else}
                  {@const threadsWithMedia = communityThreads.filter(t => t.media && t.media.length > 0)}
                  {#if threadsWithMedia.length > 0}
                      <div class="explore-media-grid community-media-grid">
                          {#each threadsWithMedia as thread (thread.id)}
                              {#each thread.media ?? [] as mediaItem (mediaItem.id)}
                                  <a href="/thread/{thread.id}" use:link class="media-grid-item">
                                      {#if mediaItem.mime_type.startsWith('image/')}
                                          <img src={mediaItem.public_url} alt="Media from {communityDetailsResponse.community.name}" />
                                      {:else if mediaItem.mime_type.startsWith('video/')}
                                          <div class="video-placeholder-explore">▶️ <span class="video-overlay-text">Video</span></div>
                                      {:else}
                                          <div class="file-placeholder-explore">📄 <span class="file-overlay-text">{mediaItem.mime_type}</span></div>
                                      {/if}
                                  </a>
                              {/each}
                          {/each}
                      </div>
                  {:else if !isLoadingCommunityThreads}
                      <p class="empty-feed">No media posted in this community yet.</p>
                  {/if}
              {/if}
              <div class="feed-status">
                {#if isLoadingCommunityThreads && communityThreads.length > 0} <p>Loading more...</p> {/if}
                {#if !hasMoreCommunityThreads && communityThreads.length > 0 && !isLoadingCommunityThreads} <p>You've reached the end.</p> {/if}
              </div>
          {/if}
      </section>
    {/if}
  </div>

  {#if showMembersModal && communityDetailsResponse?.community}
    <div class="modal-overlay" on:click={closeMembersModal}>
        <div class="modal-content members-modal" on:click|stopPropagation>
            <header class="modal-header-simple">
                <h3>Community Members</h3>
                <button class="close-btn-header" on:click={closeMembersModal}>×</button>
            </header>
            <div class="members-modal-tabs">
                <button class:active={membersModalTab === 'members'} on:click={() => switchMembersModalTab('members')}>
                    Members ({membersList.length})
                </button>
                <button class:active={membersModalTab === 'moderators'} on:click={() => switchMembersModalTab('moderators')}>
                    Moderators ({moderatorsList.length})
                </button>
            </div>
            <div class="members-modal-search">
                <input type="text" placeholder="Search {membersModalTab}..." bind:value={memberSearchQuery} />
            </div>
            <div class="members-modal-list">
                {#if isLoadingMembers} <p>Loading...</p>
                {:else if membersModalError} <p class="error-text">{membersModalError}</p>
                {:else if filteredModalMembers.length > 0}
                    {#each filteredModalMembers as member (member.user.id)}
                        <div class="member-item">
                            <UserCardSummary user={member.user} />
                            <div class="member-role-actions">
                                <span class="member-role">Role: {member.role}</span>
                                {#if isModeratorOrOwner() && member.user.id !== $currentUserStore?.id && member.role !== 'owner'}
                                    {#if roleUpdateLoadingForUser === member.user.id}
                                        <button class="btn btn-secondary small" disabled>...</button>
                                    {:else if member.role === 'member'}
                                        <button class="btn btn-success small" on:click={() => handleRoleChange(member.user.id, 'member')}>Promote to Mod</button>
                                    {:else if member.role === 'moderator'}
                                        <button class="btn btn-warning small" on:click={() => handleRoleChange(member.user.id, 'moderator')}>Demote to Member</button>
                                    {/if}
                                {/if}
                            </div>
                        </div>
                    {/each}
                {:else}
                    <p>No {membersModalTab} found{memberSearchQuery ? ` matching "${memberSearchQuery}"` : ''}.</p>
                {/if}
            </div>
        </div>
    </div>
  {/if}
  
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
          width: 100%; 
          height: 100%;
          background: linear-gradient(45deg, var(--secondary-text-color), var(--border-color));
        }
      }
      
      .info-bar { 
        display: flex; 
        justify-content: space-between; 
        align-items: flex-end; 
        padding: 12px 16px; 
        margin-top: -50px; 
      }

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
          width: 120px; 
          height: 120px; 
          border-radius: 50%;
          border: 4px solid var(--background);
          background-color: var(--secondary-text-color); 
          color: var(--background);
          display: flex; 
          align-items: center; 
          justify-content: center;
          font-size: 3rem; 
          font-weight: bold;
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
          white-space: nowrap;
          
          &:hover { background-color: var(--section-hover-bg); }
    
          &.btn-primary {
            background-color: var(--follow-button-bg);
            color: var(--follow-button-text);
            border-color: var(--follow-button-border);
            &:hover { background-color: var(--follow-button-hover-bg); }
          }
        }
      }
      
      h1 { 
        padding: 0 16px; 
        margin-top: 12px; 
        font-size: 1.8rem; 
        font-weight: bold;
      }
      
      .description { 
        padding: 0 16px; 
        margin-top: 4px; 
        color: var(--text-color); 
        font-size: 15px; 
      }
      
      .meta-info { 
        padding: 8px 16px 16px; 
        display: flex; 
        flex-wrap: wrap;
        gap: 12px; 
        color: var(--secondary-text-color); 
        font-size: 15px; 
      }
    }
    
    .community-feed { padding: 16px; }
    
    .community-main-info {
      padding: 16px;
      
      h1 { 
        font-size: 1.8rem; 
        font-weight: 800; 
        margin: 0 0 4px; 
      }
      
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
      
      .description { margin-bottom: 12px; }
    }
  
  
    .community-tabs {
      display: flex; 
      border-bottom: 1px solid var(--border-color);
      position: sticky; 
      top: 0;
      background-color: rgba(var(--background-rgb), 0.90);
      backdrop-filter: blur(10px);
      z-index: 9;
      overflow-x: auto;
      scrollbar-width: none;
      
      &::-webkit-scrollbar {
        display: none;
      }
      
      button {
        flex: 1;
        padding: 16px;
        min-width: max-content;
        background: none; 
        border: none;
        color: var(--secondary-text-color);
        font-weight: bold; 
        font-size: 15px;
        cursor: pointer; 
        position: relative;
        transition: background-color 0.2s ease;
        
        &:hover { background-color: var(--section-hover-bg); }
        
        &.active {
          color: var(--text-color);
          
          &::after {
            content: ''; 
            position: absolute; 
            bottom: 0; 
            left: 0; 
            right: 0;
            height: 4px; 
            background-color: var(--primary-color); 
            border-radius: 2px;
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
      
      h4 { 
        font-size: 1.1rem; 
        font-weight: 700; 
        margin: 1.5rem 0 0.5rem; 
      }
      
      p { margin-bottom: 0.8rem; }
      
      .creator-link { 
        color: var(--primary-color); 
        text-decoration: none; 
        &:hover {text-decoration: underline;} 
      }
      
      .rules-list {
        list-style-type: decimal;
        padding-left: 20px;
        
        li { margin-bottom: 0.5rem; }
      }
    }

    .manage-members-section {
      h4 { 
        font-size: 1.1rem; 
        font-weight: 700; 
        margin: 1.5rem 0 0.5rem; 
      }
    }
    
    .join-request-list {
      list-style: none; 
      padding: 0; 
      margin: 0;
    }
    
    .join-request-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 0;
      border-bottom: 1px solid var(--border-color);
      
      &:last-child { border-bottom: none; }

      .user-info-join {
        display: flex; 
        align-items: center; 
        gap: 10px;
        overflow: hidden;
        
        .avatar-small-join, .avatar-initials-join {
          width: 36px; 
          height: 36px; 
          border-radius: 50%;
          background-color: var(--secondary-text-color); 
          color: var(--background);
          display: flex; 
          align-items: center; 
          justify-content: center; 
          font-weight: bold;
          flex-shrink: 0;
          
          img { 
            width: 100%; 
            height: 100%; 
            object-fit: cover; 
            border-radius: 50%;
          }
        }
        
        span { 
          font-weight: 500; 
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
      }
      
      .request-actions {
        display: flex; 
        gap: 8px;
        flex-shrink: 0;
        
        .btn.small { 
          padding: 4px 10px; 
          font-size: 0.85rem; 
          font-weight: 600; 
          border-radius: 12px;
        }
        
        .btn-success { 
          background-color: var(--success-color); 
          color: white; 
          border: none; 
          &:hover { opacity: 0.9;}
        }
        
        .btn-danger { 
          background-color: var(--error-color); 
          color: white; 
          border: none; 
          &:hover { opacity: 0.9;}
        }
      }
    }

    .meta-info .meta-link {
      background: none; 
      border: none; 
      padding: 0;
      color: var(--secondary-text-color);
      cursor: pointer; 
      text-decoration: none;
      font-size: inherit;
      
      &:hover { 
        text-decoration: underline; 
        color: var(--primary-color); 
      }
    }

    .modal-overlay { 
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      background-color: rgba(0, 0, 0, 0.6);
      display: flex;
      justify-content: center;
      align-items: flex-start; 
      padding-top: 5vh; 
      z-index: 1000;
    }

    .modal-content.members-modal {
      background: var(--background); 
      color: var(--text-color);
      border-radius: 16px; 
      width: 90%; 
      max-width: 500px;
      max-height: 80vh; 
      display: flex; 
      flex-direction: column;
      box-shadow: 0 5px 20px rgba(0,0,0,0.2);
    }

    .modal-header-simple {
      display: flex;
      align-items: center;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      
      h3 { 
        flex-grow: 1; 
        text-align: left; 
        margin: 0; 
        font-size: 1.2rem; 
        font-weight: bold; 
      }
      
      .close-btn-header {
        background: transparent; 
        border: none; 
        font-size: 1.8rem; 
        cursor: pointer; 
        color: var(--text-color);
        padding: 0 8px;
      }
    }

    .members-modal-tabs {
      display: flex; 
      border-bottom: 1px solid var(--border-color);
      
      button {
        flex: 1;
        padding: 16px;
        background: none; 
        border: none;
        color: var(--secondary-text-color);
        font-weight: bold; 
        font-size: 15px;
        cursor: pointer; 
        position: relative;
        transition: background-color 0.2s ease;
        
        &:hover { background-color: var(--section-hover-bg); }
        
        &.active {
          color: var(--text-color);
          
          &::after {
            content: ''; 
            position: absolute; 
            bottom: 0; 
            left: 0; 
            right: 0;
            height: 4px; 
            background-color: var(--primary-color); 
            border-radius: 2px;
          }
        }
      }
    }
    
    .members-modal-search {
      padding: 12px 16px;
      
      input { 
        width: 100%;
        padding: 8px; 
        border: 1px solid var(--border-color);
        border-radius: 8px; 
        font-size: 15px;
        color: var(--text-color); 
        background-color: var(--background);
        
        &:focus { 
          outline: none; 
          border-color: var(--primary-color); 
        }
      }
    }
    
    .members-modal-list {
      flex-grow: 1; 
      overflow-y: auto; 
      padding: 0 8px 8px 8px;
      
      .member-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px;
        border-bottom: 1px solid var(--border-color);
        
        &:last-child { border-bottom: none; }
        
        :global(.user-card) { 
          padding: 8px 0; 
          border-bottom: none; 
          flex-grow: 1; 
          
          &:hover { background: none;} 
        }
      }
      
      .member-role-actions {
        display: flex; 
        flex-direction: column; 
        align-items: flex-end; 
        gap: 4px;
        flex-shrink: 0; 
        margin-left: 10px;
        
        .btn {
          padding: 4px 10px; 
          font-size: 0.85rem; 
          font-weight: 600; 
          border-radius: 12px;
          
          &.btn-success { 
            background-color: var(--success-color); 
            color: white; 
            border: none; 
            
            &:hover { opacity: 0.9; }
          }
          
          &.btn-warning { 
            background-color: var(--error-color); 
            color: white; 
            border: none; 
            
            &:hover { opacity: 0.9; }
          }
        }
        
        .member-role { 
          font-size: 0.8rem; 
          color: var(--secondary-text-color); 
        }
        
        .btn.small { 
          padding: 3px 8px; 
          font-size: 0.75rem; 
        }
        
        .btn-warning { 
          background-color: orange; 
          color: white; 
          border: none; 
          
          &:hover { opacity: 0.9; }
        }
      }
    }

    .top-members-preview {
      padding: 16px;
      border-bottom: 1px solid var(--border-color);
      margin-bottom: 1px;
      
      h4 { 
        font-size: 1.1rem; 
        font-weight: 700; 
        margin: 0 0 12px 0; 
      }
      
      .user-cards-row {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
        gap: 12px;
        
        :global(.user-card-container) {
          flex: 1; 
          min-width: 150px;
          display: flex;
        }
      }
    }

    .explore-media-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
      gap: 4px;
      
      .media-grid-item {
        aspect-ratio: 1 / 1; 
        background-color: var(--section-bg);
        border-radius: 8px; 
        overflow: hidden; 
        display: flex; 
        align-items: center; 
        justify-content: center;
        
        img { 
          width: 100%; 
          height: 100%; 
          object-fit: cover; 
        }
        
        .video-placeholder-explore { 
          font-size: 2rem; 
          color: var(--secondary-text-color); 
        }
      }
    }

    .feed-status, .empty-feed, .error-text.api-error {
      text-align: center; 
      padding: 20px; 
      color: var(--secondary-text-color); 
      font-size: 14px;
    }

    @keyframes pulse { 
      0% { background-color: var(--section-hover-bg); } 
      50% { background-color: var(--border-color); } 
      100% { background-color: var(--section-hover-bg); } 
    }
    
    .skeleton-thread { 
      display: flex; 
      padding: 12px 16px; 
      border-bottom: 1px solid var(--border-color); 
      gap: 12px; 
    }
    
    .skeleton-avatar { 
      width: 40px; 
      height: 40px; 
      border-radius: 50%; 
      background-color: var(--section-hover-bg); 
      animation: pulse 1.5s infinite ease-in-out; 
      flex-shrink: 0; 
    }
    
    .skeleton-content { 
      flex-grow: 1; 
      display: flex; 
      flex-direction: column; 
      gap: 8px; 
      padding-top: 4px; 
    }
    
    .skeleton-line { 
      height: 10px; 
      border-radius: 4px; 
      background-color: var(--section-hover-bg); 
      animation: pulse 1.5s infinite ease-in-out;
      
      &.short { width: 30%; } 
      &.medium { width: 60%; } 
      &.long { width: 90%; }
    }
    
    /* Responsive styles */
    @media (max-width: 1024px) {
      .community-detail-header .actions {
        margin-top: 50px;
      }
      
      .top-members-preview .user-cards-row {
        grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
      }
    }
    
    @media (max-width: 900px) {
      .community-detail-header {
        .banner-container {
          height: 150px;
        }
        
        .icon-container {
          .community-icon-large,
          .icon-placeholder-large {
            width: 100px;
            height: 100px;
            font-size: 2.5rem;
          }
        }
        
        h1 {
          font-size: 1.6rem;
        }
      }
      
      .community-tabs button {
        padding: 14px 12px;
        font-size: 14px;
      }
      
      .explore-media-grid {
        grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
      }
    }
    
    @media (max-width: 768px) {
      .community-detail-header {
        .info-bar {
          margin-top: -40px;
        }
        
        .actions {
          margin-top: 30px;
          
          .btn {
            padding: 6px 12px;
            font-size: 13px;
          }
        }
        
        h1 {
          font-size: 1.4rem;
        }
        
        .description {
          font-size: 14px;
        }
        
        .meta-info {
          font-size: 14px;
          padding: 8px 16px 12px;
        }
      }
      
      .community-tab-content {
        padding: 12px;
      }
      
      .about-section,
      .manage-members-section {
        h4 {
          font-size: 1rem;
        }
      }
      
      .top-members-preview .user-cards-row {
        grid-template-columns: 1fr;
      }
    }
    
    @media (max-width: 576px) {
      .community-detail-header {
        .banner-container {
          height: 120px;
        }
        
        .info-bar {
          flex-direction: column;
          align-items: flex-start;
          margin-top: -30px;
        }
        
        .icon-container {
          .community-icon-large,
          .icon-placeholder-large {
            width: 80px;
            height: 80px;
            font-size: 2rem;
          }
        }
        
        .actions {
          margin-top: 12px;
          margin-bottom: 8px;
          width: 100%;
          
          .btn {
            width: 100%;
            text-align: center;
          }
        }
        
        .community-main-info {
          padding: 12px;
        }
        
        h1 {
          font-size: 1.3rem;
          padding: 0;
          margin-top: 8px;
        }
        
        .description {
          padding: 0;
        }
        
        .meta-info {
          padding: 8px 0 12px;
        }
      }
      
      .community-tabs {
        button {
          padding: 12px 8px;
          font-size: 14px;
        }
      }
      
      .community-tab-content {
        padding: 10px;
      }
      
      .join-request-item {
        flex-direction: column;
        align-items: flex-start;
        gap: 10px;
        
        .user-info-join {
          width: 100%;
        }
        
        .request-actions {
          width: 100%;
          
          .btn.small {
            flex: 1;
            text-align: center;
          }
        }
      }
      
      .modal-content.members-modal {
        width: 95%;
        max-height: 85vh;
      }
      
      .members-modal-tabs button {
        padding: 12px 8px;
        font-size: 14px;
      }
      
      .explore-media-grid {
        grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
        gap: 3px;
      }
    }
    
    @media (max-width: 400px) {
      .community-detail-header {
        .icon-container {
          .community-icon-large,
          .icon-placeholder-large {
            width: 60px;
            height: 60px;
            font-size: 1.5rem;
            border-width: 3px;
          }
        }
        
        h1 {
          font-size: 1.2rem;
        }
      }
      
      .community-tabs {
        button {
          padding: 10px 5px;
          font-size: 12px;
        }
      }
      
      .member-item {
        flex-direction: column;
        align-items: flex-start;
        
        .member-role-actions {
          margin-left: 0;
          width: 100%;
          margin-top: 8px;
          
          .btn {
            width: 100%;
          }
        }
      }
      
      .explore-media-grid {
        grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
        gap: 2px;
      }
      
      .members-modal-search input {
        padding: 6px;
        font-size: 14px;
      }
    }
  </style>