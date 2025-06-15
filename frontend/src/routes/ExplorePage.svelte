<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { api, ApiError, type ThreadData, type UserProfileBasic, type FeedResponse, type TrendingHashtagItem, type CommunityListItem } from '../lib/api';
    import { currentPathname } from '../stores/locationStore';
    import { navigate, link } from 'svelte-routing';
    import ThreadComponent from '../components/ThreadComponent.svelte';
    import UserCard from '../components/UserCard.svelte';
  import CommunityCard from '../components/CommunityCard.svelte';
  
    type SearchTab = 'top' | 'latest' | 'people' | 'media' | 'communities';
    const DEFAULT_TAB: SearchTab = 'top';
  
    // --- Search State ---
    let searchQuery = ''; // Bound to search input
    let debouncedSearchQuery = ''; // For API calls after debounce
    let searchInputEl: HTMLInputElement;
    let recentSearches: string[] = [];
    let activeTab: SearchTab = DEFAULT_TAB;
  
    // --- Results State ---
    let topUsers: UserProfileBasic[] = [];
    let topThreads: ThreadData[] = [];
    let latestThreads: ThreadData[] = [];
    let peopleResults: UserProfileBasic[] = [];
    let mediaThreads: ThreadData[] = [];
    let communityResults: CommunityListItem[] = [];
  
    let isLoading = false;
    let currentError: string | null = null;
    let debounceTimer: number | undefined = undefined;
  
    // --- Trending Hashtags ---
    let trendingHashtags: TrendingHashtagItem[] = [];
    let isLoadingTrending = true;

    let liveUserSuggestions: UserProfileBasic[] = [];
    let showUserSuggestions = false;
    let liveUserFetchTimer: number | undefined = undefined;
    let searchBarFocused = false;

    let communitiesCurrentPage = 1;
    let communitiesHasMore = true;
    let isLoadingMoreCommunities = false;
  
    // --- Lifecycle & Initial Load ---
    onMount(() => {
      loadRecentSearches();
      fetchTrendingHashtags();
  
      // Check for initial query from URL (e.g., from hashtag click)
      const urlParams = new URLSearchParams(window.location.search);
      const initialQuery = urlParams.get('q');
      if (initialQuery) {
        searchQuery = initialQuery;
        debouncedSearchQuery = initialQuery;
        performSearch(initialQuery, activeTab);
      }
  
      return () => {
        clearTimeout(debounceTimer);
      };
    });
  
    // --- Recent Searches ---
    function loadRecentSearches() {
      const stored = localStorage.getItem('recentSearches_AY');
      if (stored) {
        recentSearches = JSON.parse(stored);
      }
    }
    function saveRecentSearches() {
      localStorage.setItem('recentSearches_AY', JSON.stringify(recentSearches.slice(0, 5))); // Save top 5
    }
    function addRecentSearch(term: string) {
      if (!term.trim() || recentSearches.includes(term.trim())) return;
      recentSearches = [term.trim(), ...recentSearches.filter(s => s !== term.trim())];
      saveRecentSearches();
    }
    function clearRecentSearches() {
      console.log("Clearing recent searches");
      recentSearches = [];
      saveRecentSearches();
    }
    function searchFromRecent(term: string) {
        searchQuery = term;
        handleSearchInput(); // Trigger debounce and search
        searchBarFocused = false;
    }
  
    // --- Debounce & Search Logic ---
    function handleSearchInput() {
      clearTimeout(debounceTimer);
      currentError = null;
      fetchLiveUserSuggestions(searchQuery);

      debounceTimer = window.setTimeout(() => {
        debouncedSearchQuery = searchQuery.trim();
        if (debouncedSearchQuery) {
          addRecentSearch(debouncedSearchQuery);
          performSearch(debouncedSearchQuery, activeTab);
          navigate(`/explore?q=${encodeURIComponent(debouncedSearchQuery)}`, { replace: true });
        } else {
          clearResults();
          navigate('/explore', { replace: true });
        }
      }, 900);
    }
  
    function clearSearch() {
        searchQuery = '';
        debouncedSearchQuery = '';
        clearResults();
        navigate('/explore', {replace: true});
        if(searchInputEl) searchInputEl.focus();
    }
  
    function clearResults() {
        topUsers = []; topThreads = []; latestThreads = []; 
        peopleResults = []; mediaThreads = []; communityResults = [];
        communitiesCurrentPage = 1; communitiesHasMore = true;
        currentError = null;
    }
  
    async function performSearch(query: string, tab: SearchTab, page=1, append=false) {
      if (!query) return;
      isLoading = true;
      currentError = null;
      clearResults();
  
      console.log(`Performing search for "${query}" in tab "${tab}"`);
      try {
        if (tab === 'top' || tab === 'people') {
          const userResp = await api.searchUsers(query, 1, tab === 'top' ? 3 : 10);
          if(tab === 'people') peopleResults = userResp.users || []; else topUsers = userResp.users || [];
        }
        if (tab === 'top' || tab === 'latest' || tab === 'media') {
          const threadResp = await api.searchThreads(query, 1, 10);
          console.log("Thread search response:", threadResp.threads);
          if(tab === 'latest') {
            latestThreads = (threadResp.threads || []).slice().sort(
              (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
            );
          }
          else if(tab === 'media') {
            mediaThreads = (threadResp.threads || []).filter(t => t.media_ids && t.media_ids.length > 0);
          }
          else if(tab === 'top') {
            topThreads = (threadResp.threads || []).slice().sort(
              (a, b) => (b.like_count ?? 0) - (a.like_count ?? 0)
            );
          }
        }
        if (tab === 'communities') {
          const commResp = await api.listCommunities(
              'ALL_PUBLIC',
              null,
              page,
              25,
              query
              // No categories
          );
          communityResults = append ? [...communityResults, ...(commResp.communities || [])] : (commResp.communities || []);
          communitiesHasMore = commResp.has_more;
          communitiesCurrentPage = page;
        }
      } catch (err) {
        console.error("Search error:", err);
        currentError = "Search failed. Please try again.";
      } finally {
        isLoading = false;
      }
    }
  
    async function fetchTrendingHashtags() {
      isLoadingTrending = true;
      try {
        const response = await api.getTrendingHashtags(10);
        trendingHashtags = response.trending_hashtags || [];
      } catch (err) {
        console.error("Error fetching trending hashtags:", err);
      } finally {
        isLoadingTrending = false;
      }
    }
  
    function switchTab(newTab: SearchTab) {
      if (activeTab === newTab) return;
      activeTab = newTab;
      if (debouncedSearchQuery) {
        performSearch(debouncedSearchQuery, activeTab);
      }
    }
  
    function handleThreadDelete(event: CustomEvent<{ id: number }>) {
      const id = event.detail.id;
      topThreads = topThreads.filter(t => t.id !== id);
      latestThreads = latestThreads.filter(t => t.id !== id);
      mediaThreads = mediaThreads.filter(t => t.id !== id);
    }

    async function fetchLiveUserSuggestions(query: string) {
      if (!query.trim()) {
        liveUserSuggestions = [];
        showUserSuggestions = false;
        return;
      }
      if (liveUserFetchTimer) clearTimeout(liveUserFetchTimer);
      liveUserFetchTimer = window.setTimeout(async () => {
        try {
          const resp = await api.searchUsers(query, 1, 5);
          liveUserSuggestions = resp.users || [];
          showUserSuggestions = liveUserSuggestions.length > 0;
        } catch {
          liveUserSuggestions = [];
          showUserSuggestions = false;
        }
      }, 100);
    }

    function handleSuggestionClick(user: UserProfileBasic) {
      searchQuery = user.username;
      debouncedSearchQuery = user.username;
      showUserSuggestions = false;
      performSearch(user.username, 'people');
      navigate(`/explore?q=${encodeURIComponent(user.username)}`, { replace: true });
    }

    function handleSearchBarBlur() {
      searchBarFocused = false;
      setTimeout(() => { showUserSuggestions = false; }, 150);
    }

    function handleSearchBarFocus() {
      searchBarFocused = true;
      if (liveUserSuggestions.length > 0) showUserSuggestions = true;
    }

    function handleJoinRequestForCommunityCard() {
      // Refetch communities to update join status
      if (activeTab === 'communities' && debouncedSearchQuery) {
          performSearch(debouncedSearchQuery, 'communities', 1, false); 
      }
  }
  
  </script>
  
  <div class="explore-page">
    <header class="explore-header">
      <div class="search-bar-container">
          <svg viewBox="0 0 24 24" class="search-icon"><g><path d="M10.25 3.75c-3.59 0-6.5 2.91-6.5 6.5s2.91 6.5 6.5 6.5c1.795 0 3.42-.726 4.596-1.904 1.178-1.177 1.904-2.801 1.904-4.596 0-3.59-2.91-6.5-6.5-6.5zm-8.5 6.5c0-4.694 3.806-8.5 8.5-8.5s8.5 3.806 8.5 8.5c0 1.986-.682 3.815-1.824 5.262l4.781 4.781-1.414 1.414-4.781-4.781c-1.447 1.142-3.276 1.824-5.262 1.824-4.694 0-8.5-3.806-8.5-8.5z"></path></g></svg>
          <input
              type="text"
              placeholder="Search AY.com"
              bind:this={searchInputEl}
              bind:value={searchQuery}
              on:input={handleSearchInput}
              on:focus={handleSearchBarFocus}
              on:blur={handleSearchBarBlur}
              autocomplete="off"
          />
          {#if searchQuery}
              <button class="clear-search-btn" on:click={clearSearch}>×</button>
          {/if}

          {#if searchBarFocused && !searchQuery && recentSearches.length > 0}
            <div class="recent-searches-dropdown">
              <div class="dropdown-header">
                  <span>Recent</span>
                  <button class="clear-btn-sidebar" on:mousedown={clearRecentSearches}>Clear all</button>
              </div>
                <ul>
                    {#each recentSearches.slice(0,3) as term (term)}
                        <li>
                          <button class="recent-item-btn" on:mousedown={() => searchFromRecent(term)}>{term}</button>
                        </li>
                    {/each}
                </ul>
            </div>
          {/if}

          {#if showUserSuggestions && liveUserSuggestions.length > 0}
            <div class="floating-user-suggestions">
              {#each liveUserSuggestions as user (user.id)}
                <div class="suggestion-row" on:mousedown={() => handleSuggestionClick(user)}>
                  <UserCard {user} showFollowButton={false} />
                </div>
              {/each}
            </div>
          {/if}
      </div>
      <!-- TODO: Add Filters button/modal -->
    </header>
  
    {#if debouncedSearchQuery}
      <!-- Search Results View -->
      <nav class="profile-tabs explore-tabs">
          <button class:active={activeTab === 'top'} on:click={() => switchTab('top')}>Top</button>
          <button class:active={activeTab === 'latest'} on:click={() => switchTab('latest')}>Latest</button>
          <button class:active={activeTab === 'people'} on:click={() => switchTab('people')}>People</button>
          <button class:active={activeTab === 'media'} on:click={() => switchTab('media')}>Media</button>
          <button class:active={activeTab === 'communities'} on:click={() => switchTab('communities')}>Communities</button>
      </nav>
  
      <div class="search-results-content">
          {#if isLoading}
              <p>Searching...</p> <!-- TODO: Skeleton loaders for results -->
          {:else if currentError}
              <p class="error-text api-error">{currentError}</p>
          {:else}
              <!-- Top Tab -->
              {#if activeTab === 'top'}
                  {#if topUsers.length > 0}
                    <div class="top-people-section">
                      <h4>People</h4>
                      <div class="user-results-list compact">
                          {#each topUsers as user (user.id)}
                             <UserCard {user} showFollowButton={true} />
                          {/each}
                      </div>
                      <button class="view-all-btn" on:click={() => switchTab('people')}>View all</button>
                    </div>
                  {/if}
                  {#if topThreads.length > 0}
                      <!-- <h4>Threads</h4> -->
                      {#each topThreads as thread (thread.id)}
                          <ThreadComponent {thread} on:delete={handleThreadDelete} />
                      {/each}
                  {/if}
                  {#if topUsers.length === 0 && topThreads.length === 0}
                      <p>No top results found for "{debouncedSearchQuery}".</p>
                  {/if}
              {/if}
  
              <!-- Latest Tab -->
              {#if activeTab === 'latest'}
                  {#if latestThreads.length > 0}
                      {#each latestThreads as thread (thread.id)}
                          <ThreadComponent {thread} on:delete={handleThreadDelete} />
                      {/each}
                  {:else}
                      <p>No recent threads found for "{debouncedSearchQuery}".</p>
                  {/if}
              {/if}
  
              <!-- People Tab -->
              {#if activeTab === 'people'}
                  {#if peopleResults.length > 0}
                       <div class="user-results-list full">
                          {#each peopleResults as user (user.id)}
                              <UserCard {user} showFollowButton={true} />
                          {/each}
                      </div>
                  {:else}
                      <p>No people found matching "{debouncedSearchQuery}".</p>
                  {/if}
                   <!-- TODO: Add pagination for People -->
              {/if}
  
              <!-- Media Tab -->
              {#if activeTab === 'media'}
                  {#if mediaThreads.length > 0}
                       <div class="explore-media-grid">
                          {#each mediaThreads as thread (thread.id)}
                              {#if thread.media && thread.media.length > 0}
                                  {#each thread.media as mediaItem (mediaItem.id)}
                                      <a href="/thread/{thread.id}" use:link class="media-grid-item">
                                          {#if mediaItem.mime_type.startsWith('image/')}
                                              <img src={mediaItem.public_url} alt="Media from thread {thread.id}" />
                                          {:else if mediaItem.mime_type.startsWith('video/')}
                                              <div class="video-placeholder-explore">▶️</div>
                                          {/if}
                                      </a>
                                  {/each}
                              {/if}
                          {/each}
                      </div>
                  {:else}
                      <p>No media found for "{debouncedSearchQuery}".</p>
                  {/if}
                   <!-- TODO: Add infinite scroll for Media -->
              {/if}

              {#if activeTab === 'communities'}
                   {#if isLoading && communityResults.length === 0}
                       <p>Searching communities...</p> <!-- TODO: Skeleton -->
                   {:else if communityResults.length > 0}
                       <div class="community-results-grid">
                           {#each communityResults as community (community.id)}
                               <CommunityCard {community} onJoinRequested={handleJoinRequestForCommunityCard} />
                           {/each}
                       </div>
                       <!-- TODO: Add sentinel and "Load More" button for community pagination -->
                       {#if isLoadingMoreCommunities} <p>Loading more communities...</p> {/if}
                       {#if !communitiesHasMore && communityResults.length > 0} <p>No more communities found.</p> {/if}
                   {:else if !isLoading}
                       <p>No communities found matching "{debouncedSearchQuery}".</p>
                   {/if}
              {/if}
          {/if}
      </div>
  
    {:else}
      <!-- Default View: Recent Searches and Trending Hashtags -->
  
      <section class="trending-hashtags">
          <h3>Trends for you</h3>
          {#if isLoadingTrending}
              <p>Loading trends...</p>
          {:else if trendingHashtags.length > 0}
              <ul>
                  {#each trendingHashtags as tag (tag)}
                      <li>
                          <a href="/explore?q=%23{tag.tag}" use:link class="trend-link">
                              <span class="trend-category">Trending</span>
                              <span class="trend-tag">#{tag.tag}</span>
                              <span class="trend-posts">{tag.count} posts</span>
                          </a>
                      </li>
                  {/each}
              </ul>
          {:else}
              <p>No trends right now.</p>
          {/if}
      </section>
    {/if}
  
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .explore-page {
      width: 100%;
    }
  
    .explore-header {
      position: sticky;
      top: 0;
      background-color: rgba(var(--background-rgb), 0.85);
      backdrop-filter: blur(12px);
      z-index: 10;
      padding: 8px 16px;
      border-bottom: 1px solid var(--border-color);
    }
  
    .search-bar-container {
      position: relative;
      display: flex;
      align-items: center;
      .search-icon {
        position: absolute; left: 12px; top: 50%; transform: translateY(-50%);
        width: 18px; height: 18px; fill: var(--secondary-text-color);
      }
      input[type="text"] {
        width: 100%;
        padding: 10px 12px 10px 40px;
        border-radius: 9999px;
        border: 1px solid transparent;
        background-color: var(--search-bg);
        color: var(--text-color);
        font-size: 15px;
        &:focus {
          outline: none; border-color: var(--primary-color);
          background-color: var(--background);
          box-shadow: 0 0 0 1px var(--primary-color);
        }
      }
      .clear-search-btn {
          position: absolute; right: 10px; top: 50%; transform: translateY(-50%);
          background: var(--secondary-text-color); color: var(--background);
          border: none; border-radius: 50%; width: 20px; height: 20px;
          font-size: 14px; line-height: 18px; text-align: center; cursor: pointer;
          display: flex; align-items: center; justify-content: center;
          &:hover { background: var(--text-color); }
      }
      .floating-user-suggestions {
        position: absolute;
        top: 110%;
        left: 0;
        width: 100%;
        background: var(--background);
        border: 1px solid var(--border-color);
        border-radius: 12px;
        box-shadow: 0 4px 24px rgba(0,0,0,0.10);
        z-index: 100;
        padding: 4px 0;
        max-height: 320px;
        overflow-y: auto;
        .suggestion-row {
          padding: 0 8px;
          cursor: pointer;
          &:hover {
            background: var(--section-hover-bg);
          }
          .user-card {
            border-bottom: none;
            padding: 8px 0;
          }
        }
      }
    }
  
    .explore-tabs {
        display: flex;
      border-bottom: 1px solid var(--border-color);
      position: sticky;
      top: 0px;
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
  
    .search-results-content {
      padding: 16px;
      h4 { font-size: 1.1rem; font-weight: bold; margin: 1.5rem 0 0.5rem; }
    }
  
    .recent-searches, .trending-hashtags {
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      h3 { font-size: 20px; font-weight: 800; margin-bottom: 12px; }
      ul { list-style: none; padding: 0; margin: 0; }
      li { margin-bottom: 8px; }
    }
    .section-header { display: flex; justify-content: space-between; align-items: center; }
    .clear-btn, .recent-term-btn {
        background: none; border: none; color: var(--primary-color);
        cursor: pointer; font-size: 14px; padding: 4px 0;
        &:hover { text-decoration: underline; }
    }
    .recent-term-btn { color: var(--text-color); font-weight: 500; }
  
    .trend-link {
        display: block; padding: 8px 0; text-decoration: none; color: inherit;
        &:hover { background-color: var(--section-hover-bg); }
        .trend-category { display: block; font-size: 13px; color: var(--secondary-text-color); }
        .trend-tag { display: block; font-size: 15px; font-weight: bold; color: var(--text-color); margin: 2px 0; }
        .trend-posts { display: block; font-size: 13px; color: var(--secondary-text-color); }
    }
  
    .user-results-list {
        display: flex; flex-direction: column; gap: 0;
         &.full .simple-user-card { border-bottom: 1px solid var(--border-color); &:last-child { border-bottom: none;}}
    }
    .simple-user-card {
        display: flex; align-items: center; padding: 12px 0; gap: 10px;
        &.large {
           align-items: flex-start;
           .avatar-placeholder-small { width: 48px; height: 48px; font-size: 1.5rem; }
           .user-info { flex-grow: 1; }
           .follow-btn-explore {
               padding: 6px 16px; font-size: 14px; font-weight: bold;
               border-radius: 9999px; cursor: pointer;
               background-color: var(--follow-button-bg);
               color: var(--follow-button-text);
               border: 1px solid var(--follow-button-border);
               &:hover { background-color: var(--follow-button-hover-bg); }
           }
        }
        .user-bio-snippet { font-size: 14px; color: var(--secondary-text-color); margin-top: 2px; }
    }
    .avatar-placeholder-small {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        background-color: var(--secondary-text-color);
        color: var(--background);
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: bold;
        flex-shrink: 0;
    }
  
  
    .explore-media-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
        gap: 4px;
        .media-grid-item {
            aspect-ratio: 1 / 1; background-color: var(--section-bg);
            border-radius: 8px; overflow: hidden; display: flex; align-items: center; justify-content: center;
            img { width: 100%; height: 100%; object-fit: cover; }
            .video-placeholder-explore { font-size: 2rem; color: var(--secondary-text-color); }
        }
    }
  
    .error-text { color: var(--error-color); font-size: 0.85rem; margin-top: 4px; }
   .api-error { margin-top: 1rem; text-align: center; font-weight: bold; }

   .top-people-section {
      border-bottom: 1px solid var(--border-color);
      margin-bottom: 16px;
      padding-bottom: 12px;
      h4 { margin-top: 0; }
  }
  .user-results-list.compact .user-card {
      border-bottom: none;
  }
  .view-all-btn {
      display: block;
      width: 100%;
      padding: 12px;
      text-align: left;
      color: var(--primary-color);
      background: none;
      border: none;
      border-top: 1px solid var(--border-color);
      margin-top: 8px;
      cursor: pointer;
      font-size: 15px;
      border-radius: 0 0 16px 16px;
      &:hover {
          background-color: var(--section-hover-bg);
      }
  }
  .threads-header-top {
      margin-top: 1.5rem;
  }

  .user-results-list.full .user-card {
      border-bottom: 1px solid var(--border-color);
      &:last-child {
          border-bottom: none;
      }
  }

  .recent-searches-dropdown {
      background-color: var(--background);
      border: 1px solid var(--border-color);
      border-radius: 8px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.1);
      margin-top: 4px;
      position: absolute;
      top: 110%;
      width: 100%;
      z-index: 101;
      .dropdown-header {
          display: flex; justify-content: space-between; align-items: center;
          padding: 8px 12px; font-size: 15px; font-weight: bold;
          border-bottom: 1px solid var(--border-color);
          .clear-btn-sidebar {
            background: none; border: none; color: var(--primary-color);
            cursor: pointer; font-size: 14px; padding: 4px 0;
            &:hover { text-decoration: underline; }
          }
      }
      ul { list-style: none; margin: 0; padding: 0; }
      li .recent-item-btn {
          display: block; width: 100%; text-align: left;
          padding: 10px 12px; background: none; border: none;
          color: var(--text-color); cursor: pointer; font-size: 15px;
          &:hover { background-color: var(--section-hover-bg); }
      }
  }

  .community-results-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 16px;
      margin-top: 1rem;
  }
  </style>