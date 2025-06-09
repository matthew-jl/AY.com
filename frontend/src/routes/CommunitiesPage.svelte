<script lang="ts">
    import { link, navigate } from 'svelte-routing';
    import { onMount } from 'svelte';
    import { api, type CommunityListItem, type GetUserJoinRequestsApiResponse, type JoinRequestItem } from '../lib/api';
    import CommunityCard from '../components/CommunityCard.svelte';
    import { user as currentUserStore } from '../stores/userStore';
  
    type CommunityTab = 'joined' | 'pending' | 'discover';

    let activeTab: CommunityTab = 'discover'; // Default to discover

    let joinedCommunities: CommunityListItem[] = [];
    let pendingJoinRequests: JoinRequestItem[] = []; // Store full request details
    let discoverCommunities: CommunityListItem[] = [];

    let isLoadingJoined = false;
    let isLoadingPending = false;
    let isLoadingDiscover = false;
    let error: string | null = null;

    let discoverSearchQuery = '';
    let selectedCategories: string[] = [];
    const availableCategories = ['gaming', 'tech', 'food', 'sports', 'music', 'art', 'science'];
    let showCategoryFilter = false;
    let discoverDebounceTimer: number | undefined;

    // TODO: Pagination state for each list
  
    async function fetchDataForTab(tab: CommunityTab, reset = true) {
      if (tab === 'joined' && $currentUserStore) {
          isLoadingJoined = true; if (reset) joinedCommunities = [];
          try {
              const response = await api.listCommunities('JOINED_BY_USER', $currentUserStore.id, 1, 20); // Add pagination later
              joinedCommunities = reset ? response.communities || [] : [...joinedCommunities, ...(response.communities || [])];
          } catch (e) { error = "Could not load your communities."; console.error(e); }
          finally { isLoadingJoined = false; }
      } else if (tab === 'pending' && $currentUserStore) {
          isLoadingPending = true; if (reset) pendingJoinRequests = [];
          try {
              const response = await api.getUserJoinRequests(1, 20); // Add pagination later
              pendingJoinRequests = reset ? response.requests || [] : [...pendingJoinRequests, ...(response.requests || [])];
              // TODO: Hydrate community names for pendingJoinRequests if not done by backend
          } catch (e) { error = "Could not load pending requests."; console.error(e); }
          finally { isLoadingPending = false; }
      } else if (tab === 'discover') {
          isLoadingDiscover = true; if (reset) discoverCommunities = [];
          try {
              const response = await api.listCommunities(
                  'ALL_PUBLIC',
                  $currentUserStore?.id, // Pass requester ID for is_joined status
                  1, 20, // Add pagination later
                  discoverSearchQuery.trim() || undefined, // Pass undefined if empty
                  selectedCategories.length > 0 ? selectedCategories : undefined
              );
              discoverCommunities = reset ? response.communities || [] : [...discoverCommunities, ...(response.communities || [])];
          } catch (e) { error = "Could not load communities."; console.error(e); }
          finally { isLoadingDiscover = false; }
      }
    }

    function switchTab(tab: CommunityTab) {
      if (activeTab === tab) return;
      activeTab = tab;
      error = null;
      fetchDataForTab(tab, true);
    }

    onMount(() => {
      activeTab = $currentUserStore ? 'joined' : 'discover';
      fetchDataForTab(activeTab, true);
    });

    function handleJoinRequestSent() {
      if (activeTab === 'discover') fetchDataForTab('discover', true);
      if ($currentUserStore) fetchDataForTab('pending', true);
    }

    function handleDiscoverSearchInput() {
      clearTimeout(discoverDebounceTimer);
      discoverDebounceTimer = window.setTimeout(() => {
          fetchDataForTab('discover', true);
      }, 500);
  }

  function toggleCategory(category: string) {
      const index = selectedCategories.indexOf(category);
      if (index > -1) {
          selectedCategories = selectedCategories.filter(c => c !== category);
      } else {
          selectedCategories = [...selectedCategories, category];
      }
      fetchDataForTab('discover', true); 
  }

  </script>
  
  <div class="communities-page-container">
    <header class="page-header">
      <div class="header-content">
        <h2>Communities</h2>
        <!-- TODO: Add search/filter for communities later -->
      </div>
      <button class="btn btn-primary create-community-btn" on:click={() => navigate('/communities/create')}>
        Create Community
      </button>
    </header>
  
    <nav class="profile-tabs communities-tabs">
      {#if $currentUserStore}
          <button class:active={activeTab === 'joined'} on:click={() => switchTab('joined')}>Your Communities</button>
          <button class:active={activeTab === 'pending'} on:click={() => switchTab('pending')}>Pending Requests</button>
      {/if}
      <button class:active={activeTab === 'discover'} on:click={() => switchTab('discover')}>Discover</button>
    </nav>

    <div class="communities-content">
      {#if error} <p class="error-text api-error">{error}</p> {/if}

      <!-- Discover Tab Specific Filters -->
      {#if activeTab === 'discover'}
          <div class="discover-filters">
              <input
                  type="text"
                  placeholder="Search communities..."
                  bind:value={discoverSearchQuery}
                  on:input={handleDiscoverSearchInput}
                  class="discover-search-input"
              />
              <div class="category-filter-toggle">
                  <button class="btn-link" on:click={() => showCategoryFilter = !showCategoryFilter}>
                      Filter by Category {selectedCategories.length > 0 ? `(${selectedCategories.length})` : ''} {showCategoryFilter ? '▲' : '▼'}
                  </button>
                  {#if showCategoryFilter}
                  <div class="category-dropdown">
                      {#each availableCategories as category (category)}
                          <label>
                              <input
                                  type="checkbox"
                                  value={category}
                                  checked={selectedCategories.includes(category)}
                                  on:change={() => toggleCategory(category)}
                              />
                              #{category}
                          </label>
                      {/each}
                  </div>
                  {/if}
              </div>
          </div>
      {/if}

      <!-- Joined Communities Tab -->
      {#if activeTab === 'joined'}
        <section class="community-list-section">
          {#if isLoadingJoined} <p>Loading your communities...</p>
          {:else if joinedCommunities.length > 0}
            <div class="community-grid">
              {#each joinedCommunities as community (community.id)}
                <CommunityCard {community} />
              {/each}
            </div>
          {:else} <p class="empty-list">You haven't joined or created any communities yet.</p> {/if}
        </section>
      {/if}

      <!-- Pending Join Requests Tab -->
      {#if activeTab === 'pending'}
          <section class="community-list-section">
              {#if isLoadingPending} <p>Loading your pending requests...</p>
              {:else if pendingJoinRequests.length > 0}
                  <ul class="request-list">
                  {#each pendingJoinRequests as request (request.request_id)}
                      <li class="request-item">
                          <span>Request to join <strong>{request.community_name || `Community ID ${request.community_id}`}</strong> is <em>{request.status}</em>.</span>
                          <!-- TODO: Add link to community, cancel request option -->
                      </li>
                  {/each}
                  </ul>
              {:else} <p class="empty-list">You have no pending join requests.</p> {/if}
          </section>
      {/if}

      <!-- Discover Communities Tab -->
      {#if activeTab === 'discover'}
        <section class="community-list-section">
          {#if isLoadingDiscover} <p>Loading communities...</p>
          {:else if discoverCommunities.length > 0}
            <div class="community-grid">
              {#each discoverCommunities as community (community.id)}
                <CommunityCard {community} onJoinRequested={handleJoinRequestSent} />
              {/each}
            </div>
          {:else if discoverSearchQuery || selectedCategories.length > 0}
            <p class="empty-list">No communities found matching your criteria.</p>
          {:else} <p class="empty-list">No communities to discover right now. Why not create one?</p> {/if}
        </section>
      {/if}
    </div>
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .communities-page-container {
      width: 100%;
      // padding: 16px;
    }
  
    .page-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      position: sticky;
      top: 0;
      background-color: rgba(var(--background-rgb), 0.85);
      backdrop-filter: blur(12px);
      z-index: 10;
  
      .header-content h2 {
        font-size: 20px;
        font-weight: 800;
        margin: 0;
      }
    }
  
    .create-community-btn {
      padding: 8px 16px;
      font-size: 14px;
      background-color: var(--primary-color);
      color: var(--primary-button-text);
      border: none;
      border-radius: 9999px;
      font-weight: bold;
      cursor: pointer;
      &:hover {
          background-color: var(--primary-color-hover);
      }
    }
  
    .communities-content {
      padding: 16px;
    }
  
    .community-list-section {
      margin-bottom: 2rem;
      h3 {
        font-size: 18px;
        font-weight: 700;
        margin-bottom: 1rem;
        padding-bottom: 0.5rem;
        border-bottom: 1px solid var(--border-color);
      }
    }
    .empty-list, .error-text {
        text-align: center;
        padding: 20px;
        color: var(--secondary-text-color);
    }

    .communities-tabs {
      display: flex; border-bottom: 1px solid var(--border-color);
      background-color: var(--background);
      position: sticky;
      top: 57px; /* Adjust based on main header height */
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
    .community-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 16px;
    }
    .request-list { list-style: none; padding: 0; }
    .request-item {
        padding: 10px; border-bottom: 1px solid var(--border-color);
        font-size: 0.95rem; color: var(--text-color);
        &:last-child { border-bottom: none; }
        strong { font-weight: 600; }
        em { color: var(--secondary-text-color); }
    }

    .discover-filters {
      display: flex;
      gap: 1rem;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      margin-bottom: 1rem;
      align-items: center;

      .discover-search-input {
        flex-grow: 1;
        padding: 8px 12px;
        border-radius: 16px;
        border: 1px solid var(--border-color);
        background-color: var(--input-bg);
        color: var(--text-color);
        font-size: 15px;
        &:focus { outline: none; border-color: var(--primary-color); }
      }
    }
    .category-filter-toggle {
        position: relative;
        .btn-link {
            background: none;
            border: none;
            color: var(--primary-color);
            font-weight: bold;
            cursor: pointer;
            padding: 4px 8px;
            border-radius: 6px;
            font-size: 15px;
            transition: background 0.15s;
            &:hover, &:focus {
              background: var(--section-hover-bg);
              outline: none;
            }
        }
    }
    .category-dropdown {
      position: absolute;
      top: 100%;
      right: 0;
      background-color: var(--background);
      border: 1px solid var(--border-color);
      border-radius: 8px;
      box-shadow: 0 3px 10px rgba(0,0,0,0.1);
      padding: 8px;
      z-index: 20;
      min-width: 200px;
      max-height: 250px;
      overflow-y: auto;

      label {
        display: block;
        padding: 6px 8px;
        cursor: pointer;
        border-radius: 4px;
        font-size: 14px;
        input[type="checkbox"] { margin-right: 8px; accent-color: var(--primary-color); }
        &:hover { background-color: var(--section-hover-bg); }
      }
    }
  </style>