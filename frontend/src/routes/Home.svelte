<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { api, ApiError, type ThreadData } from '../lib/api';
  import ThreadComponent from '../components/ThreadComponent.svelte';
  import { openCreateThreadModal } from '../stores/modalStore';
  import { user } from '../stores/userStore';

  type FeedTab = 'foryou' | 'following';

  let activeTab: FeedTab = 'foryou';
  let threads: ThreadData[] = [];
  let isLoading = false;
  let error: string | null = null;
  let currentPage = 1;
  let hasMore = true;
  let sentinel: Element; // For Intersection Observer

  async function fetchThreads(page = 1, limit = 10, feedType: FeedTab = activeTab) {
      if (isLoading || !hasMore && page > 1) return; // Prevent multiple simultaneous loads or loading past end
      isLoading = true;
      error = null;
      console.log(`Fetching page ${page} for ${feedType} feed...`);

      try {
          const response = await api.getFeedThreads(page, limit, feedType);
          console.log("API Response Object:", response);
          if (response && Array.isArray(response.threads)) {
              const fetchedThreads = response.threads;
              console.log("Fetched Threads Array:", fetchedThreads);
              console.log("First fetched thread (if exists):", fetchedThreads[0]); // Log first item
              if (fetchedThreads.length > 0) {
                  console.log('Appending/Replacing threads...');
                  threads = page === 1 ? fetchedThreads : [...threads, ...fetchedThreads];
                  currentPage = page;
                  // Use has_more from the response if available, otherwise estimate
                  hasMore = response.has_more ?? (fetchedThreads.length === limit);
                   console.log(`Fetch complete. Total threads: ${threads.length}, Has more: ${hasMore}`);
              } else {
                  // If page 1 returns empty, set hasMore to false. If later page empty, it's the end.
                  hasMore = false;
                  console.log("No more threads returned from API.");
              }
          } else {
              // Handle unexpected response structure
              console.warn("Unexpected API response structure:", response);
              hasMore = false; // Stop loading if response format is wrong
              if(page === 1) {
                   error = "Received invalid data from server for feed.";
              }
          }
      } catch (err) {
          console.error(`Error fetching ${feedType} feed:`, err);
          if (err instanceof ApiError) { error = `Failed to load feed: ${err.message}`; }
          else if (err instanceof Error) { error = `Error: ${err.message}`; }
          else { error = 'An unexpected error occurred loading threads.'; }
          hasMore = false; // Stop trying on error
      } finally {
          isLoading = false;
      }
  }

  function handleThreadDelete(event: CustomEvent<{ id: number }>) {
      console.log("Deleting thread from feed:", event.detail.id);
      threads = threads.filter(t => t.id !== event.detail.id);
  }

  function switchTab(tab: FeedTab) {
      if (activeTab === tab) return; // No change needed
      activeTab = tab;
      // Reset state for new tab
      threads = [];
      currentPage = 1;
      hasMore = true;
      error = null;
      fetchThreads(); // Fetch initial data for the new tab
  }

  // --- Intersection Observer for Infinite Scroll ---
  let observer: IntersectionObserver;

  function handleIntersect(entries: IntersectionObserverEntry[]) {
      const entry = entries[0]; // Usually only one entry
      if (entry.isIntersecting && hasMore && !isLoading) {
          console.log("Sentinel intersecting, loading more...");
          fetchThreads(currentPage + 1);
      }
  }

  onMount(() => {
      fetchThreads(); // Initial load

      // Setup Observer
      observer = new IntersectionObserver(handleIntersect, {
          root: null, // Use viewport
          threshold: 0.1 // Trigger when 10% visible
      });

      if (sentinel) {
          observer.observe(sentinel);
      }

      return () => {
          if (observer && sentinel) {
              observer.unobserve(sentinel);
          }
      };
  });

</script>

<div class="home-container">
  <header class="home-header">
      <h1>Home</h1>
      <div class="tabs">
          <button
              class:active={activeTab === 'foryou'}
              on:click={() => switchTab('foryou')}
          >For you</button>
          <button
              class:active={activeTab === 'following'}
              on:click={() => switchTab('following')}
          >Following</button>
      </div>
  </header>

  <!-- Simplified Create Thread Area -->
  <div class="create-thread-prompt">
       <div class="avatar-placeholder-small">
            {#if $user}
                {#if $user.profile_picture}
                <img src="{$user.profile_picture}" alt="{$user.name}" style="width:100%;height:100%;border-radius:50%;" />
                {:else}
                {$user.name.charAt(0).toUpperCase()}
                {/if}
            {/if}
       </div>
       <button class="prompt-button" on:click={openCreateThreadModal}>What's happening?!</button>
       <button class="btn btn-primary post-btn-inline" on:click={openCreateThreadModal} disabled={!$user}>Post</button>
  </div>


  <!-- Feed Area -->
  <section class="feed">
    {#if threads.length > 0}
        {#each threads as thread (thread.id)}
            <!-- TODO: Insert Ad Placeholder Logic -->
            <!-- {#if index > 0 && index % 5 === 0 } <AdPlaceholderComponent /> {/if} -->
            <ThreadComponent {thread} on:delete={handleThreadDelete} />
        {:else}
            <!-- This part only shows briefly before loading or if fetch returns empty FIRST time -->
            {#if !isLoading}
                 <p class="empty-feed">No threads to show yet.</p>
             {/if}
        {/each}

        <!-- Loading Indicator / Sentinel -->
         <div class="feed-status" bind:this={sentinel}>
             {#if isLoading}
                 <p>Loading more threads...</p> <!-- TODO: Replace with Skeleton Loader -->
                 <!-- Skeleton Placeholder -->
                 <div class="skeleton-thread">
                     <div class="skeleton-avatar"></div>
                     <div class="skeleton-content">
                         <div class="skeleton-line short"></div>
                         <div class="skeleton-line long"></div>
                         <div class="skeleton-line medium"></div>
                     </div>
                 </div>
             {:else if !hasMore}
                 <p>You've reached the end!</p>
             {:else}
                 <!-- Sentinel is invisible, observer watches it -->
             {/if}
        </div>

    {:else if isLoading}
         <!-- Initial Loading State -->
         <p>Loading feed...</p>
         <!-- Skeleton Placeholder -->
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

    {:else if error}
        <p class="error-text api-error">{error}</p>
    {:else}
        <!-- Should only show if not loading, no error, and zero threads after first fetch -->
        <p class="empty-feed">This feed is empty.</p>
    {/if}
  </section>

</div>

<style lang='scss'>
   @use '../styles/variables' as *;

  .home-container {
    // padding: 20px 30px;
    width: 100%;
    box-sizing: border-box;
  }

   .home-header {
    position: sticky;
    top: 0;
    background-color: rgba(var(--background-rgb), 0.85);
    backdrop-filter: blur(12px);
    z-index: 10;
    border-bottom: 1px solid var(--border-color);

    h1 {
      font-size: 20px;
      font-weight: 800;
      padding: 12px 16px;
      margin: 0;
    }

    .tabs {
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

        &:hover {
            background-color: var(--section-hover-bg);
        }

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
  }

  .create-thread-prompt {
      display: flex;
      align-items: flex-start;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      gap: 12px;

      .avatar-placeholder-small {
           width: 40px; height: 40px; border-radius: 50%; background-color: var(--secondary-text-color);
           color: var(--background); display: flex; align-items: center; justify-content: center;
           font-weight: bold; flex-shrink: 0; font-size: 1.1rem;
      }

      .prompt-button {
          flex-grow: 1;
          text-align: left;
          font-size: 20px;
          color: var(--secondary-text-color);
          background: none;
          border: none;
          padding: 8px 0;
          cursor: pointer;
           &:hover { }
      }

      .btn {
        display: block;
        padding: 0.8rem 1rem;
        border-radius: 9999px;
        text-decoration: none;
        font-weight: bold;
        font-size: 1rem;
        cursor: pointer;
        border: 1px solid transparent;
        transition: background-color 0.2s ease;
        margin-top: 1rem;
        //   width: 100%;
        }

        .btn-primary {
        background-color: var(--primary-color);
        color: var(--primary-button-text);
        border: 1px solid var(--border-color);
            &:hover:not(:disabled) {
                background-color: var(--primary-color-hover);
            }
            &:disabled {
                opacity: 0.6;
                cursor: not-allowed;
            }
        }

       .post-btn-inline {
            padding: 8px 16px;
            margin-top: 4px;
       }
  }


  .feed {
      /* Feed container styles */
  }

   .feed-status {
       text-align: center;
       padding: 20px;
       color: var(--secondary-text-color);
       font-size: 14px;
   }
    .empty-feed {
       text-align: center;
       padding: 40px 20px;
       color: var(--secondary-text-color);
    }

    /* Basic Skeleton Styles */
    @keyframes pulse { 0% { background-color: var(--section-hover-bg); } 50% { background-color: var(--border-color); } 100% { background-color: var(--section-hover-bg); } }
    .skeleton-thread { display: flex; padding: 12px 16px; border-bottom: 1px solid var(--border-color); gap: 12px; }
    .skeleton-avatar { width: 40px; height: 40px; border-radius: 50%; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out; flex-shrink: 0; }
    .skeleton-content { flex-grow: 1; display: flex; flex-direction: column; gap: 8px; padding-top: 4px; }
    .skeleton-line { height: 10px; border-radius: 4px; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out; }
    .skeleton-line.short { width: 30%; }
    .skeleton-line.medium { width: 60%; }
    .skeleton-line.long { width: 90%; }


  /* Error message styles */
  .error-text { color: var(--error-color); font-size: 0.85rem; margin-top: 4px; }
  .api-error { margin-top: 1rem; text-align: center; font-weight: bold; }

</style>