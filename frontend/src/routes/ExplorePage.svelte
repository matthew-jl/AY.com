<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { api, ApiError, type ThreadData, type UserProfileBasic, type FeedResponse, type TrendingHashtagItem, type CommunityListItem } from '../lib/api';
    import { currentPathname } from '../stores/locationStore';
    import { navigate, link } from 'svelte-routing';
    import ThreadComponent from '../components/ThreadComponent.svelte';
    import UserCard from '../components/UserCard.svelte';
    import CommunityCard from '../components/CommunityCard.svelte';
    import { ChevronDownIcon } from 'lucide-svelte';
    import { user as currentUserStore } from '../stores/userStore';
  
    type SearchTab = 'top' | 'latest' | 'people' | 'media' | 'communities';
    const DEFAULT_TAB: SearchTab = 'top';
  
    // --- Search State ---
    let searchQuery = ''; // Bound to search input
    let debouncedSearchQuery = ''; // For API calls after debounce
    let searchInputEl: HTMLInputElement;
    let recentSearches: string[] = [];
    let activeTab: SearchTab = DEFAULT_TAB;

    // --- Filters ---
    type UserFilterType = 'everyone' | 'following' | 'verified';
    let selectedUserFilter: UserFilterType = 'everyone';
    let selectedCategoryFilters: string[] = []; // Stores values like "world", "sports"
    const predefinedCategories = [
        { value: 'world', label: 'World' }, { value: 'sports', label: 'Sports' },
        { value: 'business', label: 'Business' }, { value: 'sci_tech', label: 'Sci/Tech' },
    ];
    let showCategoryFilterDropdown = false;
    let showUserFilterDropdown = false;
  
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
      const initialCats = urlParams.get('categories');
      const initialUserFilter = urlParams.get('user_filter') as UserFilterType | null;

      if (initialQuery) { searchQuery = initialQuery; debouncedSearchQuery = initialQuery; }
      if (initialCats) { selectedCategoryFilters = initialCats.split(','); }
      if (initialUserFilter && ['everyone', 'following', 'verified'].includes(initialUserFilter)) {
          selectedUserFilter = initialUserFilter;
      }
      if (initialQuery || initialCats || initialUserFilter) {
          performSearch(debouncedSearchQuery, activeTab);
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
          updateUrlAndSearch();
        } else {
          clearResults();
          navigate('/explore', { replace: true });
        }
      }, 900);
    }

    function updateUrlAndSearch() {
      if (debouncedSearchQuery) addRecentSearch(debouncedSearchQuery);
      performSearch(debouncedSearchQuery, activeTab);

      const params = new URLSearchParams();
      if (debouncedSearchQuery) params.set('q', debouncedSearchQuery);
      if (selectedCategoryFilters.length > 0) params.set('categories', selectedCategoryFilters.join(','));
      if (selectedUserFilter !== 'everyone') params.set('user_filter', selectedUserFilter);

      const queryString = params.toString();
      navigate(queryString ? `/explore?${queryString}` : '/explore', { replace: true });
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
          const threadResp = await api.searchThreads(query, 1, 10, selectedUserFilter, selectedCategoryFilters);
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
      updateUrlAndSearch();
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

    function toggleCategoryFilter(category: string) {
      const index = selectedCategoryFilters.indexOf(category);
      if (index > -1) selectedCategoryFilters = selectedCategoryFilters.filter(c => c !== category);
      else selectedCategoryFilters = [...selectedCategoryFilters, category];
      updateUrlAndSearch(); // Refetch with new category filter
    }
    function selectUserFilter(filter: UserFilterType) {
      selectedUserFilter = filter;
      showUserFilterDropdown = false; // Close dropdown
      updateUrlAndSearch(); // Refetch with new user filter
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
      <div class="explore-filters">
          <!-- User Filter Dropdown -->
          <div class="filter-dropdown-container">
              <button class="filter-btn" on:click={() => showUserFilterDropdown = !showUserFilterDropdown}>
                  Filter By: {selectedUserFilter.charAt(0).toUpperCase() + selectedUserFilter.slice(1)} <ChevronDownIcon size={16}/>
              </button>
              {#if showUserFilterDropdown}
                  <div class="dropdown-menu user-filter-dropdown">
                      <button on:click={() => selectUserFilter('everyone')} class:active={selectedUserFilter === 'everyone'}>Everyone</button>
                      <button on:click={() => selectUserFilter('following')} class:active={selectedUserFilter === 'following'} disabled={!$currentUserStore}>People you follow</button>
                      <button on:click={() => selectUserFilter('verified')} class:active={selectedUserFilter === 'verified'} disabled>Verified accounts only</button> <!-- Disabled until implemented -->
                  </div>
              {/if}
          </div>
          <!-- Category Filter Dropdown -->
          <div class="filter-dropdown-container">
              <button class="filter-btn" on:click={() => showCategoryFilterDropdown = !showCategoryFilterDropdown}>
                  Categories {selectedCategoryFilters.length > 0 ? `(${selectedCategoryFilters.length})` : ''} <ChevronDownIcon size={16}/>
              </button>
              {#if showCategoryFilterDropdown}
              <div class="dropdown-menu category-filter-dropdown">
                  {#each predefinedCategories as category (category.value)}
                      <label>
                          <input
                              type="checkbox"
                              value={category.value}
                              checked={selectedCategoryFilters.includes(category.value)}
                              on:change={() => toggleCategoryFilter(category.value)}
                          />
                          {category.label}
                      </label>
                  {/each}
              </div>
              {/if}
          </div>
      </div>
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
              <!-- Loading skeletons based on active tab -->
              {#if activeTab === 'top'}
                  <!-- Top section skeletons -->
                  <div class="skeleton-section">
                      <div class="skeleton-section-header"></div>
                      <!-- People skeletons -->
                      {#each { length: 2 } as _}
                          <div class="skeleton-user-card">
                              <div class="skeleton-avatar"></div>
                              <div class="skeleton-content">
                                  <div class="skeleton-line short"></div>
                                  <div class="skeleton-line medium"></div>
                              </div>
                          </div>
                      {/each}
                  </div>
                  <!-- Thread skeletons -->
                  {#each { length: 3 } as _}
                      <div class="skeleton-thread">
                          <div class="skeleton-avatar"></div>
                          <div class="skeleton-content">
                              <div class="skeleton-line short"></div>
                              <div class="skeleton-line long"></div>
                              <div class="skeleton-line medium"></div>
                          </div>
                      </div>
                  {/each}
              {:else if activeTab === 'latest'}
                  <!-- Thread skeletons for latest tab -->
                  {#each { length: 5 } as _}
                      <div class="skeleton-thread">
                          <div class="skeleton-avatar"></div>
                          <div class="skeleton-content">
                              <div class="skeleton-line short"></div>
                              <div class="skeleton-line long"></div>
                              <div class="skeleton-line medium"></div>
                          </div>
                      </div>
                  {/each}
              {:else if activeTab === 'people'}
                  <!-- People skeletons -->
                  {#each { length: 5 } as _}
                      <div class="skeleton-user-card">
                          <div class="skeleton-avatar"></div>
                          <div class="skeleton-content">
                              <div class="skeleton-line short"></div>
                              <div class="skeleton-line medium"></div>
                              <div class="skeleton-line long"></div>
                          </div>
                      </div>
                  {/each}
              {:else if activeTab === 'media'}
                  <!-- Media grid skeletons -->
                  <div class="skeleton-media-grid">
                      {#each { length: 12 } as _}
                          <div class="skeleton-media-item"></div>
                      {/each}
                  </div>
              {:else if activeTab === 'communities'}
                  <!-- Communities skeletons -->
                  <div class="skeleton-communities-grid">
                      {#each { length: 4 } as _}
                          <div class="skeleton-community-card">
                              <div class="skeleton-community-header"></div>
                              <div class="skeleton-line medium"></div>
                              <div class="skeleton-line long"></div>
                              <div class="skeleton-stats"></div>
                          </div>
                      {/each}
                  </div>
              {/if}
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

  .explore-filters {
      display: flex;
      gap: 10px;
      margin-top: 10px;
      padding: 0 0 10px 0; /* Add padding below filters if search bar is above */
  }
  .filter-dropdown-container {
      position: relative;
  }
  .filter-btn {
      background-color: var(--input-bg);
      border: 1px solid var(--border-color);
      color: var(--text-color);
      padding: 6px 12px;
      border-radius: 16px;
      font-size: 14px;
      cursor: pointer;
      display: flex;
      align-items: center;
      gap: 4px;
      &:hover { border-color: var(--secondary-text-color); }
  }
  .dropdown-menu {
    position: absolute; top: 110%;
    left: 0;
    background-color: var(--background);
    border: 1px solid var(--border-color);
    border-radius: 8px; box-shadow: 0 3px 10px rgba(0,0,0,0.1);
    padding: 8px; z-index: 20; min-width: 200px;
    max-height: 250px; overflow-y: auto;

    button, label {
      display: block; width: 100%; text-align: left;
      padding: 8px 10px; background: none; border: none;
      color: var(--text-color); cursor: pointer; font-size: 14px;
      border-radius: 4px;
      input[type="checkbox"] { margin-right: 8px; accent-color: var(--primary-color); }
      &:hover { background-color: var(--section-hover-bg); }
      &.active { background-color: var(--primary-color-light); color: var(--primary-color); font-weight: bold;}
    }
  }
  .user-filter-dropdown button:disabled {
      color: var(--secondary-text-color);
      opacity: 0.6;
      cursor: not-allowed;
      &:hover { background: none; }
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
      padding: 16px 8px;
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

  /* Responsive styles for various screen sizes */
  @media (max-width: 1200px) {
    .community-results-grid {
      grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    }
  }

  @media (max-width: 900px) {
    .explore-tabs button {
      font-size: 14px;
      padding: 14px 4px;
    }
    
    .community-results-grid {
      grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
      gap: 12px;
    }
  }

  @media (max-width: 768px) {
    .explore-header {
      padding: 8px 12px;
    }
    
    .explore-media-grid {
      grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
    }
    
    .community-results-grid {
      grid-template-columns: 1fr 1fr;
      gap: 10px;
    }
    
    .search-results-content {
      padding: 12px;
    }
    
    .trending-hashtags h3 {
      font-size: 18px;
    }
  }

  @media (max-width: 600px) {
    .explore-tabs {
      overflow-x: auto;
      -ms-overflow-style: none;
      scrollbar-width: none;
      &::-webkit-scrollbar {
        display: none;
      }
      
      button {
        white-space: nowrap;
        padding: 12px 16px;
        flex: 0 0 auto;
        
        &:first-child {
          margin-left: 0;
        }
      }
    }
    
    .search-results-content {
      padding: 10px;
    }

    .floating-user-suggestions {
      max-height: 280px;
    }
    
    .explore-media-grid {
      grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
      gap: 3px;
    }
    
    .community-results-grid {
      grid-template-columns: 1fr;
      gap: 8px;
    }
    
    .avatar-placeholder-small {
      width: 36px;
      height: 36px;
    }
    
    .simple-user-card.large .avatar-placeholder-small {
      width: 40px;
      height: 40px;
    }
  }

  @media (max-width: 480px) {
    .explore-header {
      padding: 6px 10px;
    }
    
    .search-bar-container {
      input[type="text"] {
        padding: 8px 10px 8px 36px;
        font-size: 14px;
      }
      
      .search-icon {
        left: 10px;
        width: 16px;
        height: 16px;
      }
    }
    
    .trending-hashtags, .recent-searches {
      padding: 10px 12px;
      
      h3 {
        font-size: 16px;
        margin-bottom: 10px;
      }
      
      .trend-link {
        padding: 6px 0;
        
        .trend-tag {
          font-size: 14px;
        }
        
        .trend-category, .trend-posts {
          font-size: 12px;
        }
      }
    }
    
    .user-results-list {
      .simple-user-card, .user-card {
        padding: 10px 0;
      }
    }
    
    .top-people-section h4 {
      font-size: 16px;
    }
    
    .view-all-btn {
      padding: 10px;
      font-size: 14px;
    }
    
    .explore-media-grid .media-grid-item .video-placeholder-explore {
      font-size: 1.5rem;
    }
  }

  @media (max-width: 380px) {
    .explore-tabs button {
      padding: 10px 12px;
      font-size: 13px;
    }
    
    .avatar-placeholder-small {
      width: 32px;
      height: 32px;
    }
    
    .search-results-content h4 {
      font-size: 15px;
      margin: 1rem 0 0.4rem;
    }
    
    .explore-media-grid {
      grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
      gap: 2px;
    }
    
    .api-error {
      font-size: 13px;
    }
    
    .simple-user-card .user-bio-snippet {
      font-size: 12px;
      -webkit-line-clamp: 2;
      display: -webkit-box;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }
  }

  /* Add skeleton loading animations */
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
  }
  
  .skeleton-line.short { width: 30%; }
  .skeleton-line.medium { width: 60%; }
  .skeleton-line.long { width: 90%; }
  
  .skeleton-section {
    border-bottom: 1px solid var(--border-color);
    padding-bottom: 12px;
    margin-bottom: 16px;
  }
  
  .skeleton-section-header {
    height: 18px;
    width: 80px;
    background-color: var(--section-hover-bg);
    animation: pulse 1.5s infinite ease-in-out;
    border-radius: 4px;
    margin-bottom: 12px;
  }
  
  .skeleton-user-card {
    display: flex;
    padding: 12px 0;
    border-bottom: 1px solid var(--border-color);
    gap: 12px;
  }
  
  .skeleton-media-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 4px;
  }
  
  .skeleton-media-item {
    aspect-ratio: 1 / 1;
    background-color: var(--section-hover-bg);
    animation: pulse 1.5s infinite ease-in-out;
    border-radius: 8px;
  }
  
  .skeleton-communities-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
    margin-top: 1rem;
  }
  
  .skeleton-community-card {
    padding: 16px;
    border: 1px solid var(--border-color);
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  
  .skeleton-community-header {
    height: 24px;
    width: 70%;
    background-color: var(--section-hover-bg);
    animation: pulse 1.5s infinite ease-in-out;
    border-radius: 4px;
  }
  
  .skeleton-stats {
    display: flex;
    gap: 16px;
    margin-top: 8px;
  }
  
  .skeleton-stats::before,
  .skeleton-stats::after {
    content: "";
    height: 8px;
    border-radius: 4px;
    background-color: var(--section-hover-bg);
    animation: pulse 1.5s infinite ease-in-out;
    flex: 1;
  }
  
  /* Make skeletons responsive */
  @media (max-width: 768px) {
    .skeleton-communities-grid {
      grid-template-columns: 1fr 1fr;
      gap: 10px;
    }
    
    .skeleton-media-grid {
      grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
    }
  }
  
  @media (max-width: 600px) {
    .skeleton-thread {
      padding: 10px 12px;
      gap: 8px;
    }
    
    .skeleton-avatar {
      width: 36px;
      height: 36px;
    }
    
    .skeleton-communities-grid {
      grid-template-columns: 1fr;
    }
    
    .skeleton-media-grid {
      grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
      gap: 3px;
    }
  }
  
  @media (max-width: 480px) {
    .skeleton-avatar {
      width: 32px;
      height: 32px;
    }
    
    .skeleton-media-grid {
      grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
      gap: 2px;
    }
  }
  </style>