<script lang="ts">
  import { createEventDispatcher } from 'svelte';
    import type { ChatData } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
  
    export let chat: ChatData;
    export let isActive: boolean = false;

    const dispatch = createEventDispatcher<{delete: number}>(); // number is chat.id
    let showOptionsMenu = false;

    function handleDeleteChat() {
      if (confirm(`Are you sure you want to delete/hide your conversation with ${displayName}? This cannot be undone for you.`)) {
          dispatch('delete', chat.id);
      }
      showOptionsMenu = false;
    }
  
    // Determine display details (already done in MessagesPage, but good to have here for standalone use)
    $: otherParticipant = chat.type === 'direct' && $currentUserStore
      ? chat.participants.find(p => p.id !== $currentUserStore?.id)
      : null;
  
    $: displayName = chat.type === 'direct'
      ? otherParticipant?.name || chat.name || 'Chat' // Fallback to chat.name if otherParticipant not found
      : chat.name || 'Group Chat';
  
    $: displayAvatarUrl = chat.type === 'direct'
      ? otherParticipant?.profile_picture_url || null
      : null; // TODO: Group chat avatar logic
  
    $: avatarInitial = displayName?.charAt(0)?.toUpperCase() ?? '?';
  
    $: lastMessagePreview = chat.last_message
      ? `${chat.last_message.sender_id === $currentUserStore?.id ? 'You: ' : (chat.last_message.sender_summary?.name ? chat.last_message.sender_summary.name + ': ' : '')}${
          chat.last_message.content
            ? (chat.last_message.content.substring(0, 25) + (chat.last_message.content.length > 25 ? '...' : ''))
            : (chat.last_message.media_items && chat.last_message.media_items.length > 0 ? 'Sent media' : '...')
        }`
      : 'No messages yet';
  
  </script>
  
  <div class="chat-list-item" class:active={isActive} role="button" tabindex="0" on:contextmenu|preventDefault={() => showOptionsMenu =!showOptionsMenu}>
    <div class="chat-item-avatar">
      {#if displayAvatarUrl}
        <img src={displayAvatarUrl} alt="{displayName}'s avatar" />
      {:else}
        <div class="avatar-initials-chat">{avatarInitial}</div>
      {/if}
    </div>
    <div class="chat-item-info">
      <span class="chat-name">{displayName}</span>
      <p class="last-message-preview">{@html lastMessagePreview}</p> <!-- Use @html if preview might contain simple styled text from sender name -->
    </div>
    {#if chat.unread_count && chat.unread_count > 0 && !isActive}
      <span class="unread-badge">{chat.unread_count}</span>
    {/if}

    <button class="more-chat-options" on:click|stopPropagation={() => showOptionsMenu = !showOptionsMenu} aria-label="More options for chat with {displayName}">⋮</button>
    {#if showOptionsMenu}
      <div class="options-menu" role="menu">
          <button role="menuitem" on:click|stopPropagation={handleDeleteChat}>Delete Chat</button>
      </div>
    {/if}
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .chat-list-item {
      position: relative;
      display: flex;
      align-items: center;
      padding: 10px 16px;
      cursor: pointer;
      border-bottom: 1px solid var(--border-color);
      gap: 10px;
      transition: background-color 0.15s ease-in-out;
  
      &:hover {
        background-color: var(--section-hover-bg);
      }
      &.active {
        background-color: var(--primary-color-light, rgba(var(--primary-color-rgb, 29, 155, 240), 0.1));
        border-right: 3px solid var(--primary-color); // Or some other active indicator
      }
    }
  
    .chat-item-avatar {
      width: 48px;
      height: 48px;
      border-radius: 50%;
      flex-shrink: 0;
      overflow: hidden;
      background-color: var(--secondary-text-color); 
      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }
    .avatar-initials-chat {
      width: 100%; height: 100%; display: flex;
      align-items: center; justify-content: center;
      font-size: 1.5rem; font-weight: bold; color: var(--background);
    }
  
    .chat-item-info {
      flex-grow: 1;
      overflow: hidden; 
      .chat-name {
        font-weight: 600; 
        font-size: 15px;
        display: block;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        color: var(--text-color);
        margin-bottom: 2px;
      }
      .last-message-preview {
        font-size: 14px;
        color: var(--secondary-text-color);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        margin: 0;
         :global(strong) { 
             font-weight: 500;
             color: var(--text-color);
         }
      }
    }
  
    .unread-badge {
      background-color: var(--primary-color);
      color: white;
      font-size: 0.7rem;
      font-weight: bold;
      padding: 3px 7px;
      border-radius: 10px;
      margin-left: auto;
      flex-shrink: 0;
    }

    .more-chat-options {
    background: none; border: none; color: var(--secondary-text-color);
    font-size: 1.2rem; cursor: pointer; padding: 4px 8px;
    border-radius: 50%; margin-left: auto;
    &:hover { background-color: var(--section-hover-bg); }
  }
  .options-menu {
    position: absolute;
    top: 35px;
    right: 10px;
    background-color: var(--background);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0,0,0,0.1);
    z-index: 10;
    min-width: 150px;
    button {
        display: block; width: 100%; text-align: left;
        padding: 8px 12px; background: none; border: none;
        color: var(--text-color); cursor: pointer;
        font-size: 14px;
        &:hover { background-color: var(--section-hover-bg); }
    }
  }
  </style>