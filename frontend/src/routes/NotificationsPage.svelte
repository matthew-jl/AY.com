<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { api, ApiError, type NotificationData } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
    import { getAccessToken } from '../lib/api';
    import { link, navigate } from 'svelte-routing';
  import { timeAgo } from '../lib/utils/timeAgo';
    // import NotificationItem from '../components/NotificationItem.svelte'; // TODO: Create this later
  
    type NotificationTab = 'all' | 'mentions';
  
    let activeTab: NotificationTab = 'all';
    let notifications: NotificationData[] = [];
    let isLoading = true;
    let error: string | null = null;
  
    let unreadCount = 0;
  
    let ws: WebSocket | null = null;
  
    async function fetchInitialNotifications() {
      isLoading = true;
      error = null;
      try {
        const response = await api.getNotifications(1, 20, activeTab === 'all' ? false : false);
        notifications = response.notifications || [];
        fetchUnreadCount();
      } catch (err) {
        console.error("Error fetching notifications:", err);
        if (err instanceof ApiError) { error = `Failed to load notifications: ${err.message}`; }
        else { error = "An unexpected error occurred."; }
      } finally {
        isLoading = false;
      }
    }
  
    async function fetchUnreadCount() {
        try {
            const response = await api.getUnreadNotificationCount();
            unreadCount = response.count;
            // TODO: Update a global store for unread count if needed for a badge in sidebar
        } catch (err) {
            console.error("Error fetching unread count:", err);
        }
    }
  
    function connectWebSocket() {
      const token = getAccessToken();
      if (!token || !$currentUserStore) {
        console.warn("No token or user for WebSocket connection.");
        return;
      }
  
      // If connecting directly to notification-service WS port (e.g., 8081)
      const wsUrl = `ws://localhost:8081/ws/notifications?token=${token}`;
      console.log("Attempting to connect to WebSocket:", wsUrl);
  
      ws = new WebSocket(wsUrl);
  
      ws.onopen = () => {
        console.log("WebSocket connection established for notifications.");
      };
  
      ws.onmessage = (event) => {
        console.log("WebSocket message received:", event.data);
        try {
          const notification = JSON.parse(event.data as string) as NotificationData;
          notifications = [notification, ...notifications];
          unreadCount++;
          // TODO: Show a browser notification if permission granted
        } catch (e) {
          console.error("Error parsing WebSocket message:", e);
        }
      };
  
      ws.onerror = (err) => {
        console.error("WebSocket error:", err);
      };
  
      ws.onclose = (event) => {
        console.log("WebSocket connection closed:", event.code, event.reason);
        ws = null;
        // Optional: Implement reconnection logic
      };
    }
  
    onMount(() => {
      fetchInitialNotifications();
      if ($currentUserStore) {
          connectWebSocket();
      }
  
      const authUnsubscribe = currentUserStore.subscribe(user => {
          if (user && !ws) {
              connectWebSocket();
          } else if (!user && ws) {
              ws.close();
              ws = null;
          }
      });
  
      return () => {
        if (ws) {
          ws.close();
        }
        authUnsubscribe();
      };
    });
  
    function switchTab(tab: NotificationTab) {
      if (activeTab === tab) return;
      activeTab = tab;
      // TODO: Refetch notifications if backend supports filtering by type
    }
  
    async function markAsRead(notificationId: number) {
        const notif = notifications.find(n => n.id === notificationId);
        if (notif && !notif.is_read) {
            try {
                await api.markNotificationAsRead(notificationId);
                notifications = notifications.map(n => n.id === notificationId ? { ...n, is_read: true } : n);
                if (unreadCount > 0) unreadCount--;
            } catch (err) { console.error("Failed to mark as read:", err); }
        }
    }
  
    async function markAllAsRead() {
        try {
            await api.markAllNotificationsAsRead();
            notifications = notifications.map(n => ({ ...n, is_read: true }));
            unreadCount = 0;
        } catch (err) { console.error("Failed to mark all as read:", err); }
    }
  
  
    // --- Client-side filtering for tabs (until backend supports type filter for GetNotifications) ---
    $: displayedNotifications = activeTab === 'mentions'
      ? notifications.filter(n => n.type === 'mention')
      : notifications;
  
    function getNotificationLink(notification: NotificationData): string {
        switch (notification.type) {
            case 'thread_like':
            case 'mention':
            case 'reply':
                return `/thread/${notification.entity_id}`; // Link to the thread
            case 'new_follower':
                const usernameMatch = notification.message.match(/^@(\w+)/);
                if (usernameMatch) {
                  return `/profile/${usernameMatch[1]}`;
                }
                return `/profile/${notification.entity_id}`;
            default:
                return '#';
        }
    }
  
  </script>
  
  <div class="notifications-page">
    <header class="page-header">
      <div class="header-content">
          <h2>Notifications</h2>
      </div>
      {#if notifications.length > 0 && unreadCount > 0}
          <button class="btn-link mark-all-read" on:click={markAllAsRead} disabled={isLoading}>Mark all as read</button>
      {/if}
    </header>
  
    <nav class="profile-tabs notification-tabs">
      <button class:active={activeTab === 'all'} on:click={() => switchTab('all')}>All</button>
      <button class:active={activeTab === 'mentions'} on:click={() => switchTab('mentions')}>Mentions</button>
      <!-- TODO: Add Verified tab later if needed -->
    </nav>
  
    <section class="notifications-list">
      {#if isLoading && notifications.length === 0}
        <p>Loading notifications...</p> <!-- TODO: Skeleton Loader -->
         {#each { length: 7 } as _}
             <div class="skeleton-notification-item">
                 <div class="skeleton-icon"></div>
                 <div class="skeleton-details">
                     <div class="skeleton-line avatar"></div>
                     <div class="skeleton-line text short"></div>
                     <div class="skeleton-line text long"></div>
                 </div>
             </div>
         {/each}
      {:else if error}
        <p class="error-text api-error">{error}</p>
      {:else if displayedNotifications.length > 0}
        {#each displayedNotifications as notification (notification.id)}
          <!-- Using <a> tag to make the whole item clickable -->
          <a  href={getNotificationLink(notification)}
              use:link
              class="notification-item"
              class:unread={!notification.is_read}
              on:click={() => markAsRead(notification.id)}
              role="link"
              tabindex="0"
              aria-label="View notification: {notification.message}"
          >
            <div class="notification-icon">
              {#if notification.type === 'new_follower'}👤
              {:else if notification.type === 'thread_like'}❤️
              {:else if notification.type === 'mention'}@
              {:else if notification.type === 'reply'}💬
              {:else}ℹ️{/if}
            </div>
            <div class="notification-details">
              <!-- TODO: Display actor's profile picture if available -->
              <!-- <img src={notification.actor_profile_picture || defaultAvatar} alt="" class="actor-avatar-small" /> -->
              <p class="notification-message">{@html notification.message}</p>
              <span class="notification-timestamp">{timeAgo(notification.created_at)}</span>
            </div>
            {#if !notification.is_read}
              <div class="unread-dot"></div>
            {/if}
          </a>
        {:else}
          <!-- No items in this tab after filtering -->
          <p class="empty-notifications">
              {#if activeTab === 'all'}No notifications yet.
              {:else if activeTab === 'mentions'}You have no mentions yet.
              {/if}
          </p>
        {/each}
      {:else if !isLoading}
          <!-- No notifications at all (empty initial fetch) -->
           <p class="empty-notifications">Nothing to see here — yet.</p>
      {/if}
    </section>
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .notifications-page { width: 100%; }
  
    .page-header {
      position: sticky; top: 0;
      background-color: rgba(var(--background-rgb), 0.85);
      backdrop-filter: blur(12px);
      z-index: 10;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      display: flex;
      justify-content: space-between;
      align-items: center;
  
      .header-content h2 { font-size: 20px; font-weight: 800; margin: 0; }
      .mark-all-read {
          background: none; border: none; color: var(--primary-color);
          font-size: 14px; font-weight: 500; cursor: pointer;
          padding: 6px 0;
          &:hover { text-decoration: underline; }
          &:disabled { color: var(--secondary-text-color); cursor: default; text-decoration: none; }
      }
    }
  
    .notification-tabs {
      display: flex; border-bottom: 1px solid var(--border-color);
      background-color: var(--background);
      position: sticky;
      top: 57px;
      z-index: 9;
  
      button { 
        flex: 1;
        padding: 16px;
        background: none; border: none;
        color: var(--secondary-text-color);
        font-weight: bold; font-size: 15px;
        cursor: pointer; position: relative;
        transition: background-color 0.2s ease;
        &:hover { background-color: var(--section-hover-bg); }
        &.active {
          color: var(--text-color);
          &::after {
            content: ''; position: absolute; bottom: 0; left: 0; right: 0;
            height: 4px; background-color: var(--primary-color); border-radius: 2px;
          }
        }
       }
    }
  
    .notifications-list {
    }
  
    .notification-item {
      display: flex;
      align-items: flex-start;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      text-decoration: none;
      color: var(--text-color);
      cursor: pointer;
      position: relative;
      transition: background-color 0.15s ease-in-out;
  
      &:hover {
        background-color: var(--section-hover-bg);
      }
  
      &.unread {
        /* background-color: rgba(var(--primary-color-rgb), 0.05); */ /* Subtle unread bg */
        /* border-left: 3px solid var(--primary-color); */ /* Or a left border */
      }
    }
  
    .notification-icon {
      font-size: 1.5rem;
      margin-right: 12px;
      padding-top: 2px;
      color: var(--secondary-text-color);
       .notification-item.unread & {
           color: var(--primary-color);
       }
    }
  
    .notification-details {
      flex-grow: 1;
      .notification-message {
        margin: 0 0 4px 0;
        font-size: 15px;
        line-height: 1.4;
        :global(a) {
            color: var(--primary-color);
            text-decoration: none;
            &:hover { text-decoration: underline; }
        }
      }
      .notification-timestamp {
        font-size: 13px;
        color: var(--secondary-text-color);
      }
    }
  
    .unread-dot {
        position: absolute;
        top: 16px;
        right: 16px;
        width: 10px;
        height: 10px;
        background-color: var(--primary-color);
        border-radius: 50%;
    }
  
  
    .empty-notifications, .error-text.api-error {
        text-align: center; padding: 40px 20px; color: var(--secondary-text-color);
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