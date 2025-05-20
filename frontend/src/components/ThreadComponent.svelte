<script lang="ts">
  import type { ThreadData } from '../lib/api';
  import { api, ApiError } from '../lib/api';
  import { user } from '../stores/userStore';
  import { createEventDispatcher } from 'svelte';
  import { link } from 'svelte-routing';
  import { timeAgo } from '../lib/utils/timeAgo';
  import { MessageSquare, Repeat2, Heart, Bookmark, Share2, MoreHorizontal } from 'lucide-svelte';

  export let thread: ThreadData;

  const dispatch = createEventDispatcher();

  let isLiked = thread.is_liked_by_current_user ?? false;
  let isBookmarked = thread.is_bookmarked_by_current_user ?? false;
  let likeCount = thread.like_count ?? 0;
  let bookmarkCount = thread.bookmark_count ?? 0;

  let interactionError: string | null = null;
  let isDeleting = false;

  $: isOwnThread = $user?.id === thread.user_id;
  $: author = thread.author;

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
    isBookmarked = !isBookmarked;

    try {
      if (isBookmarked) {
        await api.bookmarkThread(thread.id);
      } else {
        await api.unbookmarkThread(thread.id);
      }
      console.log(`Bookmark/Unbookmark success for thread ${thread.id}`);
    } catch (err) {
      console.error("Bookmark error:", err);
      isBookmarked = originalBookmarked;
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
  }

  // TODO: Add Reply and Repost handlers
</script>

<article class="thread-card" aria-labelledby="thread-author-{thread.id}">
    <div class="thread-avatar">
        <div class="avatar-img">
            {#if author?.profile_picture}
              <img src="{author.profile_picture}" alt="{author.name}'s avatar" class="avatar-img" />  
            {:else if author}
                <div class="avatar-initials-placeholder">
                    {author.name?.charAt(0)?.toUpperCase() ?? '?'}
                </div>
            {:else}
                <div class="avatar-initials-placeholder">?</div>
            {/if}
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
                <MessageSquare size={18} />
                <span>{thread.reply_count > 0 ? thread.reply_count : ''}</span>
            </button>
             <button class="action-btn repost" aria-label="Repost">
                <Repeat2 size={18} />
                <span>{thread.repost_count > 0 ? thread.repost_count : ''}</span>
            </button>
             <button class="action-btn like" class:liked={isLiked} on:click={handleLike} aria-pressed={isLiked} aria-label={isLiked ? 'Unlike' : 'Like'}>
                <Heart size={18} fill={isLiked ? '#f91880' : 'none'} stroke={isLiked ? '#f91880' : 'currentColor'} />
                 <span>{likeCount > 0 ? likeCount : ''}</span>
            </button>
             <button class="action-btn bookmark" class:bookmarked={isBookmarked} on:click={handleBookmark} aria-pressed={isBookmarked} aria-label={isBookmarked ? 'Remove bookmark' : 'Bookmark'}>
                 <Bookmark size={18} fill={isBookmarked ? 'var(--primary-color)' : 'none'} stroke={isBookmarked ? 'var(--primary-color)' : 'currentColor'} />
                 <span>{bookmarkCount > 0 ? bookmarkCount : ''}</span>
            </button>
            <!-- Share Button Placeholder -->
             <button class="action-btn share" aria-label="Share">
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
    width: 40px;
    height: 40px;

    .avatar-img {
        width: 100%;
        height: 100%;
        border-radius: 50%;
        object-fit: cover;
        background-color: var(--border-color); /* Fallback bg while image loads */
    }
     .avatar-initials-placeholder {
          width: 100%; height: 100%; border-radius: 50%; background-color: var(--secondary-text-color);
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
      gap: 5px;
      font-size: 13px;
      padding: 6px;
      border-radius: 50%;
      transition: color 0.2s ease, background-color 0.2s ease;

      svg {
          width: 18px;
          height: 18px;
          fill: currentColor;
      }

      span {
          line-height: 1;
      }

      /* Specific hover colors */
    &.reply:hover { color: var(--primary-color); background-color: rgba(var(--primary-color-rgb, 29, 155, 240), 0.1); }
    &.repost:hover { color: #00ba7c; background-color: rgba(0, 186, 124, 0.1); }
    &.like:hover { color: #f91880; background-color: rgba(249, 24, 128, 0.1); }
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