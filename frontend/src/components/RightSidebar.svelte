<script lang="ts">
  import { link, navigate } from 'svelte-routing';
  import { onMount } from 'svelte';

  let searchQuery = '';
  let recentSearches: string[] = [];

  function handleSearchSubmit() {
    if (searchQuery.trim()) {
      addRecentSearch(searchQuery.trim());
      navigate(`/explore?q=${encodeURIComponent(searchQuery.trim())}`);
      searchQuery = '';
    }
  }

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
    function clearAllRecent() {
      recentSearches = [];
      saveRecentSearches();
    }
    function searchFromRecentSidebar(term: string) {
      addRecentSearch(term); // Still add it even if clicked, moves to top
      navigate(`/explore?q=${encodeURIComponent(term)}`);
    }
  

  onMount(loadRecentSearches);
</script>

<aside class="right-sidebar">
  <div class="sticky-container">
    <div class="search-container">
      <form class="search-bar" on:submit|preventDefault={handleSearchSubmit}>
        <svg viewBox="0 0 24 24" class="search-icon">...</svg>
        <input type="text" placeholder="Search" aria-label="Search query" bind:value={searchQuery} />
        {#if searchQuery} <button type="button" class="clear-search-btn-sidebar" on:click={() => searchQuery = ''}>×</button> {/if}
        <button type="submit" style="display:none;" aria-hidden="true"></button>
      </form>
       <!-- Display recent searches if search query is empty and recent searches exist -->
      {#if !searchQuery && recentSearches.length > 0}
        <div class="recent-searches-dropdown">
            <div class="dropdown-header">
                <span>Recent</span>
                <button class="clear-btn-sidebar" on:click={clearAllRecent}>Clear all</button>
            </div>
            <ul>
                {#each recentSearches.slice(0,3) as term (term)}
                    <li><button class="recent-item-btn-sidebar" on:click={() => searchFromRecentSidebar(term)}>{term}</button></li>
                {/each}
            </ul>
        </div>
      {/if}
    </div>

    <!-- ... (Premium Box, What's Happening, Who to Follow sections remain) ... -->
  </div>
</aside>

<style lang="scss">
  @use '../styles/variables' as *;

  .right-sidebar {
    width: $right-sidebar-width;
    padding-left: 15px;
    padding-right: 15px; 
    position: relative; 
    height: 100vh; 
  }

  .sticky-container {
    position: sticky; 
    top: 10px; 
    height: calc(100vh - 20px); 
    overflow-y: auto;
    scrollbar-width: none; 
    &::-webkit-scrollbar { 
        display: none;
    }
  }

  .content-box {
    background: var(--section-bg);
    border-radius: 16px;
    margin-bottom: 15px;
    overflow: hidden;
  }

  .content-box h3 {
    font-size: 20px;
    font-weight: 800;
    padding: 12px 16px;
    margin: 0;
    border-bottom: 1px solid var(--border-color);
  }

  .premium-box {
     padding: 12px 16px;
     h3 {
         font-size: 17px;
         font-weight: bold;
         padding: 0;
         border-bottom: none;
         margin-bottom: 4px;
     }
     p {
         font-size: 14px;
         color: var(--secondary-text-color);
         line-height: 1.3;
         margin-bottom: 12px;
     }
     .premium-button {
         background: var(--follow-button-bg);
         color: var(--follow-button-text);
         border: 1px solid var(--follow-button-border);
         padding: 6px 16px;
         border-radius: 9999px;
         cursor: pointer;
         font-weight: bold;
         font-size: 14px;
         transition: background-color 0.2s ease-in-out;
         &:hover {
             background: var(--follow-button-hover-bg);
         }
     }
  }


  .list-item {
    list-style: none;
    margin: 0;
    padding: 0;
    border-bottom: 1px solid var(--border-color);
     &:last-child {
       border-bottom: none;
     }

    a {
        display: block;
        padding: 12px 16px;
        text-decoration: none;
        color: inherit;
        transition: background-color 0.2s ease-in-out;
         &:hover {
           background-color: var(--section-hover-bg);
           cursor: pointer;
         }
    }
  }


  .trend-item {
    .item-context {
      font-size: 13px;
      color: var(--secondary-text-color);
      margin-bottom: 2px;
    }
    .item-title {
      font-weight: bold;
      font-size: 15px;
      color: var(--text-color);
      line-height: 1.3;
      margin-bottom: 4px;
    }
  }

  .follow-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;

    &:hover {
       background-color: var(--section-hover-bg);
    }

    .follow-link {
      display: flex;
      align-items: center;
      text-decoration: none;
      color: inherit;
      flex-grow: 1;
      padding: 0;
       &:hover {
         background-color: transparent;
       }
    }

     .avatar-placeholder {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        background-color: var(--secondary-text-color);
        margin-right: 12px;
        flex-shrink: 0;
     }
     .user-info {
        display: flex;
        flex-direction: column;
        line-height: 1.2;
        overflow: hidden;
     }
     .user-name {
        font-weight: bold;
        font-size: 15px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
     }
     .user-handle {
        font-size: 14px;
        color: var(--secondary-text-color);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
     }

    .follow-button {
        background: var(--follow-button-bg);
        color: var(--follow-button-text);
        border: 1px solid var(--follow-button-border);
        padding: 6px 16px;
        border-radius: 9999px;
        cursor: pointer;
        font-weight: bold;
        font-size: 14px;
        flex-shrink: 0;
        margin-left: 10px;
        transition: background-color 0.2s ease-in-out;
         &:hover {
           background: var(--follow-button-hover-bg);
         }
    }
  }

  .show-more {
    padding: 12px 16px;
    a {
        color: var(--primary-color);
        font-size: 15px;
        text-decoration: none;
         &:hover {
           text-decoration: underline;
           background-color: transparent;
         }
    }
  }

  ul {
      margin: 0;
      padding: 0;
  }

  .search-container {
      position: sticky; 
      top: 0;
      background: var(--background); 
      padding-top: 5px;
      padding-bottom: 5px;
      z-index: 1; 
      margin-left: -10px;
      margin-right: -10px;
      padding-left: 10px;
      padding-right: 10px;

  }

  .search-bar {
    position: relative;
    margin-bottom: 15px;

    input {
      width: 100%;
      padding: 12px 12px 12px 40px;
      border-radius: 9999px;
      border: 1px solid transparent;
      background: var(--search-bg);
      color: var(--text-color);
      font-size: 15px;
      &:focus {
        outline: none;
        border-color: var(--search-border-focus);
        background: var(--background);
        box-shadow: 0 0 0 1px var(--search-border-focus);
      }
    }

    .clear-search-btn-sidebar {
        position: absolute; right: 10px; top: 50%; transform: translateY(-50%);
        background: var(--secondary-text-color); color: var(--background);
        border: none; border-radius: 50%; width: 20px; height: 20px;
        font-size: 14px; line-height: 18px; text-align: center; cursor: pointer;
        display: flex; align-items: center; justify-content: center;
        &:hover { background: var(--text-color); }
    }

    .search-icon {
        position: absolute;
        top: 50%;
        left: 12px;
        transform: translateY(-50%);
        width: 18px;
        height: 18px;
        fill: var(--secondary-text-color);
        pointer-events: none;
    }

  }

  .recent-searches-dropdown {
      background-color: var(--background);
      border: 1px solid var(--border-color);
      border-radius: 8px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.1);
      margin-top: -10px;
      position: absolute;
      width: calc(100% - 20px);
      z-index: 5;

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
      li .recent-item-btn-sidebar {
          display: block; width: 100%; text-align: left;
          padding: 10px 12px; background: none; border: none;
          color: var(--text-color); cursor: pointer; font-size: 15px;
          &:hover { background-color: var(--section-hover-bg); }
      }
  }

</style>