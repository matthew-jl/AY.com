<script lang="ts">
  import type { ThreadData } from '../lib/api'; // Use the defined interface
  import { api, ApiError } from '../lib/api';
  import { user } from '../stores/userStore'; // To check ownership
  import { createEventDispatcher } from 'svelte';
  import { link } from 'svelte-routing'; // For potential links in content/author
  import { timeAgo } from '../lib/utils/timeAgo'; // Helper for relative time (create this)
  import { MessageSquare, Repeat2, Heart, Bookmark, Share2, MoreHorizontal } from 'lucide-svelte';

  export let thread: ThreadData;

  const dispatch = createEventDispatcher();

  // Local reactive state for interactions (allows optimistic UI)
  let isLiked = thread.is_liked ?? false;
  let isBookmarked = thread.is_bookmarked ?? false;
  let likeCount = thread.like_count ?? 0;
  // let bookmarkCount = thread.bookmark_count ?? 0; // Add if needed

  let interactionError: string | null = null;
  let isDeleting = false;

  $: isOwnThread = $user?.id === thread.user_id;
  $: author = thread.author; // Assuming author is pre-populated

  // --- Interaction Handlers ---
  async function handleLike() {
    interactionError = null;
    const originalLiked = isLiked;
    const originalCount = likeCount;

    // Optimistic update
    isLiked = !isLiked;
    likeCount = isLiked ? originalCount + 1 : originalCount - 1;

    try {
      if (isLiked) {
        await api.likeThread(thread.id);
      } else {
        await api.unlikeThread(thread.id);
      }
      console.log(`Like/Unlike success for thread ${thread.id}`);
    } catch (err) {
      console.error("Like/Unlike error:", err);
      // Revert optimistic update on error
      isLiked = originalLiked;
      likeCount = originalCount;
      interactionError = "Failed to update like status.";
    }
  }

  async function handleBookmark() {
    interactionError = null;
    const originalBookmarked = isBookmarked;
    isBookmarked = !isBookmarked; // Optimistic update

    try {
      if (isBookmarked) {
        await api.bookmarkThread(thread.id);
      } else {
        await api.unbookmarkThread(thread.id);
      }
      console.log(`Bookmark/Unbookmark success for thread ${thread.id}`);
    } catch (err) {
      console.error("Bookmark error:", err);
      isBookmarked = originalBookmarked; // Revert
      interactionError = "Failed to update bookmark status.";
    }
  }

  async function handleDelete() {
    if (!isOwnThread || !confirm(`Are you sure you want to delete this thread?`)) {
        return;
    }
    isDeleting = true;
    interactionError = null;
    try {
        await api.deleteThread(thread.id);
        console.log(`Deleted thread ${thread.id}`);
        dispatch('delete', { id: thread.id }); // Notify parent
    } catch (err) {
         console.error("Delete error:", err);
         interactionError = "Failed to delete thread.";
         isDeleting = false;
    }
    // No need to set isDeleting=false on success, component will be removed
  }

  // TODO: Add Reply and Repost handlers
</script>

<article class="thread-card" aria-labelledby="thread-author-{thread.id}">
    <div class="thread-avatar">
        <div class="avatar-placeholder-small">
            {author?.name?.charAt(0)?.toUpperCase() ?? '?'}
            <!-- TODO: <img src={author.profile_picture} alt=""> -->
        </div>
         <!-- TODO: Add line connecting replies later -->
    </div>
    <div class="thread-content">
        <div class="thread-header">
            {#if author}
                <a href="/profile/{author.username}" use:link class="author-link" id="thread-author-{thread.id}">
                    <span class="author-name">{author.name}</span>
                    <span class="author-handle">@{author.username}</span>
                </a>
            {:else}
                <span class="author-name">Unknown User</span>
            {/if}
            <span class="dot">·</span>
            <span class="timestamp" title={new Date(thread.posted_at).toLocaleString()}>
                <!-- Use relative time -->
                {timeAgo(thread.posted_at)}
            </span>
            {#if isOwnThread}
                 <button class="more-options-btn" on:click={handleDelete} disabled={isDeleting} aria-label="Delete thread">
                    <MoreHorizontal size={18} />   
                  <!-- <svg viewBox="0 0 24 24"><g><path d="M3 12c0-1.1.9-2 2-2s2 .9 2 2-.9 2-2 2-2-.9-2-2zm9 2c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm7 0c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2z"></path></g></svg> -->
                </button>
            {/if}
        </div>

        {#if thread.content}
            <p class="thread-text">{thread.content}</p>
        {/if}

        <!-- Media Grid -->
        {#if thread.media && thread.media.length > 0}
            <div class="media-grid count-{thread.media.length}">
                 {#each thread.media as media (media.id)}
                    <div class="media-item">
                         {#if media.mime_type.startsWith('image/')}
                             <img src={media.public_url} alt="Thread media" loading="lazy"/>
                         {:else if media.mime_type.startsWith('video/')}
                             <video controls preload="metadata" src={media.public_url}></video>
                         {:else}
                             <div class="file-placeholder">Unsupported Media</div>
                         {/if}
                    </div>
                 {/each}
            </div>
        {/if}

        <div class="thread-actions">
            <button class="action-btn reply" aria-label="Reply">
                <!-- <svg viewBox="0 0 24 24"><g><path d="M1.751 10c0-4.42 3.584-8 8.005-8h4.366c4.49 0 8.129 3.64 8.129 8.13 0 2.96-1.607 5.68-4.196 7.11l-8.054 4.46v-3.69h-.067c-4.49.1-8.183-3.51-8.183-8.01zm8.005-6c-3.317 0-6.005 2.69-6.005 6 0 3.37 2.77 6.08 6.138 6.01l.351-.01h1.761v2.3l5.087-2.81c1.951-1.08 3.163-3.13 3.163-5.36 0-3.39-2.744-6.13-6.129-6.13H9.756z"></path></g></svg> -->
                <MessageSquare size={18} />
                <span>{thread.reply_count > 0 ? thread.reply_count : ''}</span>
            </button>
             <button class="action-btn repost" aria-label="Repost">
                <!-- <svg viewBox="0 0 24 24"><g><path d="M4.5 3.88l4.43-4.43L10 1l-6 6-6-6 1.07-1.55L4.5 3.88zM19.5 20.12l-4.43 4.43L14 23l6-6 6 6-1.07 1.55L19.5 20.12zM14.93 4.24L22 11.31v-2.83L14.93 1.4l-2.12 2.12 7.07 7.07-7.07 7.07 2.12 2.12L22 12.69v-2.83L14.93 4.24zM9.07 19.76L2 12.69v2.83L9.07 22.6l2.12-2.12-7.07-7.07 7.07-7.07-2.12-2.12L2 11.31v2.83L9.07 19.76z"></path></g></svg> -->
                <Repeat2 size={18} />
                <span>{thread.repost_count > 0 ? thread.repost_count : ''}</span>
            </button>
             <button class="action-btn like" class:liked={isLiked} on:click={handleLike} aria-pressed={isLiked} aria-label={isLiked ? 'Unlike' : 'Like'}>
               <!-- {#if isLiked}
                    <svg viewBox="0 0 24 24" class="liked-icon"><g><path d="M12 21.638h-.014C9.403 21.59 1.95 14.856 1.95 8.478c0-3.064 2.525-5.754 5.403-5.754 2.29 0 4.134 1.336 5.103 3.24H12v1.453c0 .414.336.75.75.75h5.5c.414 0 .75-.336.75-.75V7.677c1.926-1.912 3.24-4.36 3.24-7.138 0-3.064-2.525-5.754-5.403-5.754-2.29 0-4.134 1.336-5.103 3.24h-.037c-.967-1.903-2.81-3.24-5.102-3.24z"></path></g></svg>
                {:else}
                   <svg viewBox="0 0 24 24"><g><path d="M16.697 5.5c-1.222-.06-2.679.51-3.89 2.16l-.805 1.09-.806-1.09C9.984 6.01 8.526 5.44 7.304 5.5c-1.845.09-3.391 1.64-3.391 3.43 0 3.16 2.18 5.78 5.08 8.28l.806.69.806-.69c2.9-2.5 5.08-5.12 5.08-8.28 0-1.79-1.546-3.34-3.393-3.43z"></path></g></svg>
                {/if} -->
                <Heart size={18} fill={isLiked ? '#f91880' : 'none'} stroke={isLiked ? '#f91880' : 'currentColor'} />
                 <span>{likeCount > 0 ? likeCount : ''}</span>
            </button>
             <button class="action-btn bookmark" class:bookmarked={isBookmarked} on:click={handleBookmark} aria-pressed={isBookmarked} aria-label={isBookmarked ? 'Remove bookmark' : 'Bookmark'}>
                 <Bookmark size={18} fill={isBookmarked ? 'var(--primary-color)' : 'none'} stroke={isBookmarked ? 'var(--primary-color)' : 'currentColor'} />
                 <!-- No count usually shown for bookmark -->
            </button>
            <!-- Share Button Placeholder -->
             <button class="action-btn share" aria-label="Share">
                 <!-- <svg viewBox="0 0 24 24"><g><path d="M12 2.59l5.7 5.7-1.41 1.42L13 6.41V16h-2V6.41L7.71 9.71 6.3 8.29l5.7-5.7zM20 17v3h-2v-3H6v3H4v-3H2v-2h20v2h-2z"></path></g></svg> -->
                  <Share2 size={18} />
             </button>
        </div>
        {#if interactionError} <p class="error-text interaction-error">{interactionError}</p> {/if}
    </div>
</article>

<style lang="scss">
  @use '../styles/variables' as *;

  .thread-card {
    display: flex;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-color);
    transition: background-color 0.15s ease-in-out;

    &:hover {
      background-color: rgba(var(--text-color-rgb, 0, 0, 0), 0.03);
       [data-theme="dark"] & {
           background-color: rgba(var(--text-color-rgb, 255, 255, 255), 0.03);
       }
    }
  }

  .thread-avatar {
    margin-right: 12px;
    flex-shrink: 0;
     .avatar-placeholder-small {
          width: 40px; height: 40px; border-radius: 50%; background-color: var(--secondary-text-color);
          color: var(--background); display: flex; align-items: center; justify-content: center;
          font-weight: bold; font-size: 1.1rem;
     }
  }

  .thread-content {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
  }

  .thread-header {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 2px;
    color: var(--secondary-text-color);
    font-size: 15px;

    .author-link {
        text-decoration: none;
        color: inherit;
        display: flex;
        align-items: center;
        gap: 4px;
         &:hover .author-name {
             text-decoration: underline;
         }
    }

    .author-name {
      font-weight: bold;
      color: var(--text-color);
       white-space: nowrap;
       overflow: hidden;
       text-overflow: ellipsis;
       max-width: 150px;
    }
    .author-handle {
       white-space: nowrap;
       overflow: hidden;
       text-overflow: ellipsis;
       max-width: 120px;
    }
    .dot { margin: 0 2px; }
    .timestamp { white-space: nowrap; }
     .more-options-btn {
         margin-left: auto;
         background: none; border: none; padding: 4px; border-radius: 50%; cursor: pointer;
         display: flex; align-items: center; justify-content: center; color: var(--secondary-text-color);
          &:hover {
              background-color: rgba(var(--primary-color-rgb), 0.1);
              color: var(--primary-color);
          }
          svg { width: 18px; height: 18px; fill: currentColor; }
     }
  }

  .thread-text {
    color: var(--text-color);
    font-size: 15px;
    line-height: 1.4;
    white-space: pre-wrap;
    word-wrap: break-word;
    margin-bottom: 12px;
  }

  .media-grid {
      border-radius: 16px;
      overflow: hidden;
      margin-bottom: 12px;
      display: grid;
      gap: 2px;
      border: 1px solid var(--border-color);

      &.count-1 { grid-template-columns: 1fr; }
      &.count-2 { grid-template-columns: 1fr 1fr; }
      &.count-3 { grid-template-columns: 1fr 1fr; .media-item:first-child { grid-row: span 2;} }
      &.count-4 { grid-template-columns: 1fr 1fr; }

      .media-item {
          position: relative;
          background-color: var(--section-bg);
          aspect-ratio: 16 / 9;
          /* Ensure first item stretches in 3-item layout */
          &:first-child:has(+ .media-item + .media-item:last-child) { aspect-ratio: 8 / 9; }
           &:nth-child(2):has(~ .media-item:last-child) { /* Target second item only if there's a third */ }
           &:nth-child(3):last-child { /* Target third item only if it's the last */ }


          img, video {
              display: block;
              width: 100%;
              height: 100%;
              object-fit: cover;
              cursor: pointer;
          }
           .file-placeholder { /* Style for unsupported media */
                display: flex; align-items: center; justify-content: center;
                height: 100%; color: var(--secondary-text-color); font-size: 0.9rem;
           }
      }
  }

  .thread-actions {
    display: flex;
    justify-content: space-between;
    margin-top: 8px;
    max-width: 425px;
  }

  .action-btn {
      background: none;
      border: none;
      color: var(--secondary-text-color);
      cursor: pointer;
      display: flex;
      align-items: center;
      gap: 5px; /* Space between icon and count */
      font-size: 13px;
      padding: 6px;
      border-radius: 50%;
      transition: color 0.2s ease, background-color 0.2s ease;

      svg {
          width: 18px;
          height: 18px;
          fill: currentColor;
      }

      span { /* Number count */
          line-height: 1;
      }

      /* Specific hover colors */
    &.reply:hover { color: var(--primary-color); background-color: rgba(var(--primary-color-rgb, 29, 155, 240), 0.1); }
    &.repost:hover { color: #00ba7c; background-color: rgba(0, 186, 124, 0.1); } /* Green */
    &.like:hover { color: #f91880; background-color: rgba(249, 24, 128, 0.1); } /* Pink */
    &.bookmark:hover { color: var(--primary-color); background-color: rgba(var(--primary-color-rgb, 29, 155, 240), 0.1); }
    &.share:hover { color: var(--primary-color); background-color: rgba(var(--primary-color-rgb, 29, 155, 240), 0.1); }

       /* Active state colors */
    &.like.liked { color: #f91880; .liked-icon { fill: #f91880; } }
    &.bookmark.bookmarked { color: var(--primary-color); .bookmarked-icon { fill: var(--primary-color); } }
  }

  .interaction-error {
      font-size: 12px;
      margin-top: 4px;
      color: var(--error-color);
  }

</style>