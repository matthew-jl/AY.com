<script lang="ts">
    import { onMount, tick } from 'svelte';
    import { api, ApiError, type ChatData, type MessageData, type UserSummary, type MediaMetadata, type SendMessageRequestData, type UserProfileBasic, type SearchUsersApiResponse, type CreateGroupChatRequestData, type AddParticipantRequestData } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
    import { getAccessToken } from '../lib/api';
    import ChatListItem from '../components/ChatListItem.svelte';
    import MessageBubble from '../components/MessageBubble.svelte';
    import MessageInput from '../components/MessageInput.svelte';
    import { timeAgo } from '../lib/utils/timeAgo';
  import { EllipsisVertical, Option, Pencil, Search, Users } from 'lucide-svelte';
  
    let chats: ChatData[] = [];
    let selectedChat: ChatData | null = null;
    let messages: MessageData[] = [];
    let isLoadingChats = true;
    let isLoadingMessages = false;
    let chatError: string | null = null;
    let messageError: string | null = null;
  
    // MessageInput component
    let currentMessageContent = '';
    let currentSelectedFiles: FileList | null = null;
    let currentMediaPreviews: { url: string; file: File; type: 'image' | 'video' | 'other' }[] = [];
    let isSendingMessage = false;

    // New Chat Modal/UI
    let showNewChatModal = false;
    let newChatSearchQuery = '';
    let newChatUserResults: UserProfileBasic[] = [];
    let newChatLoadingUsers = false;
    let newChatError: string | null = null;
    let newChatDebounceTimer: number | undefined;

    let showMessageSearch = false;
    let messageSearchQuery = '';

    let displayedMessagesToList: MessageData[] = [];

    $: {
      if (messageSearchQuery.trim() === '') {
        displayedMessagesToList = messages; // Show all messages if no search
      } else {
        displayedMessagesToList = messages.filter(msg =>
          !msg.is_deleted &&
          msg.content?.toLowerCase().includes(messageSearchQuery.toLowerCase().trim())
        );
      }
      if (messageSearchQuery.trim() === '') {
          tick().then(scrollToBottom);
      }
    }

    // Create Group Chat Modal/UI
    let showCreateGroupModal = false;
    let newGroupName = '';
    let newGroupSearchQuery = '';
    let newGroupUserResults: UserProfileBasic[] = [];
    let selectedParticipantsForGroup: UserProfileBasic[] = []; 
    let newGroupLoadingUsers = false;
    let newGroupError: string | null = null;
    let newGroupDebounceTimer: number | undefined;

    // Manage Participants Modal
    let showManageParticipantsModal = false;
    let chatToManage: ChatData | null = null;
    let addUserSearchQuery = '';
    let addUserResults: UserProfileBasic[] = [];
    let addUserLoading = false;
    let manageParticipantsError: string | null = null;
  
    let ws: WebSocket | null = null;
    let messageListElement: HTMLElement;
  
    onMount(() => {
      fetchUserChats();
      connectWebSocket();
  
      return () => {
        if (ws) ws.close();
      };
    });
  
    async function fetchUserChats() {
      isLoadingChats = true; chatError = null;
      try {
        const response = await api.getUserChats(1, 50); // Fetch more chats initially
        chats = (response.chats || []).map(chat => ({
            ...chat,
            // Determine display name and avatar for direct chats client-side
            display_name: chat.type === 'direct' && $currentUserStore
                ? chat.participants.find(p => p.id !== $currentUserStore?.id)?.name || 'Unknown User'
                : chat.name || 'Group Chat',
            display_avatar: chat.type === 'direct' && $currentUserStore
                ? chat.participants.find(p => p.id !== $currentUserStore?.id)?.profile_picture_url || null
                : null, // TODO: Group chat avatar logic
        }));
      } catch (err) {
        console.error("Error fetching chats:", err);
        chatError = "Could not load your chats.";
      } finally {
        isLoadingChats = false;
      }
    }
  
    async function selectChat(chat: ChatData) {
      if (selectedChat?.id === chat.id && messages.length > 0) return; // Already selected and loaded
  
      console.log("Selecting chat:", chat.id, chat.display_name);
      selectedChat = chat;
      messages = []; // Clear previous messages
      isLoadingMessages = true;
      messageError = null;
  
      try {
        const response = await api.getMessagesForChat(chat.id, 1, 50); // Load more messages initially
        messages = (response.messages || []).reverse(); // Reverse to show oldest first, newest at bottom
        // console.log("messages:" , messages);
        await tick(); // Wait for DOM to update
        scrollToBottom();
        // TODO: Send "subscribe_chat" message over WebSocket
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'subscribe_chat', chat_id: chat.id }));
        }
      } catch (err) {
        console.error(`Error fetching messages for chat ${chat.id}:`, err);
        messageError = "Could not load messages for this chat.";
      } finally {
        isLoadingMessages = false;
      }

      if (selectedChat) {
        const chatIndex = chats.findIndex(c => c.id === selectedChat!.id);
        if (chatIndex > -1 && chats[chatIndex].unread_count) {
            chats[chatIndex].unread_count = 0;
            chats = [...chats];
        }
    }
    }
  
    // --- WebSocket Handling ---
    function connectWebSocket() {
      const token = getAccessToken();
      if (!token || !$currentUserStore) return;
  
      const wsUrl = `ws://localhost:8082/ws/messages?token=${token}`; // Message service WS
      console.log("Attempting to connect to Message WebSocket:", wsUrl);
      ws = new WebSocket(wsUrl);
  
      ws.onopen = () => console.log("Message WebSocket connected.");
      ws.onmessage = (event) => {
        console.log("WS Message received:", event.data);
        try {
          const data = JSON.parse(event.data as string);

          if (data.id && data.chat_id && data.sender_id && data.sent_at) { // raw message data
              const newMsg = data as MessageData;
              if (selectedChat && newMsg.chat_id === selectedChat.id) {
                  messages = [...messages, newMsg];
                  console.log("New message in selected chat:", newMsg);
                  console.log("Current messages:", messages);
                  tick().then(scrollToBottom);
              }
              // Update chat list (last message, unread count, sort)
              const chatIndex = chats.findIndex(c => c.id === newMsg.chat_id);
            if (chatIndex > -1) {
                const updatedChat = { ...chats[chatIndex], last_message: newMsg, updated_at: newMsg.sent_at };
                // For direct chats, re-evaluate display name if last message sender is relevant
                if (updatedChat.type === 'direct' && $currentUserStore && newMsg.sender_summary) {
                    const otherParticipant = updatedChat.participants.find(p => p.id !== $currentUserStore?.id);
                    if(otherParticipant) {
                        updatedChat.display_name = otherParticipant.name;
                        updatedChat.display_avatar = otherParticipant.profile_picture_url;
                    } else if (newMsg.sender_id !== $currentUserStore.id && newMsg.sender_summary) {
                        // If otherParticipant not found but message came from them
                        updatedChat.display_name = newMsg.sender_summary.name;
                        updatedChat.display_avatar = newMsg.sender_summary.profile_picture_url;
                    }
                }
    
    
                chats[chatIndex] = updatedChat;
                // Bring chat to top by sorting
                chats = [...chats].sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());
    
                if (selectedChat?.id !== newMsg.chat_id) {
                    // Increment unread count for non-active chat
                    chats[chatIndex].unread_count = (chats[chatIndex].unread_count || 0) + 1;
                }
            }
          } else if (data.type && data.type === 'message_deleted' && data.chat_id && data.message_id) {
              console.log("WS: message_deleted event received", data);
              if (selectedChat && data.chat_id === selectedChat.id) {
                  messages = messages.map(m =>
                      m.id === data.message_id
                          ? { ...m, content: "Message deleted", is_deleted: true, media_items: [] }
                          : m
                  );
              }
              // Update last_message in chats list if this was the last one
              const chatIndex = chats.findIndex(c => c.id === data.chat_id);
              if (chatIndex > -1 && chats[chatIndex].last_message?.id === data.message_id) {
                  // Refetch last message for this chat or mark as "Message deleted"
                  // For simplicity now, just nullify. A robust solution would refetch.
                  chats[chatIndex].last_message = null; // Or a placeholder
                  chats = [...chats];
              }
          }
        } catch (e) { console.error("Error parsing WS message:", e); }
      };
      ws.onerror = (err) => console.error("Message WebSocket error:", err);
      ws.onclose = () => { console.log("Message WebSocket closed."); ws = null; /* TODO: Reconnect logic */ };
    }
  
    async function handleSendMessageFromInput(event: CustomEvent<{ content: string; filesToUpload: File[] }>) {
      const { content, filesToUpload } = event.detail;
      if ((!content && filesToUpload.length === 0) || !selectedChat || !$currentUserStore) return;

      isSendingMessage = true;
      messageError = null;

      let uploadedMediaIDs: number[] = [];
      try {
        if (filesToUpload.length > 0) {
          console.log("Uploading media for message...");
          const uploadPromises = filesToUpload.map(async (file) => {
            const formData = new FormData(); formData.append('media_file', file);
            const response = await api.uploadMedia(formData);
            return response.media.id;
          });
          uploadedMediaIDs = await Promise.all(uploadPromises);
        }

        const messageData: SendMessageRequestData = {
          content: content,
          media_ids: uploadedMediaIDs.length > 0 ? uploadedMediaIDs : undefined,
        };

        await api.sendMessageToChat(selectedChat.id, messageData);
        // Clear input fields in MessageInput after successful send
        currentMessageContent = '';
        currentMediaPreviews = [];  
        currentSelectedFiles = null;

      } catch (err) { messageError = "Failed to send message."; }
      finally { isSendingMessage = false; }
    }
  
    function scrollToBottom() {
      if (messageListElement) {
        messageListElement.scrollTop = messageListElement.scrollHeight;
      }
    }
  
    function handleInputFilesSelected(event: CustomEvent<FileList>) {
    currentSelectedFiles = event.detail;
    currentMediaPreviews = []; // Clear previous
    if (currentSelectedFiles) {
      Array.from(currentSelectedFiles).forEach(file => {
        const url = URL.createObjectURL(file);
        let type: 'image' | 'video' | 'other' = 'other';
        if (file.type.startsWith('image/')) type = 'image';
        else if (file.type.startsWith('video/')) type = 'video';
        currentMediaPreviews = [...currentMediaPreviews, { url, file, type }];
      });
    }
  }

  function handleInputRemovePreview(event: CustomEvent<string>) {
    const urlToRemove = event.detail;
    URL.revokeObjectURL(urlToRemove); // Revoke object URL
    currentMediaPreviews = currentMediaPreviews.filter(p => p.url !== urlToRemove);
    // Update currentSelectedFiles if necessary (more complex, for now just previews)
  }
  
  function openNewChatOptions() {
    showNewChatModal = true;
    newChatSearchQuery = '';
    newChatUserResults = [];
    newChatError = null;
  }
  function openCreateGroupFlow() {
    closeNewChat();
    showCreateGroupModal = true;
    newGroupName = ''; newGroupSearchQuery = ''; newGroupUserResults = []; selectedParticipantsForGroup = []; newGroupError = null;
  }
  function closeCreateGroupModal() { showCreateGroupModal = false; }
  function closeNewChat() { showNewChatModal = false; }

  // User search for adding to a new group OR starting a DM
  async function handleUserSearchForNewChat(eventTargetInput: HTMLInputElement, forGroup: boolean) {
    const query = eventTargetInput.value;
    if (forGroup) newGroupSearchQuery = query; else newChatSearchQuery = query;

    clearTimeout(newGroupDebounceTimer);
    newGroupDebounceTimer = window.setTimeout(async () => {
      const currentQuery = (forGroup ? newGroupSearchQuery : newChatSearchQuery).trim();
      if (currentQuery.length < 2) {
        if (forGroup) newGroupUserResults = []; else newChatUserResults = [];
        if (forGroup) newGroupError = null; else newChatError = null;
        return;
      }
      if (forGroup) newGroupLoadingUsers = true; else newChatLoadingUsers = true;
      if (forGroup) newGroupError = null; else newChatError = null;

      try {
        const response: SearchUsersApiResponse = await api.searchUsers(currentQuery, 1, 10);
        const results = (response.users || []).filter(u => u.id !== $currentUserStore?.id); // Exclude self

        if (forGroup) {
            // Filter out users already selected for the group
            newGroupUserResults = results.filter(
                searchedUser => !selectedParticipantsForGroup.some(selected => selected.id === searchedUser.id)
            );
        } else {
            newChatUserResults = results;
        }
      } catch (err) {
        console.error("New chat/group user search error:", err);
        if (forGroup) newGroupError = "Error searching users."; else newChatError = "Error searching users.";
      } finally {
        if (forGroup) newGroupLoadingUsers = false; else newChatLoadingUsers = false;
      }
    }, 500);
  }

  async function startDirectChatWithUser(userToChatWith: UserProfileBasic) {
    if (!$currentUserStore) return;
    closeNewChat(); // Close modal
    isLoadingChats = true; // Indicate loading

    try {
      // Check if a chat already exists with this user
      const existingChat = chats.find(chat =>
        chat.type === 'direct' &&
        chat.participants.length === 2 &&
        chat.participants.some(p => p.id === userToChatWith.id)
      );

      if (existingChat) {
        selectChat(existingChat);
      } else {
        const newChatData = await api.getOrCreateDirectChat({ other_user_id: userToChatWith.id });
        // The newChatData from backend might not be fully hydrated for chat list display.
        // We need to manually construct a ChatData object for the UI.
        const newChatForList: ChatData = {
            ...newChatData, // id, type, creator_id, created_at, updated_at
            participants: [ // Construct participant summaries
                { id: $currentUserStore.id, name: $currentUserStore.name, username: $currentUserStore.username, profile_picture_url: $currentUserStore.profile_picture },
                { id: userToChatWith.id, name: userToChatWith.name, username: userToChatWith.username, profile_picture_url: userToChatWith.profile_picture }
            ],
            display_name: userToChatWith.name,
            display_avatar: userToChatWith.profile_picture,
            last_message: null, // New chat has no messages
            unread_count: 0
        };
        chats = [newChatForList, ...chats]; // Add to top
        selectChat(newChatForList);
      }
    } catch (err) {
      console.error("Error starting direct chat:", err);
      chatError = "Could not start chat."; // Show error on main page
    } finally {
      isLoadingChats = false;
    }
  }

  async function handleDeleteChat(event: CustomEvent<number>) {
    const chatIdToDelete = event.detail;
    console.log("Attempting to delete chat:", chatIdToDelete);
    // Optimistic UI update
    const originalChats = [...chats];
    chats = chats.filter(c => c.id !== chatIdToDelete);
    if (selectedChat?.id === chatIdToDelete) {
        selectedChat = null;
        messages = [];
    }

    try {
        await api.deleteChat(chatIdToDelete);
        console.log("Chat hidden successfully on backend.");
    } catch (err) {
        console.error("Error deleting chat:", err);
        chats = originalChats;
        chatError = "Failed to delete chat.";
    }
  }

  async function handleUnsendMessage(event: CustomEvent<number>) {
    const messageIdToUnsend = event.detail;
    if (!selectedChat) return;
    console.log("Attempting to unsend message:", messageIdToUnsend, "in chat:", selectedChat.id);

    try {
        await api.deleteMessage(selectedChat.id, messageIdToUnsend);
        console.log("Unsend request successful for message:", messageIdToUnsend);
        // Backend will send a WS event "message_deleted"
        // OR you can optimistically remove it here:
        // messages = messages.filter(m => m.id !== messageIdToUnsend);
    } catch (err) {
        console.error("Error unsending message:", err);
        messageError = "Failed to unsend message.";
    }
  }

  function toggleMessageSearch() {
      showMessageSearch = !showMessageSearch;
      if (!showMessageSearch) {
          messageSearchQuery = ''; // Clear search when hiding
      } else {
          tick().then(() => { // Focus input when shown
              const searchInput = document.getElementById('message-search-input');
              if (searchInput) searchInput.focus();
          });
      }
  }

  function toggleParticipantForNewGroup(user: UserProfileBasic) {
    const index = selectedParticipantsForGroup.findIndex(p => p.id === user.id);
    if (index > -1) {
      selectedParticipantsForGroup = selectedParticipantsForGroup.filter(p => p.id !== user.id);
    } else {
      selectedParticipantsForGroup = [...selectedParticipantsForGroup, user];
    }
    newGroupSearchQuery = '';
    newGroupUserResults = [];
  }

  async function handleCreateGroupSubmit() {
    if (!newGroupName.trim()) {
      newGroupError = "Group name is required."; return;
    }
    isLoadingChats = true; newGroupError = null;

    const participantIds = selectedParticipantsForGroup.map(p => p.id);
    const groupData: CreateGroupChatRequestData = {
      name: newGroupName.trim(),
      initial_participant_ids: participantIds,
    };

    try {
      const newGroupChat = await api.createGroupChat(groupData);
      // Manually construct display name/avatar or rely on backend hydration of participants
      const createdChatForList: ChatData = {
          ...newGroupChat, // Contains id, type, creator_id, name, created_at, updated_at
          participants: [ // Add current user and selected participants with summary
              {id: $currentUserStore!.id, name: $currentUserStore!.name, username: $currentUserStore!.username, profile_picture_url: $currentUserStore!.profile_picture},
              ...selectedParticipantsForGroup.map(p => ({id: p.id, name: p.name, username: p.username, profile_picture_url: p.profile_picture}))
          ],
          display_name: newGroupChat.name,
          display_avatar: null, // TODO: Group avatar logic
          last_message: null,
          unread_count: 0,
      };

      chats = [createdChatForList, ...chats].sort((a,b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());
      selectChat(createdChatForList);
      closeCreateGroupModal();
    } catch (err) {
      console.error("Create group error:", err);
      if (err instanceof ApiError) { newGroupError = `Failed: ${err.message}`; }
      else { newGroupError = "Could not create group."; }
    } finally {
      isLoadingChats = false;
    }
  }

  function openManageParticipantsModal(chat: ChatData) {
      if (chat.type !== 'group') return;
      chatToManage = chat;
      addUserSearchQuery = '';
      addUserResults = [];
      manageParticipantsError = null;
      showManageParticipantsModal = true;
  }
  function closeManageParticipantsModal() { showManageParticipantsModal = false; chatToManage = null; }

  async function handleAddUserSearch() { // Search for users to add to existing group
    clearTimeout(newGroupDebounceTimer); // Reuse debounce timer
    newGroupDebounceTimer = window.setTimeout(async () => {
        if (!chatToManage || addUserSearchQuery.trim().length < 2) {
            addUserResults = []; manageParticipantsError = null; return;
        }
        addUserLoading = true; manageParticipantsError = null;
        try {
            const response = await api.searchUsers(addUserSearchQuery.trim(), 1, 10);
            addUserResults = (response.users || []).filter(u =>
                u.id !== $currentUserStore?.id // Exclude self
                // && !chatToManage!.participants.some(p => p.id === u.id) // Exclude existing members
            );
        } catch (e) { manageParticipantsError = "Error searching users."; }
        finally { addUserLoading = false; }
    }, 500);
  }

  async function addParticipantToManagedGroup(userToAdd: UserProfileBasic) {
      if (!chatToManage || !$currentUserStore) return;
      isLoadingChats = true; manageParticipantsError = null;
      try {
          const addData: AddParticipantRequestData = { target_user_id: userToAdd.id };
          await api.addParticipantToGroup(chatToManage.id, addData);
          // Optimistically update UI or refetch chat details
          chatToManage.participants = [...chatToManage.participants, {
              id: userToAdd.id, name: userToAdd.name, username: userToAdd.username, profile_picture_url: userToAdd.profile_picture
          }];
          chatToManage = {...chatToManage}; // Trigger reactivity
          addUserSearchQuery = ''; addUserResults = []; // Clear search
          // TODO: Broadcast "participant_added" via WS from backend
      } catch (err) {
          if(err instanceof ApiError) manageParticipantsError = err.message; else manageParticipantsError = "Failed to add participant.";
      } finally { isLoadingChats = false; }
  }

  async function removeParticipantFromManagedGroup(userIdToRemove: number) {
      if (!chatToManage || !$currentUserStore || !confirm(`Remove this user from the group?`)) return;
      isLoadingChats = true; manageParticipantsError = null;
      try {
          await api.removeParticipantFromGroup(chatToManage.id, userIdToRemove);
          chatToManage.participants = chatToManage.participants.filter(p => p.id !== userIdToRemove);
          chatToManage = {...chatToManage};
          // TODO: Broadcast "participant_removed" via WS
      } catch (err) {
          if(err instanceof ApiError) manageParticipantsError = err.message; else manageParticipantsError = "Failed to remove participant.";
      } finally { isLoadingChats = false; }
  }

  </script>
  
  <div class="messages-page-layout">
    <!-- Chat List Panel -->
    <aside class="chat-list-panel">
      <header>
        <h2>Messages</h2>
        <!-- TODO: New Chat Button -->
         <div>
           <button class="new-chat-btn" title="New Direct Message" aria-label="New message" on:click={openNewChatOptions}>
             <Pencil size={18} />
           </button>
           <button class="new-chat-btn" title="New Group Chat" on:click={openCreateGroupFlow}>
             <Users size={18} />
           </button>
         </div>
      </header>
      <!-- TODO: Chat Search Input -->
      <div class="chat-list">
        {#if isLoadingChats}
          <p>Loading chats...</p> <!-- TODO: Skeleton for chat list -->
        {:else if chatError}
          <p class="error-text">{chatError}</p>
        {:else if chats.length > 0}
          {#each chats as chat (chat.id)}
            <div role="button" tabindex="0" on:click={() => selectChat(chat)} on:keydown={(e) => e.key === 'Enter' && selectChat(chat)}>
              <ChatListItem {chat} isActive={selectedChat?.id === chat.id} on:delete={handleDeleteChat}/>
            </div>
          {/each}
        {:else}
          <p class="empty-list">No chats yet. Start a new conversation!</p>
        {/if}
      </div>
    </aside>
  
    <!-- Message Display Area -->
    <main class="message-display-area">
      {#if selectedChat}
        <header class="message-header">
          <div class="chat-info">
              {#if selectedChat.display_avatar}
                   <img src={selectedChat.display_avatar} alt="{selectedChat.display_name}'s avatar" class="header-avatar" />
              {:else if selectedChat.display_name}
                   <div class="avatar-initials-header">{selectedChat.display_name.charAt(0).toUpperCase()}</div>
              {/if}
              <h3>{selectedChat.display_name || `Chat ${selectedChat.id}`}</h3>
          </div>
          <div class="header-actions">
            <!-- Search Button -->
            <button class="icon-btn" on:click={toggleMessageSearch} title="Search messages in this chat">
              <Search size={18}/>
            </button>
            {#if selectedChat.type === 'group'}
                <button class="icon-btn options-btn" title="Group options" on:click={() => openManageParticipantsModal(selectedChat!)}>
                  <EllipsisVertical size={18} />
                </button>
            {/if}
        </div>
        </header>

        {#if showMessageSearch}
        <div class="message-search-input-bar">
            <input
                type="text"
                id="message-search-input"
                placeholder="Search in this chat..."
                bind:value={messageSearchQuery}
            />
            {#if messageSearchQuery}
                <button class="clear-search-btn" on:click={() => messageSearchQuery = ''}>×</button>
            {/if}
        </div>
      {/if}

        <div class="message-list" bind:this={messageListElement}>
          {#if isLoadingMessages}
            <p>Loading messages...</p> <!-- TODO: Skeleton for messages -->
          {:else if messageError}
            <p class="error-text">{messageError}</p>
          {:else if displayedMessagesToList.length > 0}
            {#each displayedMessagesToList as message (message.id)}
              {#if !message.is_deleted}
                  <MessageBubble {message} on:unsend={handleUnsendMessage} />
              {:else}
                  <div class="message-bubble-wrapper {message.sender_id === $currentUserStore?.id ? 'own-message' : ''}">
                    <div class="message-bubble deleted-message">
                        <em>Message deleted</em>
                        <span class="message-timestamp">{timeAgo(message.sent_at)}</span>
                    </div>
                  </div>
              {/if}
            {/each}
          {:else if messageSearchQuery && displayedMessagesToList.length === 0}
            <p class="empty-list">No messages found matching "{messageSearchQuery}".</p>
          {:else}
            <p class="empty-list">No messages in this chat yet. Send one!</p>
          {/if}
        </div>
        <MessageInput
          bind:currentContent={currentMessageContent}
          bind:selectedFiles={currentSelectedFiles}
          bind:mediaPreviews={currentMediaPreviews}
          isSending={isSendingMessage}
          on:send={handleSendMessageFromInput}
          on:filesselected={handleInputFilesSelected}
          on:removepreview={handleInputRemovePreview}
        />
      {:else}
        <div class="no-chat-selected">
          <h2>Select a chat to start messaging</h2>
          <p>Or, start a new conversation.</p>
          <button class="new-chat-btn" title="New message" aria-label="New message" on:click={openNewChatOptions}>
            +
          </button>
        </div>
      {/if}
    </main>
  </div>

  {#if showNewChatModal}
    <div class="modal-overlay" on:click={closeNewChat}>
      <div class="modal-content new-chat-modal" on:click|stopPropagation>
        <header class="modal-header-simple">
            <h3>New Message</h3>
            <button class="close-button-simple" on:click={closeNewChat}>×</button>
        </header>
        <div class="new-chat-search">
            <input type="text" placeholder="Search people" bind:value={newChatSearchQuery} on:input={(e) => handleUserSearchForNewChat(e.target as HTMLInputElement, false)} />
        </div>
        <div class="new-chat-results">
            {#if newChatLoadingUsers} <p>Searching...</p>
            {:else if newChatError} <p class="error-text">{newChatError}</p>
            {:else if newChatUserResults.length > 0}
                {#each newChatUserResults as userResult (userResult.id)}
                    <div class="user-result-item" on:click={() => startDirectChatWithUser(userResult)} role="button" tabindex="0">
                        <div class="chat-item-avatar">
                            {#if userResult.profile_picture} <img src={userResult.profile_picture} alt={userResult.name} />
                            {:else} <div class="avatar-initials-chat">{userResult.name.charAt(0).toUpperCase()}</div> {/if}
                        </div>
                        <div class="user-result-info">
                            <span class="user-result-name">{userResult.name}</span>
                            <span class="user-result-username">@{userResult.username}</span>
                        </div>
                    </div>
                {/each}
            {:else if newChatSearchQuery.trim().length >= 2}
                <p>No users found matching "{newChatSearchQuery}".</p>
            {/if}
        </div>
      </div>
    </div>
  {/if}

  {#if showCreateGroupModal}
    <div class="modal-overlay" on:click={closeCreateGroupModal}>
      <div class="modal-content new-chat-modal" style="max-width: 500px;" on:click|stopPropagation>
        <header class="modal-header-simple">
            <h3>Create Group Chat</h3>
            <button class="close-button-simple" on:click={closeCreateGroupModal}>×</button>
        </header>
        <form class="create-group-form" on:submit|preventDefault={handleCreateGroupSubmit}>
            <div class="form-group">
                <label for="groupName">Group Name</label>
                <input type="text" id="groupName" bind:value={newGroupName} required />
            </div>
            <div class="form-group">
                <label for="groupUserSearch">Add Participants</label>
                <input type="text" id="groupUserSearch" placeholder="Search people to add" bind:value={newGroupSearchQuery} on:input={(e) => handleUserSearchForNewChat(e.target as HTMLInputElement, true)} />
            </div>

            <!-- Search Results for New Group -->
            {#if newGroupLoadingUsers} <p>Searching...</p>
            {:else if newGroupUserResults.length > 0}
                <ul class="user-select-list">
                {#each newGroupUserResults as userRes (userRes.id)}
                    <li on:click={() => toggleParticipantForNewGroup(userRes)} role="button">
                        <span>{userRes.name} (@{userRes.username})</span>
                        <span>➕</span>
                    </li>
                {/each}
                </ul>
            {/if}
            {#if newGroupError} <p class="error-text">{newGroupError}</p> {/if}


            <!-- Selected Participants -->
            {#if selectedParticipantsForGroup.length > 0}
                <div class="selected-participants">
                    <strong>Selected:</strong>
                    {#each selectedParticipantsForGroup as p (p.id)}
                        <span class="participant-tag">
                            {p.name} <button type="button" on:click={() => toggleParticipantForNewGroup(p)}>×</button>
                        </span>
                    {/each}
                </div>
            {/if}

            <button type="submit" class="btn btn-primary" disabled={isLoadingChats || !newGroupName.trim()}>
                {isLoadingChats ? 'Creating...' : 'Create Group'}
            </button>
        </form>
      </div>
    </div>
  {/if}

  {#if showManageParticipantsModal && chatToManage}
    <div class="modal-overlay" on:click={closeManageParticipantsModal}>
      <div class="modal-content new-chat-modal" style="max-width: 500px;" on:click|stopPropagation>
        <header class="modal-header-simple">
            <h3>Manage Participants: {chatToManage.name}</h3>
            <button class="close-button-simple" on:click={closeManageParticipantsModal}>×</button>
        </header>
        <div class="participants-list-section">
            <h4>Current Members ({chatToManage.participants.length})</h4>
            <ul class="current-participants-list">
                {#each chatToManage.participants as participant (participant.id)}
                    <li class="participant-item">
                        <div class="chat-item-avatar small"> <!-- Smaller avatar -->
                            {#if participant.profile_picture_url} <img src={participant.profile_picture_url} alt={participant.name} />
                            {:else} <div class="avatar-initials-chat small">{participant.name.charAt(0).toUpperCase()}</div> {/if}
                        </div>
                        <span>{participant.name} (@{participant.username})</span>
                        {#if participant.id !== $currentUserStore?.id && participant.id !== chatToManage.creator_id} <!-- Can't remove self or creator here -->
                            <button class="remove-participant-btn" on:click={() => removeParticipantFromManagedGroup(participant.id)} disabled={isLoadingChats}>Remove</button>
                        {/if}
                    </li>
                {/each}
            </ul>
        </div>
        <div class="add-participants-section">
            <h4>Add New Members</h4>
            <div class="form-group">
                <input type="text" placeholder="Search people to add" bind:value={addUserSearchQuery} on:input={handleAddUserSearch} />
            </div>
            {#if addUserLoading} <p>Searching...</p>
            {:else if addUserResults.length > 0}
                <ul class="user-select-list">
                {#each addUserResults as userRes (userRes.id)}
                    <li on:click={() => addParticipantToManagedGroup(userRes)} role="button">
                        <span>{userRes.name} (@{userRes.username})</span>
                        <span>➕ Add</span>
                    </li>
                {/each}
                </ul>
            {/if}
            {#if manageParticipantsError} <p class="error-text">{manageParticipantsError}</p> {/if}
        </div>
        <!-- TODO: Option to Leave Group -->
      </div>
    </div>
    {/if}

  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .messages-page-layout {
      display: flex;
      height: 100vh;
      overflow: hidden; 
    }
  
    .chat-list-panel {
      width: 300px; 
      border-right: 1px solid var(--border-color);
      display: flex;
      flex-direction: column;
      background-color: var(--sidebar-bg); 
  
      header {
        display: flex; justify-content: space-between; align-items: center;
        padding: 12px 16px; border-bottom: 1px solid var(--border-color);
        h2 { font-size: 20px; font-weight: 800; margin: 0; }
      }
      .chat-list {
        flex-grow: 1; overflow-y: auto;
        scrollbar-width: none;
      }
    }
    .new-chat-btn { background: none; border: none; font-size: 1.5rem; cursor: pointer; color: var(--primary-color); }
  
    .message-display-area {
      flex-grow: 1;
      display: flex;
      flex-direction: column;
      background-color: var(--background);
    }
  
    .message-header {
      display: flex; align-items: center; justify-content: space-between;
      padding: 12px 16px; border-bottom: 1px solid var(--border-color);
      background-color: var(--background);
      .chat-info { display: flex; align-items: center; gap: 10px;
          .header-avatar, .avatar-initials-header { 
            width: 36px; height: 36px; border-radius: 50%;
            flex-shrink: 0;
            overflow: hidden;
            background-color: var(--secondary-text-color);
            font-size: 1rem; font-weight: bold; color: var(--background);
            text-align: center; line-height: 36px;
          }
      }
      h3 { font-size: 1.1rem; font-weight: bold; margin: 0; }
      .options-btn { background:none; border:none; font-size: 1.5rem; cursor:pointer; color: var(--secondary-text-color); }
      .header-actions {
        display: flex;
        gap: 8px;
        margin-left: auto;
         .icon-btn {
            background: none; border: none; cursor: pointer;
            padding: 8px; border-radius: 50%; color: var(--secondary-text-color);
         }
    }
    }
  
    .message-list {
      flex-grow: 1;
      overflow-y: auto;
      padding: 16px;
      display: flex;
      flex-direction: column;
      scrollbar-width: none;
      &::-webkit-scrollbar { 
        display: none;
      }
    }
  
    .message-bubble-wrapper {
        display: flex;
        margin-bottom: 8px;
        max-width: 75%; 
        align-self: flex-start; 
  
        &.own-message {
            align-self: flex-end;
            flex-direction: row-reverse; 
            .message-bubble {
                background-color: var(--primary-color);
                color: var(--primary-button-text);
                border-radius: 18px 18px 4px 18px; 
            }
        }
    }
    .sender-avatar-bubble {
        width: 32px; height: 32px; border-radius: 50%; margin-right: 8px; flex-shrink: 0;
        overflow: hidden; background-color: var(--secondary-text-color);
        img { width: 100%; height: 100%; object-fit: cover; }
        .avatar-initials-bubble { /* ... */ }
         .own-message & { margin-right: 0; margin-left: 8px; }
    }
  
    .message-bubble {
      padding: 8px 12px;
      border-radius: 18px 18px 18px 4px;
      background-color: var(--section-bg);
      word-wrap: break-word;
      p { margin: 0 0 4px; line-height: 1.4; }
      .sender-name { font-size: 0.8rem; font-weight: bold; color: var(--secondary-text-color); margin-bottom: 2px;}
      .message-timestamp {
        font-size: 0.75rem;
        color: var(--secondary-text-color);
        text-align: right;
        display: block;
        margin-top: 4px;
         .own-message & { color: rgba(255,255,255,0.7); }
      }
    }
  
     .message-media-grid {
        display: grid; gap: 4px; margin-top: 8px;
        border-radius: 12px; overflow: hidden;
        &.count-1 { grid-template-columns: minmax(0, 1fr); }
        /* Add more for count-2, count-3, count-4 if desired */
        .msg-media-item {
            aspect-ratio: 16 / 9; /* Or square */
            background-color: var(--border-color);
            img, video { width: 100%; height: 100%; object-fit: cover; }
        }
    }
  
    .no-chat-selected {
      flex-grow: 1; display: flex; flex-direction: column;
      align-items: center; justify-content: center;
      text-align: center; color: var(--secondary-text-color);
      padding: 2rem;
      h2 { font-size: 1.5rem; color: var(--text-color); margin-bottom: 0.5rem; }
    }
  
    .empty-list { text-align: center; padding: 20px; color: var(--secondary-text-color); }
    .error-text { color: var(--error-color); }

    .modal-overlay {
    position: fixed; top: 0; left: 0; width: 100%; height: 100%;
    background-color: rgba(var(--text-color-rgb, 0,0,0), 0.4);
    display: flex; justify-content: center; align-items: center; z-index: 1000;
  }
  .modal-content.new-chat-modal {
    background: var(--background); color: var(--text-color);
    border-radius: 16px; width: 90%; max-width: 450px;
    max-height: 70vh; display: flex; flex-direction: column;
    box-shadow: 0 5px 20px rgba(0,0,0,0.2);
  }
  .modal-header-simple {
    display: flex; justify-content: space-between; align-items: center;
    padding: 12px 16px; border-bottom: 1px solid var(--border-color);
    h3 { margin: 0; font-size: 1.2rem; font-weight: bold; }
    .close-button-simple { /* ... */ }
  }
  .new-chat-search {
    padding: 12px 16px; border-bottom: 1px solid var(--border-color);
    input { width: 100%; padding: 10px; border-radius: 8px; /* ... */ }
  }
  .new-chat-results {
    flex-grow: 1; overflow-y: auto; padding: 8px 0;
    p { text-align: center; color: var(--secondary-text-color); padding: 1rem; }
  }
  .user-result-item {
    display: flex; align-items: center; padding: 10px 16px; cursor: pointer; gap: 10px;
    &:hover { background-color: var(--section-hover-bg); }
    .user-result-info { /* ... */ }
    .user-result-name { font-weight: 600; }
    .user-result-username { color: var(--secondary-text-color); font-size: 0.9rem; }
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

    .deleted-message {
      font-style: italic;
      color: var(--secondary-text-color) !important; 
      background-color: transparent !important; 
      border: 1px dashed var(--border-color);
      padding: 6px 10px !important;
      em {
          
      }
      &.own-message {

      }
      .message-timestamp {
          color: var(--secondary-text-color) !important;
      }
    }

    .message-search-input-bar {
      padding: 8px 16px;
      border-bottom: 1px solid var(--border-color);
      display: flex;
      align-items: center;
      gap: 8px;
      background-color: var(--background);

      input[type="text"] {
          flex-grow: 1;
          padding: 8px 12px;
          border-radius: 18px;
          border: 1px solid var(--border-color);
          background-color: var(--input-bg);
          color: var(--text-color);
          font-size: 14px;
           &:focus { outline: none; border-color: var(--primary-color); }
      }
      .clear-search-btn {
          background: none; border: none; color: var(--secondary-text-color);
          font-size: 1.2rem; cursor: pointer; padding: 4px;
          &:hover { color: var(--text-color); }
      }
  }

  .create-group-form, .participants-list-section, .add-participants-section {
      padding: 16px;
      display: flex;
      flex-direction: column;
      gap: 1rem;
  }
  .user-select-list {
      list-style: none; padding: 0; margin: 0.5rem 0;
      max-height: 150px; overflow-y: auto;
      border: 1px solid var(--border-color); border-radius: 6px;
      li {
          padding: 8px 12px; cursor: pointer; display: flex; justify-content: space-between;
          &:hover { background-color: var(--section-hover-bg); }
          span:last-child { color: var(--primary-color); font-weight: bold; }
      }
  }
  .selected-participants {
      margin-top: 0.5rem; display: flex; flex-wrap: wrap; gap: 6px;
      .participant-tag {
          background-color: var(--primary-color-light, rgba(var(--primary-color-rgb), 0.15));
          color: var(--primary-color);
          padding: 4px 8px; border-radius: 12px; font-size: 0.9rem;
          display: inline-flex; align-items: center; gap: 4px;
          button { background: none; border: none; color: var(--primary-color); cursor: pointer; padding: 0; line-height: 1; }
      }
  }
   .current-participants-list {
      list-style: none; padding: 0; margin: 0; max-height: 200px; overflow-y: auto;
   }
   .participant-item {
       display: flex; align-items: center; gap: 10px; padding: 8px 0;
       border-bottom: 1px solid var(--border-color);
       &:last-child { border-bottom: none; }
       .chat-item-avatar.small { width: 32px; height: 32px; }
       .avatar-initials-chat.small { font-size: 1rem; }
       span { flex-grow: 1; }
       .remove-participant-btn {
           background: none; border: 1px solid var(--error-color); color: var(--error-color);
           padding: 4px 8px; border-radius: 12px; font-size: 0.8rem; cursor: pointer;
            &:hover { background-color: var(--error-bg); }
            &:disabled { opacity: 0.6; }
       }
   }
  
    @keyframes pulse { 0% { background-color: var(--section-hover-bg); } 50% { background-color: var(--border-color); } 100% { background-color: var(--section-hover-bg); } }
      .skeleton-notification-item { display: flex; padding: 12px 16px; border-bottom: 1px solid var(--border-color); gap: 12px; align-items: flex-start; }
      .skeleton-icon { width: 24px; height: 24px; border-radius: 4px; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out; margin-right: 12px; margin-top: 2px;}
      .skeleton-details { flex-grow: 1; display: flex; flex-direction: column; gap: 8px; }
      .skeleton-line { height: 10px; border-radius: 4px; background-color: var(--section-hover-bg); animation: pulse 1.5s infinite ease-in-out;
          &.avatar { width: 30px; height: 30px; border-radius: 50%; margin-bottom: 6px; }
          &.text.short { width: 40%; }
          &.text.long { width: 80%; }
      }
  
  </style>