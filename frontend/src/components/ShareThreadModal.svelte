<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import type { ThreadData, ChatData } from '../lib/api';
    import { api, ApiError } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
  
    export let threadToShare: ThreadData;
  
    const dispatch = createEventDispatcher<{ close: void; sent: { chatIds: number[] } }>();
  
    let userChats: ChatData[] = [];
    let isLoadingChats = true;
    let selectedChatIds: number[] = [];
    let sending = false;
    let error: string | null = null;
    let successMessage: string | null = null;
  
    onMount(async () => {
      console.log("Fetching user chats for share modal...");
      isLoadingChats = true;
      try {
        const response = await api.getUserChats(1, 100);
        console.log("Fetched user chats for share modal:", response);
        userChats = response.chats || [];
      } catch (err) {
        console.error("Error fetching chats for share modal:", err);
        error = "Could not load your chats.";
      } finally {
        isLoadingChats = false;
      }
    });
  
    function toggleChatSelection(chatId: number) {
      if (selectedChatIds.includes(chatId)) {
        selectedChatIds = selectedChatIds.filter(id => id !== chatId);
      } else {
        selectedChatIds = [...selectedChatIds, chatId];
      }
    }

    function getChatDisplayInfo(chat: ChatData) {
      // For direct chats, find the other participant
      if (chat.type === 'direct' && $currentUserStore) {
        const otherParticipant = chat.participants.find(p => p.id !== $currentUserStore.id);
        
        if (otherParticipant) {
          return {
            name: otherParticipant.name || `User ${otherParticipant.id}`,
            avatarUrl: otherParticipant.profile_picture_url,
            initial: otherParticipant.name?.charAt(0)?.toUpperCase() || '?'
          };
        }
      }
      
      // For group chats or fallback
      return {
        name: chat.name || `Chat ${chat.id}`,
        avatarUrl: null, // Group chat avatar logic could be added here if available
        initial: (chat.name?.charAt(0) || 'G').toUpperCase()
      };
    }
  
    async function handleSendThread() {
      if (selectedChatIds.length === 0) {
        error = "Please select at least one chat to send to.";
        return;
      }
      sending = true;
      error = null;
      successMessage = null;
  
      const sharedThreadContentPayload = {
        is_shared_thread: true,
        thread_id: threadToShare.id,
        thread_author_name: threadToShare.author?.name || 'Unknown User',
        thread_author_username: threadToShare.author?.username || 'unknown',
        thread_author_avatar: threadToShare.author?.profile_picture || null,
        thread_content_snippet: threadToShare.content.substring(0, 100) + (threadToShare.content.length > 100 ? '...' : ''),
        // TODO: Get first media item thumbnail if available
        // thread_first_media_thumbnail: threadToShare.media?.[0]?.public_url || null,
      };
  
      const messagePayload = {
          content: JSON.stringify(sharedThreadContentPayload),
          type: 'shared_thread' as 'shared_thread',
      };
  
      try {
        const sendPromises = selectedChatIds.map(chatId =>
          api.sendMessageToChat(chatId, messagePayload)
        );
        await Promise.all(sendPromises);
        successMessage = `Thread sent to ${selectedChatIds.length} chat(s)!`;
        setTimeout(() => {
          dispatch('sent', { chatIds: selectedChatIds });
          dispatch('close');
        }, 1500);
      } catch (err) {
        console.error("Error sending thread to chats:", err);
        if (err instanceof ApiError) { error = `Failed to send: ${err.message}`; }
        else { error = "An error occurred while sending."; }
      } finally {
        sending = false;
      }
    }
  </script>
  
  <div class="modal-overlay" on:click={() => dispatch('close')}>
    <div class="modal-content share-thread-modal" on:click|stopPropagation>
      <header class="modal-header-simple">
        <h3>Send Thread via Direct Message</h3>
        <button class="close-button-simple" on:click={() => dispatch('close')}>×</button>
      </header>

      <div class="share-content">
        {#if isLoadingChats}
          <p>Loading your chats...</p>
        {:else if error && userChats.length === 0}
          <p class="error-text">{error}</p>
        {:else if userChats.length === 0}
          <p>You have no active chats to send this to.</p>
        {:else}
          <p>Select chats to send this thread to:</p>
          <div class="chat-selection-list">
            {#each userChats as chat (chat.id)}
              {@const displayInfo = getChatDisplayInfo(chat)}
              <label class="chat-select-item" class:selected={selectedChatIds.includes(chat.id)}>
                <input
                  type="checkbox"
                  value={chat.id}
                  checked={selectedChatIds.includes(chat.id)}
                  on:change={() => toggleChatSelection(chat.id)}
                />
                <!-- Chat avatar -->
                <div class="chat-item-avatar small">
                  {#if displayInfo.avatarUrl}
                    <img src={displayInfo.avatarUrl} alt="{displayInfo.name}'s avatar" />
                  {:else}
                    <div class="avatar-initials-chat small">{displayInfo.initial}</div>
                  {/if}
                </div>
                <!-- Chat name with type indicator for group chats -->
                <span class="chat-name">
                  {displayInfo.name}
                  {#if chat.type === 'group'}
                    <span class="chat-type-indicator">Group</span>
                  {/if}
                </span>
              </label>
            {/each}
          </div>
        {/if}

        {#if error && userChats.length > 0} <p class="error-text api-error">{error}</p> {/if}
        {#if successMessage} <p class="success-text">{successMessage}</p> {/if}
      </div>

      <footer class="modal-footer">
        <button class="btn btn-secondary" on:click={() => dispatch('close')} disabled={sending}>Cancel</button>
        <button class="btn btn-primary" on:click={handleSendThread} disabled={sending || selectedChatIds.length === 0 || isLoadingChats}>
          {sending ? 'Sending...' : 'Send'}
        </button>
      </footer>
    </div>
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
    
    .modal-overlay {
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      background-color: rgba(0, 0, 0, 0.6);
      display: flex;
      justify-content: center;
      align-items: flex-start;
      padding-top: 5vh;
      z-index: 1000;
    }
    
    .modal-content.share-thread-modal {
      background: var(--background);
      color: var(--text-color);
      border-radius: 16px;
      width: 90%;
      max-width: 450px;
      max-height: 80vh;
      display: flex;
      flex-direction: column;
      box-shadow: 0 5px 20px rgba(0,0,0,0.2);
    }
    
    .modal-header-simple {
      display: flex;
      align-items: center;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      
      h3 {
        flex-grow: 1;
        text-align: left;
        margin: 0;
        font-size: 1.2rem;
        font-weight: bold;
      }
      
      .close-button-simple {
        background: transparent;
        border: none;
        font-size: 1.8rem;
        cursor: pointer;
        color: var(--text-color);
        padding: 0 8px;
      }
    }
    
    .share-content {
      padding: 16px;
      flex-grow: 1;
      overflow-y: auto;
      
      p {
        margin-bottom: 12px;
        color: var(--secondary-text-color);
      }
    }
    
    .chat-selection-list {
      max-height: 300px;
      overflow-y: auto;
      border: 1px solid var(--border-color);
      border-radius: 8px;
      margin-top: 10px;
      scrollbar-width: none;
      
      &::-webkit-scrollbar {
        display: none;
      }
    }
    
    .chat-select-item {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 10px 12px;
      cursor: pointer;
      border-bottom: 1px solid var(--border-color);
      
      &:last-child {
        border-bottom: none;
      }
      
      &:hover {
        background-color: var(--section-hover-bg);
      }
      
      &.selected {
        background-color: rgba(var(--primary-color-rgb), 0.05);
        font-weight: 500;
      }
      
      input[type="checkbox"] {
        accent-color: var(--primary-color);
        margin-right: 0;
      }
    }
    
    .chat-item-avatar {
      width: 48px;
      height: 48px;
      border-radius: 50%;
      flex-shrink: 0;
      overflow: hidden;
      background-color: var(--secondary-text-color);
      
      &.small {
        width: 32px;
        height: 32px;
      }
      
      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }
    
    .avatar-initials-chat {
      width: 100%;
      height: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: bold;
      color: var(--background);
      
      &.small {
        font-size: 1rem;
      }
    }
    
    .modal-footer {
      padding: 12px 16px;
      border-top: 1px solid var(--border-color);
      display: flex;
      justify-content: flex-end;
      gap: 10px;
      
      .btn {
        margin-top: 0;
        width: auto;
        padding: 8px 20px;
        border-radius: 9999px;
        font-weight: bold;
        font-size: 14px;
        cursor: pointer;
        border: 1px solid transparent;
        transition: background-color 0.2s ease;
      }
      
      .btn-secondary {
        background-color: transparent;
        color: var(--primary-color);
        border-color: var(--primary-color);
        
        &:hover:not(:disabled) {
          background-color: var(--sidebar-hover-bg);
        }
        
        &:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }
      }
      
      .btn-primary {
        background-color: var(--follow-button-bg);
        color: var(--follow-button-text);
        border-color: var(--follow-button-border);
        
        &:hover:not(:disabled) {
          background-color: var(--follow-button-hover-bg);
        }
        
        &:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }
      }
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
    
    .success-text {
      color: var(--success-color);
      background-color: var(--success-bg);
      padding: 0.8rem;
      border-radius: 6px;
      text-align: center;
      margin-top: 1rem;
      font-weight: bold;
    }
    
    @media (max-width: 576px) {
      .modal-content.share-thread-modal {
        width: 95%;
        max-height: 85vh;
      }
      
      .chat-item-avatar.small {
        width: 28px;
        height: 28px;
      }
      
      .chat-select-item {
        padding: 8px 10px;
      }
      
      .modal-footer .btn {
        padding: 6px 14px;
        font-size: 13px;
      }
    }
    
    @media (max-width: 400px) {
      .modal-header-simple h3 {
        font-size: 1.1rem;
      }
      
      .share-content {
        padding: 12px;
      }
      
      .chat-selection-list {
        max-height: 250px;
      }
    }

    .chat-name {
      display: flex;
      flex-direction: column;
      overflow: hidden;
      
      .chat-type-indicator {
        font-size: 0.7rem;
        color: var(--secondary-text-color);
        font-weight: normal;
      }
    }
  </style>