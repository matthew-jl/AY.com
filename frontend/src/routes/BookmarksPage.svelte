<script lang="ts">
    import { onMount } from 'svelte';
    import { api, ApiError, type ThreadData, type FeedResponse } from '../lib/api';
    import ThreadComponent from '../components/ThreadComponent.svelte';
    import { user as currentUserStore } from '../stores/userStore';
  
    let bookmarkedThreads: ThreadData[] = [];
    let isLoading = false;
    let error: string | null = null;
    let searchQuery = '';
  
    $: filteredThreads = searchQuery
      ? bookmarkedThreads.filter(thread =>
          thread.content.toLowerCase().includes(searchQuery.toLowerCase()) ||
          thread.author?.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          thread.author?.username.toLowerCase().includes(searchQuery.toLowerCase())
        )
      : bookmarkedThreads;
  
  async function fetchBookmarkedThreads(page = 1, limit = 20, reset = false) {
    if (isLoading) return;
    isLoading = true;
    console.log("Loading bookmarks...");
    if (reset) {
        bookmarkedThreads = [];
        error = null;
    }

    try {
      const response: FeedResponse = await api.getBookmarkedThreads(page, limit);
      if (response && response.threads) {
        bookmarkedThreads = reset ? response.threads : [...bookmarkedThreads, ...response.threads];
        console.log("Fetched bookmarks:", bookmarkedThreads.length);
      }
    } catch (err) {
      console.error("Error fetching bookmarks:", err);
      if (err instanceof ApiError && err.status === 401) {
          error = "Please log in to view your bookmarks.";
      } else if (err instanceof ApiError) {
          error = `Could not load bookmarks: ${err.message}`;
      } else {
          error = "An unexpected error occurred while loading bookmarks.";
      }
    } finally {
      isLoading = false;
      console.log("Done loading bookmarks.");
    }
  }

  function handleThreadInteractionUpdate(event: CustomEvent<{ id: number; interaction?: 'unbookmark' | 'delete' }>) {
    const { id, interaction } = event.detail;
    if (interaction === 'unbookmark' || interaction === 'delete') {
        bookmarkedThreads = bookmarkedThreads.filter(t => t.id !== id);
    }
  }

    onMount(() => {
      console.log("onMount is running");
      fetchBookmarkedThreads(1, 20, true);
    });
  
  </script>
  
  <div class="bookmarks-page">
    <header class="page-header">
      <div class="header-content">
          <h2>Bookmarks</h2>
      </div>
      <div class="search-bar-bookmarks">
          <svg viewBox="0 0 24 24" class="search-icon"><g><path d="M10.25 3.75c..."></path></g></svg>
          <input type="text" placeholder="Search Bookmarks" bind:value={searchQuery} />
      </div>
    </header>
  
    <section class="bookmarks-feed">
      {#if isLoading && bookmarkedThreads.length === 0}
        <p>Loading bookmarks...</p> <!-- TODO: Skeleton for initial load -->
      {#each {length: 5} as _}
        <div class="skeleton-thread">...</div>
      {/each}
      {:else if error}
        <p class="error-text api-error">{error}</p>
      {:else if filteredThreads.length > 0}
        {#each filteredThreads as thread (thread.id)}
        <ThreadComponent {thread} 
            on:delete={handleThreadInteractionUpdate} 
            on:interaction={(e) => {
                if (e.detail.type === 'unbookmark') {
                    handleThreadInteractionUpdate(e);
                }
            }} 
        />
        {/each}
      {:else if searchQuery && filteredThreads.length === 0}
        <p class="empty-feed">No bookmarks found for "{searchQuery}".</p>
      {:else}
        <p class="empty-feed">You haven't bookmarked any threads yet.</p>
      {/if}
    </section>
  </div>
  
  <style lang="scss">
  @use '../styles/variables' as *;

  .bookmarks-page { width: 100%; }

  .page-header {
    position: sticky; 
    top: 0;
    background-color: rgba(var(--background-rgb), 0.85);
    backdrop-filter: blur(12px);
    z-index: 10;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-color);

    .header-content {
      h2 { 
        font-size: 20px; 
        font-weight: 800; 
        margin: 0 0 4px; 
      }
      .secondary-text { 
        font-size: 13px; 
        color: var(--secondary-text-color); 
        margin: 0; 
      }
    }
  }

  .search-bar-bookmarks {
    position: relative; 
    display: flex; 
    align-items: center; 
    margin-top: 12px;
    
    .search-icon {
      position: absolute; 
      left: 12px; 
      top: 50%; 
      transform: translateY(-50%);
      width: 18px; 
      height: 18px; 
      fill: var(--secondary-text-color);
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
        outline: none; 
        border-color: var(--primary-color);
        background-color: var(--background);
        box-shadow: 0 0 0 1px var(--primary-color);
      }
    }
  }

  .bookmarks-feed {
    /* Basic styles for the feed container */
  }
  
  .empty-feed {
     text-align: center;
     padding: 40px 20px;
     color: var(--secondary-text-color);
  }
  
  .error-text { 
    color: var(--error-color); 
    font-size: 0.85rem; 
    margin-top: 4px; 
  }
  
  .api-error { 
    margin-top: 1rem; 
    text-align: center; 
    font-weight: bold; 
  }
  
  /* Skeleton loading styles */
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

  /* Responsive styles */
  @media (max-width: 1200px) {
    /* Smaller desktop/large tablet */
    .page-header {
      padding: 10px 14px;
    }
  }
  
  @media (max-width: 900px) {
    /* Tablet/small tablet */
    .page-header .header-content h2 {
      font-size: 18px;
    }
    
    .search-bar-bookmarks input[type="text"] {
      padding: 9px 12px 9px 36px;
      font-size: 14px;
    }
    
    .search-bar-bookmarks .search-icon {
      width: 16px;
      height: 16px;
      left: 10px;
    }
  }
  
  @media (max-width: 768px) {
    /* Large mobile */
    .page-header {
      padding: 10px 12px;
    }
    
    .empty-feed {
      padding: 30px 16px;
      font-size: 14px;
    }
  }
  
  @media (max-width: 576px) {
    /* Mobile */
    .page-header .header-content h2 {
      font-size: 16px;
    }
    
    .search-bar-bookmarks {
      margin-top: 10px;
      
      input[type="text"] {
        padding: 8px 10px 8px 34px;
        font-size: 14px;
      }
      
      .search-icon {
        width: 15px;
        height: 15px;
      }
    }
    
    .empty-feed {
      padding: 25px 12px;
      font-size: 13px;
    }
    
    .api-error {
      font-size: 13px;
    }
  }
  
  @media (max-width: 400px) {
    /* Small mobile */
    .page-header {
      padding: 8px 10px;
    }
    
    .page-header .header-content h2 {
      font-size: 15px;
    }
    
    .search-bar-bookmarks {
      margin-top: 8px;
      
      input[type="text"] {
        padding: 7px 8px 7px 30px;
        font-size: 13px;
      }
      
      .search-icon {
        width: 14px;
        height: 14px;
        left: 8px;
      }
    }
  }
</style>