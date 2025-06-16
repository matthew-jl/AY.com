<script lang="ts">
    import type { SharedThreadContent } from '../lib/api';
    import { link } from 'svelte-routing';
  
    export let sharedContent: SharedThreadContent;
  
    const authorName = sharedContent.thread_author_name || 'User';
    const authorUsername = sharedContent.thread_author_username || 'unknown';
    const authorAvatar = sharedContent.thread_author_avatar;
  </script>
  
  <a href="/thread/{sharedContent.thread_id}" use:link class="shared-thread-bubble" aria-label="View shared thread by @{authorUsername}">
    <div class="shared-thread-header">
      <div class="shared-author-avatar">
        {#if authorAvatar}
          <img src={authorAvatar} alt="{authorName}'s avatar" />
        {:else}
          <div class="avatar-initials-shared">{authorName.charAt(0).toUpperCase()}</div>
        {/if}
      </div>
      <div class="shared-author-info">
        <span class="shared-author-name">{authorName}</span>
        <span class="shared-author-handle">@{authorUsername}</span>
      </div>
    </div>
    <p class="shared-content-snippet">{sharedContent.thread_content_snippet}</p>
    {#if sharedContent.thread_first_media_thumbnail}
      <div class="shared-media-thumbnail">
          <img src={sharedContent.thread_first_media_thumbnail} alt="Thread media preview"/>
      </div>
    {/if}
    <span class="view-thread-prompt">View Thread →</span>
  </a>
  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .shared-thread-bubble {
      display: block;
      border: 1px solid var(--border-color);
      border-radius: 12px;
      padding: 12px;
      margin-top: 6px;
      margin-bottom: 4px;
      text-decoration: none;
      color: var(--text-color);
      background-color: var(--background);
      transition: background-color 0.15s ease;
  
      &:hover {
        background-color: var(--section-hover-bg);
      }
    }
  
    .shared-thread-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;
    }
  
    .shared-author-avatar {
      width: 24px; height: 24px; border-radius: 50%; overflow: hidden;
      background-color: var(--secondary-text-color);
      img { width: 100%; height: 100%; object-fit: cover; }
    }
    .avatar-initials-shared {
        width: 100%; height: 100%; display: flex;
        align-items: center; justify-content: center;
        font-size: 0.8rem; font-weight: bold; color: var(--background);
    }
  
    .shared-author-info {
      display: flex;
      align-items: baseline;
      gap: 4px;
      font-size: 14px;
      .shared-author-name { font-weight: 600; }
      .shared-author-handle { color: var(--secondary-text-color); font-size: 13px;}
    }
  
    .shared-content-snippet {
      font-size: 14px;
      line-height: 1.3;
      color: var(--secondary-text-color);
      margin-bottom: 8px;
      display: -webkit-box;
      -webkit-line-clamp: 3;
      -webkit-box-orient: vertical;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  
    .shared-media-thumbnail {
        width: 100%;
        aspect-ratio: 16 / 9;
        border-radius: 8px;
        overflow: hidden;
        background-color: var(--border-color);
        margin-bottom: 8px;
        img { width: 100%; height: 100%; object-fit: cover; }
    }
  
    .view-thread-prompt {
        display: block;
        text-align: right;
        font-size: 13px;
        font-weight: 500;
        color: var(--primary-color);
    }
  </style>