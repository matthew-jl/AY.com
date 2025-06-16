<script lang="ts">    import { onMount } from 'svelte';
    import { link, navigate } from 'svelte-routing';
    import { api, ApiError, type ThreadData, type MediaMetadata, type FeedResponse } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
    import ThreadComponent from '../components/ThreadComponent.svelte';
    // import CreateThreadForm from '../components/CreateThreadForm.svelte';
    import { ArrowLeft, Bookmark, Heart, MessageSquare, Repeat2, Share2, X } from 'lucide-svelte';
  import { closeCreateThreadModal, isCreateThreadModalOpen, openCreateThreadModal } from '../stores/modalStore';
  import CreateThreadModal from './CreateThreadModal.svelte';
  import ShareThreadModal from './ShareThreadModal.svelte';
  
    export let threadId: string;
  
    let mainThread: ThreadData | null = null;
    let replies: ThreadData[] = [];
    let isLoadingThread = true;
    let isLoadingReplies = false;
    let error: string | null = null;
    let currentPage = 1;
    let hasMoreReplies = true;
  
    // Media overlay state
    let showMediaOverlay = false;
    let mediaOverlayItems: MediaMetadata[] = [];
    let currentMediaIndex = 0;

    let parentThreadToReplyTo: ThreadData | null = null;
  
  // --- Data Fetching ---
    async function fetchThreadDetails(id: number) {
      isLoadingThread = true;
      error = null;
      
      try {
        // Use the getThread API method which returns ThreadData directly
        const response = await api.getThread(id);
        
        if (response) {
          mainThread = response;
          console.log("Thread fetched:", mainThread);
          // Fetch replies
          currentPage = 1; // Reset page for replies
          replies = []; // Clear old replies
          hasMoreReplies = true; // Assume has more initially
          fetchReplies(mainThread.id, 1, false);
        } else {
          error = "Thread not found or returned invalid data.";
        }
      } catch (err) {
        console.error("Error fetching thread details:", err);
        if (err instanceof ApiError && err.status === 404) {
          error = "Thread not found.";
        } else {
          error = "Could not load thread. Please try again later.";
        }
      } finally {
        isLoadingThread = false;
      }
    }
    async function fetchReplies(parentId: number, page = 1, append = false) {
      if (isLoadingReplies && !append) return;
      if (!hasMoreReplies && append) return;
      
      isLoadingReplies = true;
      const limit = 10; // replies per page
      
      try {
        const response: FeedResponse = await api.getRepliesForThread(parentId, page, limit);
        const loadedReplies = response.threads || [];

        if (append) {
            replies = [...replies, ...loadedReplies];
        } else {
            replies = loadedReplies;
        }
        currentPage = page;
        hasMoreReplies = response.has_more ?? (loadedReplies.length === limit);

        console.log(`Loaded ${loadedReplies.length} replies for thread ${parentId}, page ${page}. HasMore: ${hasMoreReplies}`);
      } catch (err) {
        console.error("Error fetching replies:", err);
        hasMoreReplies = false;
      } finally {
        isLoadingReplies = false;
      }
    }
  
    function loadMoreReplies() {
      if (!isLoadingReplies && hasMoreReplies && mainThread) {
        fetchReplies(mainThread.id, currentPage + 1, true);
      }
    }
  
    // Initialize thread fetch on mount or when threadId changes
    $: if (threadId && !isNaN(parseInt(threadId))) {
        const numId = parseInt(threadId);
        if (!mainThread || mainThread.id !== numId) {
            fetchThreadDetails(numId);
        }
    } else if (threadId && mainThread && mainThread.id !== parseInt(threadId)) {
        fetchThreadDetails(parseInt(threadId));
    }

    function openReplyModal(threadToReply: ThreadData) {
        parentThreadToReplyTo = threadToReply;
        openCreateThreadModal();
    }
  
    function handleReplyCreated(event: CustomEvent<ThreadData>) {
        const newReply = event.detail;
        if (!mainThread) return;

        // Add new reply to the top of the replies list
        replies = [newReply, ...replies];

        // Increment reply count on the parent thread (main or another reply)
        if (newReply.parent_thread_id === mainThread.id) {
            mainThread = { ...mainThread, reply_count: (mainThread.reply_count || 0) + 1 };
        } else {
            // If replying to a reply, find and update that reply's count
            replies = replies.map(r =>
                r.id === newReply.parent_thread_id
                    ? { ...r, reply_count: (r.reply_count || 0) + 1 }
                    : r
            );
        }
        closeCreateThreadModal();
    }
  
    function handleThreadInteractionUpdate(event: CustomEvent<{ id: number; interactionType: string; newState: boolean; newCount: number }>) {
      const { id, interactionType, newState, newCount } = event.detail;
      
      const updateThread = (thread: ThreadData): ThreadData => {
        if (thread.id === id) {
          const updatedThread = { ...thread };
          
          if (interactionType === 'like') {
            updatedThread.is_liked_by_current_user = newState;
            updatedThread.like_count = newCount;
          } else if (interactionType === 'bookmark') {
            updatedThread.is_bookmarked_by_current_user = newState;
          }
          
          return updatedThread;
        }
        return thread;
      };
      
      // Update main thread if it matches
      if (mainThread && mainThread.id === id) {
        mainThread = updateThread(mainThread);
      }
      
      // Update in replies if it matches any
      replies = replies.map(reply => updateThread(reply));
    }

    function handleThreadDelete(event: CustomEvent<{ id: number }>) {
      const idToDelete = event.detail.id;
      if (mainThread && mainThread.id === idToDelete) { navigate('/home', { replace: true }); }
      else {
        replies = replies.filter(r => r.id !== idToDelete);
        if (mainThread && mainThread.reply_count && mainThread.reply_count > 0) {
          // If a reply to the main thread was deleted
          if (replies.find(r => r.parent_thread_id === mainThread?.id && r.id === idToDelete) === undefined) {
             mainThread = { ...mainThread, reply_count: mainThread.reply_count - 1 };
          }
        }
      }
    }

    // --- Media Overlay Functions ---
    function openMediaOverlay(event: CustomEvent<{ media: MediaMetadata[], index: number }>) {
      const { media, index } = event.detail;
      if (!media || media.length === 0) return;
      
      mediaOverlayItems = media;
      currentMediaIndex = index || 0;
      showMediaOverlay = true;
      
      // Prevent scrolling of the body when overlay is open
      document.body.style.overflow = 'hidden';
    }
    
    function closeMediaOverlay() {
      showMediaOverlay = false;
      
      // Re-enable scrolling
      document.body.style.overflow = '';
    }
    
    function goToNextMedia() {
      if (currentMediaIndex < mediaOverlayItems.length - 1) {
        currentMediaIndex++;
      }
    }
    
    function goToPreviousMedia() {
      if (currentMediaIndex > 0) {
        currentMediaIndex--;
      }
    }
    
    function handleKeydown(event: KeyboardEvent) {
      if (!showMediaOverlay) return;
      
      switch (event.key) {
        case 'Escape':
          closeMediaOverlay();
          break;
        case 'ArrowRight':
          goToNextMedia();
          break;
        case 'ArrowLeft':
          goToPreviousMedia();
          break;
      }
    }

    // Media navigation shorthand
    function navigateMedia(direction: 'prev' | 'next') {
      if (direction === 'prev') {
        goToPreviousMedia();
      } else {
        goToNextMedia();
      }
    }
  // Add these variables and functions to handle thread interactions in overlay
  let overlayShareModalOpen = false;
  let overlayThreadToShare: ThreadData | null = null;
  
  async function handleOverlayAction(action: 'like' | 'bookmark' | 'reply' | 'share') {
    if (!mainThread) return;
    
    switch (action) {
      case 'like':
        await handleOverlayLike();
        break;
      case 'bookmark':
        await handleOverlayBookmark();
        break;
      case 'reply':
        closeMediaOverlay();
        openReplyModal(mainThread);
        break;
      case 'share':
        overlayThreadToShare = mainThread;
        overlayShareModalOpen = true;
        break;
    }
  }

  async function handleOverlayLike() {
    if (!mainThread) return;
    
    const originalLiked = mainThread.is_liked_by_current_user || false;
    const originalCount = mainThread.like_count || 0;
    
    // Optimistic update
    mainThread = {
      ...mainThread,
      is_liked_by_current_user: !originalLiked,
      like_count: originalLiked ? originalCount - 1 : originalCount + 1
    };

    try {
      if (!originalLiked) {
        await api.likeThread(mainThread.id);
      } else {
        await api.unlikeThread(mainThread.id);
      }
    } catch (err) {
      console.error("Like/Unlike error in overlay:", err);
      // Revert optimistic update on error
      mainThread = {
        ...mainThread,
        is_liked_by_current_user: originalLiked,
        like_count: originalCount
      };
    }
  }

  async function handleOverlayBookmark() {
    if (!mainThread) return;
    
    const originalBookmarked = mainThread.is_bookmarked_by_current_user || false;
    const originalCount = mainThread.bookmark_count || 0;
    
    // Optimistic update
    mainThread = {
      ...mainThread,
      is_bookmarked_by_current_user: !originalBookmarked,
      bookmark_count: originalBookmarked ? originalCount - 1 : originalCount + 1
    };

    try {
      if (!originalBookmarked) {
        await api.bookmarkThread(mainThread.id);
      } else {
        await api.unbookmarkThread(mainThread.id);
      }
    } catch (err) {
      console.error("Bookmark error in overlay:", err);
      // Revert optimistic update on error
      mainThread = {
        ...mainThread,
        is_bookmarked_by_current_user: originalBookmarked,
        bookmark_count: originalCount
      };
    }
  }
  
  function handleOverlayThreadSent(event: CustomEvent<{chatIds: number[]}>) {
    console.log(`Thread ${mainThread?.id} sent to chats from overlay:`, event.detail.chatIds);
    overlayShareModalOpen = false;
  }
</script>

<div class="thread-detail-page">
  <header class="page-header-simple">
    <button class="back-button" on:click={() => window.history.back()} aria-label="Go back">
      <ArrowLeft size={20} />
    </button>
    <h2>Thread</h2>
  </header>

  {#if isLoadingThread && !mainThread}
    <div class="loading-container">
      <!-- Thread skeleton -->
      <div class="skeleton-thread full-page">
        <div class="skeleton-avatar large"></div>
        <div class="skeleton-content">
          <div class="skeleton-line short"></div>
          <div class="skeleton-line long"></div>
          <div class="skeleton-line medium"></div>
          <div class="skeleton-media"></div>
        </div>
      </div>
    </div>
  {:else if error}
    <div class="error-container">
      <p class="error-text api-error">{error}</p>
      <a href="/home" use:link class="btn btn-outline btn-sm">Return to Home</a>
    </div>
  {:else if mainThread}
    <!-- Main Thread Display -->
    <div class="main-thread-section">
      <ThreadComponent
        thread={mainThread}
        disableNavigationClick
        on:delete={handleThreadDelete}
        on:interaction={handleThreadInteractionUpdate}
        on:mediaClick={openMediaOverlay}
        on:replyto={() => openReplyModal(mainThread!)}
      />
    </div>    <!-- Reply Input Section -->
    {#if $currentUserStore}
      <div class="reply-form-prompt">
        <div class="avatar-placeholder-small">
            {#if $currentUserStore.profile_picture} <img src={$currentUserStore.profile_picture} alt="Your avatar" class="avatar-image"/>
            {:else} {$currentUserStore.name.charAt(0).toUpperCase()} {/if}
        </div>
        <button class="prompt-to-reply-btn" on:click={() => openReplyModal(mainThread!)}>
            Post your reply...
        </button>
      </div>
    {:else}
      <div class="login-to-reply">
        <p>
          <a href="/login?redirect={window.location.pathname}" use:link>Log in</a> or
          <a href="/register?redirect={window.location.pathname}" use:link>Sign up</a> to reply.
        </p>
      </div>
    {/if}

    <!-- Replies Section -->
    <div class="replies-section">
      <h3 class="replies-header">Replies</h3>
      
      {#if isLoadingReplies && replies.length === 0}
        <div class="loading-replies">
          <!-- Reply skeletons -->
          {#each Array(2) as _, i}
            <div class="skeleton-thread">
              <div class="skeleton-avatar"></div>
              <div class="skeleton-content">
                <div class="skeleton-line short"></div>
                <div class="skeleton-line long"></div>
                <div class="skeleton-line medium"></div>
              </div>
            </div>
          {/each}
        </div>
      {:else if replies.length > 0}
        <div class="replies-list">
          {#each replies as reply (reply.id)}
            <ThreadComponent
              thread={reply}
              on:delete={handleThreadDelete}
              on:interaction={handleThreadInteractionUpdate}
              on:mediaClick={openMediaOverlay}
              on:replyto={() => openReplyModal(reply)}
            />
          {/each}
        </div>
        
        {#if hasMoreReplies}
          <div class="load-more-container">
            <button 
              class="btn btn-outline btn-sm load-more-btn" 
              on:click={loadMoreReplies} 
              disabled={isLoadingReplies}
            >
              {isLoadingReplies ? 'Loading...' : 'Load More Replies'}
            </button>
          </div>
        {:else if replies.length > 5}
          <p class="end-of-replies">You've reached the end of replies</p>
        {/if}
      {:else if !isLoadingReplies}
        <p class="empty-replies">No replies yet. Be the first!</p>
      {/if}
    </div>
  {:else}
    <p class="error-text">Thread could not be loaded.</p>
  {/if}
</div>

<!-- Create Thread Modal -->
{#if $isCreateThreadModalOpen && parentThreadToReplyTo}
  <CreateThreadModal
      parentThreadId={parentThreadToReplyTo.id}
      replyingToUsername={parentThreadToReplyTo.author?.username}
      on:close={closeCreateThreadModal}
      on:threadcreated={handleReplyCreated}
  />
{/if}

<!-- Media Overlay -->
{#if showMediaOverlay && mediaOverlayItems.length > 0}
  <div 
    class="media-overlay" 
    role="dialog"
    aria-modal="true" 
    aria-label="Media preview"
    on:click={closeMediaOverlay}
  >
    <div 
      class="media-overlay-content" 
      on:click|stopPropagation
    >
      <button 
        class="media-close-btn" 
        on:click={closeMediaOverlay} 
        aria-label="Close media preview"
      >
        <X size={24} />
      </button>
      
      <div class="media-display">
        {#if mediaOverlayItems[currentMediaIndex].mime_type.startsWith('image/')}
          <img 
            src={mediaOverlayItems[currentMediaIndex].public_url} 
            alt="Media content from thread" 
            class="media-image" 
          />
        {:else if mediaOverlayItems[currentMediaIndex].mime_type.startsWith('video/')}
          <video 
            src={mediaOverlayItems[currentMediaIndex].public_url} 
            controls 
            autoplay 
            class="media-video"
          >
            <track kind="captions" src="" label="English" />
            Your browser does not support the video tag.
          </video>
        {:else}
          <div class="media-file">
            <p>This file type cannot be previewed</p>
            <a href={mediaOverlayItems[currentMediaIndex].public_url} target="_blank" rel="noopener noreferrer" class="btn btn-primary">
              Download File
            </a>
          </div>
        {/if}
      </div>
      
      <!-- Add thread interaction buttons in media overlay -->
      {#if mainThread}
        <div class="media-overlay-thread-actions">
          <button class="action-btn reply" aria-label="Reply" on:click|stopPropagation={() => handleOverlayAction('reply')}>
            <MessageSquare size={20} />
          </button>
          <button class="action-btn repost" aria-label="Repost">
            <Repeat2 size={20} />
            <span>{mainThread.repost_count > 0 ? mainThread.repost_count : ''}</span>
          </button>
          <button 
            class="action-btn like" 
            class:liked={mainThread.is_liked_by_current_user} 
            on:click|stopPropagation={() => handleOverlayAction('like')}
            aria-pressed={mainThread.is_liked_by_current_user} 
            aria-label={mainThread.is_liked_by_current_user ? 'Unlike' : 'Like'}>
            <Heart 
              size={20} 
              fill={mainThread.is_liked_by_current_user ? '#f91880' : 'none'} 
              stroke={mainThread.is_liked_by_current_user ? '#f91880' : 'white'}
            />
            <span>{mainThread.like_count > 0 ? mainThread.like_count : ''}</span>
          </button>
          <button 
            class="action-btn bookmark" 
            class:bookmarked={mainThread.is_bookmarked_by_current_user} 
            on:click|stopPropagation={() => handleOverlayAction('bookmark')} 
            aria-pressed={mainThread.is_bookmarked_by_current_user} 
            aria-label={mainThread.is_bookmarked_by_current_user ? 'Remove bookmark' : 'Bookmark'}>
            <Bookmark 
              size={20} 
              fill={mainThread.is_bookmarked_by_current_user ? 'var(--primary-color)' : 'none'} 
              stroke={mainThread.is_bookmarked_by_current_user ? 'var(--primary-color)' : 'white'} 
            />
            <span>{mainThread.bookmark_count > 0 ? mainThread.bookmark_count : ''}</span>
          </button>
          <button class="action-btn share" aria-label="Share" on:click|stopPropagation={() => handleOverlayAction('share')}>
            <Share2 size={20} />
          </button>
        </div>
      {/if}
      
      {#if mediaOverlayItems.length > 1}
        <div class="media-navigation">
          <button 
            class="media-nav-btn prev" 
            on:click|stopPropagation={() => navigateMedia('prev')} 
            aria-label="Previous media"
            disabled={currentMediaIndex === 0}
          >
            ←
          </button>
          <span class="media-counter">{currentMediaIndex + 1} / {mediaOverlayItems.length}</span>
          <button 
            class="media-nav-btn next" 
            on:click|stopPropagation={() => navigateMedia('next')} 
            aria-label="Next media"
            disabled={currentMediaIndex === mediaOverlayItems.length - 1}
          >
            →
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

{#if overlayShareModalOpen && overlayThreadToShare}
  <ShareThreadModal 
    threadToShare={overlayThreadToShare} 
    on:close={() => overlayShareModalOpen = false} 
    on:sent={handleOverlayThreadSent} 
  />
{/if}

<style lang="scss">
  @use '../styles/variables' as *;

  .thread-detail-page {
    width: 100%;
    height: 100%;
    min-height: 100vh;
  }

  .page-header-simple {
    display: flex;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-color);
    background-color: var(--background);
    backdrop-filter: blur(12px);
    position: sticky;
    top: 0;
    z-index: 10;

    .back-button {
      background: none;
      border: none;
      color: var(--text-color);
      border-radius: 50%;
      width: 36px;
      height: 36px;
      display: flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      transition: background-color 0.2s;
      
      &:hover {
        background-color: var(--section-hover-bg);
      }
    }

    h2 {
      font-size: 1.2rem;
      font-weight: bold;
      margin: 0 0 0 12px;
    }
  }

  .loading-container {
    padding: 16px;
  }

  .error-container {
    padding: 24px 16px;
    text-align: center;
    
    .btn {
      margin-top: 16px;
    }
  }

  .main-thread-section {
    border-bottom: 1px solid var(--border-color);
    /* ThreadComponent handles its own padding */
  }

  .reply-form-prompt {
      display: flex;
      align-items: center;
      padding: 12px 16px;
      border-bottom: 10px solid var(--section-bg);
      gap: 10px;
       [data-theme="dark"] & { border-bottom-color: var(--border-color); }
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
          overflow: hidden;
          .avatar-image {
            width: 100%;
            height: 100%;
            border-radius: 50%;
            object-fit: cover;
          }
        }
       .prompt-to-reply-btn {
           flex-grow: 1;
           text-align: left;
           padding: 10px 12px;
           border-radius: 20px;
           border: 1px solid var(--border-color);
           background-color: var(--input-bg);
           color: var(--secondary-text-color);
           font-size: 15px;
           cursor: text;
           &:hover { border-color: var(--primary-color); }
       }
  }

  .login-to-reply {
    text-align: center;
    padding: 24px 16px;
    border-bottom: 10px solid var(--section-bg);
    color: var(--secondary-text-color);
    
    p {
      margin: 0;
    }
    
    a {
      color: var(--primary-color);
      text-decoration: none;
      font-weight: 500;
      
      &:hover {
        text-decoration: underline;
      }
    }
  }

  .replies-section {
    padding: 0 0 24px;
  }
  
  .replies-header {
    font-size: 1.1rem;
    font-weight: 700;
    padding: 16px 16px 8px;
    margin: 0;
  }
  .replies-list {
    /* ThreadComponent handles its own styles */
    margin-bottom: 8px;
  }

  .loading-replies {
    padding: 0 16px;
  }

  .load-more-container {
    display: flex;
    justify-content: center;
    padding: 16px;
    
    .load-more-btn {
      min-width: 140px;
    }
  }

  .empty-replies, .end-of-replies {
    text-align: center;
    padding: 24px 16px;
    color: var(--secondary-text-color);
    margin: 0;
  }

  .error-text {
    text-align: center;
    padding: 24px 16px;
    color: var(--error-color);
  }

  .api-error {
    background-color: var(--error-bg);
    padding: 12px;
    border-radius: 8px;
    margin: 16px;
  }

  .btn {
    display: inline-block;
    padding: 8px 16px;
    border-radius: 9999px;
    font-weight: 600;
    font-size: 15px;
    cursor: pointer;
    text-align: center;
    transition: background-color 0.2s;
    text-decoration: none;
    
    &.btn-outline {
      background: transparent;
      border: 1px solid var(--border-color);
      color: var(--text-color);
      
      &:hover:not(:disabled) {
        background-color: var(--section-hover-bg);
      }
    }
    
    &.btn-primary {
      background-color: var(--primary-color);
      color: var(--primary-button-text);
      border: none;
      
      &:hover:not(:disabled) {
        background-color: var(--primary-color-hover);
      }
    }
    
    &.btn-sm {
      padding: 6px 12px;
      font-size: 14px;
    }
    
    &:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
  }

  /* Skeleton loaders */
  .skeleton-thread {
    display: flex;
    gap: 12px;
    padding: 16px;
    border-bottom: 1px solid var(--border-color);
    animation: pulse 1.5s ease-in-out infinite;
    
    .skeleton-avatar {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      background-color: var(--section-hover-bg);
      flex-shrink: 0;
      
      &.large {
        width: 48px;
        height: 48px;
      }
    }
    
    .skeleton-content {
      width: 100%;
      
      .skeleton-line {
        height: 12px;
        border-radius: 4px;
        background-color: var(--section-hover-bg);
        margin-bottom: 8px;
        
        &.short {
          width: 30%;
        }
        
        &.medium {
          width: 60%;
        }
        
        &.long {
          width: 90%;
        }
      }
      
      .skeleton-media {
        height: 200px;
        border-radius: 12px;
        background-color: var(--section-hover-bg);
        margin-top: 12px;
      }
    }
  }

  @keyframes pulse {
    0% { opacity: 0.6; }
    50% { opacity: 1; }
    100% { opacity: 0.6; }
  }

  /* Media overlay styles */
  .media-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.9);
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    padding: 0;
    margin: 0;
    transition: opacity 0.2s ease;
  }

  .media-overlay-content {
    position: relative;
    width: 90%;
    height: 85%;
    max-width: 1200px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    cursor: default;
  }

  .media-close-btn {
    position: absolute;
    top: -40px;
    right: 0;
    background: none;
    border: none;
    color: white;
    font-size: 24px;
    cursor: pointer;
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    z-index: 1002;
    
    &:hover {
      background-color: rgba(255, 255, 255, 0.2);
    }
    
    &:focus {
      outline: 2px solid white;
      outline-offset: 2px;
    }
  }

  .media-display {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    
    .media-image {
      max-width: 100%;
      max-height: 85vh;
      object-fit: contain;
      border-radius: 2px;
    }
    
    .media-video {
      max-width: 100%;
      max-height: 85vh;
      border-radius: 2px;
    }
    
    .media-file {
      background-color: rgba(255, 255, 255, 0.1);
      border-radius: 8px;
      padding: 24px;
      text-align: center;
      color: white;
      
      p {
        margin-bottom: 16px;
      }
    }
  }

  .media-navigation {
    position: absolute;
    bottom: -50px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 16px;
    color: white;
    
    .media-nav-btn {
      background: rgba(0, 0, 0, 0.5);
      border: 1px solid rgba(255, 255, 255, 0.3);
      color: white;
      font-size: 20px;
      cursor: pointer;
      width: 40px;
      height: 40px;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 50%;
      
      &:hover:not(:disabled) {
        background-color: rgba(255, 255, 255, 0.2);
      }
      
      &:focus {
        outline: 2px solid white;
        outline-offset: 2px;
      }
      
      &:disabled {
        opacity: 0.3;
        cursor: not-allowed;
      }
    }
    
    .media-counter {
      font-size: 14px;
      min-width: 60px;
      text-align: center;
    }
  }

  
  .media-overlay-thread-actions {
    display: flex;
    justify-content: space-around;
    width: 100%;
    max-width: 500px;
    margin: 16px auto 0;
    background-color: rgba(0, 0, 0, 0.5);
    border-radius: 30px;
    padding: 10px;
    position: absolute;
    bottom: 20px;
    left: 50%;
    transform: translateX(-50%);
  }
  
  .media-overlay-thread-actions .action-btn {
    background: none;
    border: none;
    color: white;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 14px;
    padding: 8px 12px;
    border-radius: 50%;
    transition: color 0.2s ease, background-color 0.2s ease;
    
    svg {
      width: 20px;
      height: 20px;
    }
    
    span {
      color: white;
    }
    
    &:hover {
      background-color: rgba(255, 255, 255, 0.2);
    }
    
    &.like.liked {
      color: #f91880;
    }
    
    &.bookmark.bookmarked {
      color: var(--primary-color);
    }
  }
  
  
  /* Enhanced responsive styles */
  @media (max-width: 768px) {
    .media-overlay-content {
      width: 95%;
    }
    
    .media-display .media-image,
    .media-display .media-video {
      max-height: 80vh;
    }
    .media-overlay-thread-actions {
      bottom: 10px;
      padding: 8px;
      max-width: 400px;
    }
    
    .media-overlay-thread-actions .action-btn {
      padding: 6px 10px;
      
      svg {
        width: 18px;
        height: 18px;
      }
      
      span {
        font-size: 12px;
      }
    }
    
    .media-navigation {
      bottom: -50px;
    }
  }

  @media (max-width: 480px) {
    .media-overlay-content {
      width: 98%;
      height: 70%;
    }
    
    .media-close-btn {
      top: -30px;
      right: 0;
    }
    
    .media-navigation {
      bottom: -40px;
      
      .media-nav-btn {
        width: 36px;
        height: 36px;
      }
    }
    
    .media-display .media-image,
    .media-display .media-video {
      max-height: 65vh;
    }
    .media-overlay-thread-actions {
      bottom: 5px;
      padding: 5px;
      max-width: 300px;
    }
    
    .media-overlay-thread-actions .action-btn {
      padding: 5px 8px;
      
      svg {
        width: 16px;
        height: 16px;
      }
      
      span {
        display: none;
      }
    }
  }

</style>