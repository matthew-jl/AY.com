<script lang="ts">
  import type { MessageData } from '../lib/api';
  import { user as currentUserStore } from '../stores/userStore';
  import { linkifyContent } from '../lib/utils/richText';
  import { getTimeFromProtoTimestamp, timeAgo } from '../lib/utils/timeAgo';
  import { createEventDispatcher } from 'svelte';
  import { Trash } from 'lucide-svelte';
  
  export let message: MessageData;
  
  $: isOwn = message.sender_id === $currentUserStore?.id;
  $: senderName = isOwn ? 'You' : message.sender_summary?.name || 'Unknown User';
  $: linkifiedMessageContent = linkifyContent(message.content);

  const dispatch = createEventDispatcher<{unsend: number}>(); // number is message.id

  const UNSEND_WINDOW_MS = 60 * 1000; // 1 minute
  let canUnsend = false;
  let showUnsendOption = false;

  $: {
      if (isOwn && message && !message.is_deleted) {
          const sentTime = getTimeFromProtoTimestamp(message.sent_at);
          canUnsend = (Date.now() - sentTime) < UNSEND_WINDOW_MS;
      } else {
          canUnsend = false;
      }
  }

  function handleUnsend() {
      if (canUnsend && confirm("Unsend this message? This will delete it for everyone.")) {
          dispatch('unsend', message.id);
      }
      showUnsendOption = false;
  }
</script>
  
  <div class="message-bubble-wrapper" class:own-message={isOwn}
    role="group"
    on:mouseenter={() => { if (canUnsend) showUnsendOption = true; }}
    on:mouseleave={() => showUnsendOption = false}
    on:focusin={() => { if (canUnsend) showUnsendOption = true; }}
    on:focusout={() => setTimeout(() => showUnsendOption = false, 100)}
  >
    {#if !isOwn && message.sender_summary}
      <div class="sender-avatar-bubble">
        {#if message.sender_summary.profile_picture_url}
          <img src={message.sender_summary.profile_picture_url} alt={senderName} />
        {:else}
          <div class="avatar-initials-bubble">{senderName.charAt(0).toUpperCase()}</div>
        {/if}
      </div>
    {/if}
  
    <div class="message-bubble">
      {#if !isOwn && message.sender_summary}
        <p class="sender-name">{senderName}</p>
      {/if}
  
      {#if message.content}
          <p class="message-text">{@html linkifiedMessageContent}</p>
      {/if}
  
      <!-- Display Media -->
      {#if message.media_items && message.media_items.length > 0}
          <div class="message-media-grid count-{message.media_items.length}">
              {#each message.media_items as media (media.id)}
                  <div class="msg-media-item">
                      {#if media.mime_type.startsWith('image/')}
                          <img src={media.public_url} alt="Message media {media.id}" loading="lazy"/>
                          <video controls preload="metadata" src={media.public_url}>
                            <track kind="captions" label="No captions" src="" default />
                          </video>
                          <video controls preload="metadata" src={media.public_url}></video>
                      {:else} <div class="file-placeholder"><span>{media.mime_type}</span></div> {/if}
                  </div>
              {/each}
          </div>
      {/if}
      <span class="message-timestamp">{timeAgo(message.sent_at)}</span>
      {#if canUnsend && showUnsendOption}
        <button class="unsend-btn" on:click|stopPropagation={handleUnsend} title="Unsend message">
            <Trash size={16} />
        </button>
      {/if}
    </div>
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .message-bubble-wrapper {
      display: flex;
      margin-bottom: 8px;
      max-width: 75%;
      align-self: flex-start;
      gap: 8px;
  
      &.own-message {
        align-self: flex-end;
    
        .message-bubble {
          background-color: var(--primary-color);
          color: var(--primary-button-text);
          border-radius: 18px 18px 4px 18px;
           .sender-name { display: none; }
           .message-timestamp { color: rgba(255,255,255,0.7); }
        }
      }
    }
  
    .sender-avatar-bubble {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      flex-shrink: 0;
      align-self: flex-end;
      overflow: hidden;
      background-color: var(--secondary-text-color);
      img { width: 100%; height: 100%; object-fit: cover; }
    }
    .avatar-initials-bubble {
      width: 100%; height: 100%; display: flex;
      align-items: center; justify-content: center;
      font-size: 1rem; font-weight: bold; color: var(--background);
    }
  
    .message-bubble {
      position: relative;
      padding: 8px 12px;
      border-radius: 18px 18px 18px 4px;
      background-color: var(--section-bg);
      color: var(--text-color);
      word-wrap: break-word;
      min-width: 50px;
  
      .sender-name {
        font-size: 0.8rem;
        font-weight: bold;
        color: var(--primary-color);
        margin-bottom: 2px;
      }
      .message-text {
        margin: 0 0 4px;
        line-height: 1.4;
        white-space: pre-wrap;
         :global(a) {
             color: var(--primary-color);
             text-decoration: none;
             &:hover { text-decoration: underline; }
              .own-message & {
                  color: var(--primary-button-text);
                  text-decoration: underline;
              }
         }
      }
      .message-timestamp {
        font-size: 0.75rem;
        color: var(--secondary-text-color);
        text-align: right;
        display: block;
        margin-top: 4px;
      }
    }
  
    .message-media-grid {
      display: grid;
      gap: 4px;
      margin-top: 8px;
      border-radius: 12px;
      overflow: hidden;
  
      &.count-1 { grid-template-columns: minmax(0, 1fr); }
      /* TODO: Add styles for count-2, count-3, count-4 if allowing multiple media per message */
      .msg-media-item {
        aspect-ratio: 16 / 9;
        background-color: var(--border-color);
        border-radius: 8px;
        overflow: hidden;
        img, video { width: 100%; height: 100%; object-fit: cover; }
        .file-placeholder { /* ... */ }
      }
    }

    .unsend-btn {
      position: absolute;
      top: -8px;
      right: -8px;
      background-color: var(--error-bg);
      color: var(--error-color);
      border: 1px solid var(--error-color);
      border-radius: 50%;
      width: 24px;
      height: 24px;
      font-size: 12px;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      box-shadow: 0 1px 3px rgba(0,0,0,0.1);
      z-index: 5;
    }
   .own-message .unsend-btn {
       left: -8px; right: auto;
   }
  </style>