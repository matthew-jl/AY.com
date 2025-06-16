<script lang="ts">
    import type { UserProfileBasic } from '../lib/api';
    import { link, navigate } from 'svelte-routing';
    import { api } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
  import { BadgeCheckIcon } from 'lucide-svelte';
  
    export let user: UserProfileBasic;
    export let showFollowButton = true;
  
    let isLoadingFollow = false;
    // TODO: Make follow button state dynamic based on actual follow status
  
    async function handleFollowToggle() {
      if (!$currentUserStore || $currentUserStore.id === user.id) return;
      isLoadingFollow = true;
      try {
        await api.followUser(user.username);
      } catch (error) {
        console.error("Error following user:", error);
        alert("Failed to follow user.");
      } finally {
        isLoadingFollow = false;
      }
    }
  
    function navigateToProfile(event: MouseEvent | KeyboardEvent) {
      // Allow follow button to handle its own click
      if (event.target instanceof HTMLElement && event.target.closest('button.follow-button')) {
          return;
      }
      navigate(`/profile/${user.username}`);
    }
  </script>
  
  <div class="user-card" on:click={navigateToProfile} on:keydown={(e) => e.key === 'Enter' && navigateToProfile(e)} role="link" tabindex="0" aria-label="View profile for {user.name}">
    <div class="user-card-avatar">
      {#if user.profile_picture}
        <img src={user.profile_picture} alt="{user.name}'s profile picture" />
      {:else}
        <div class="avatar-initials-placeholder-card">{user.name?.charAt(0)?.toUpperCase() ?? '?'}</div>
      {/if}
    </div>
    <div class="user-card-info">
      <div class="name-group">
        <span class="user-card-name">{user.name}</span>
        {#if user.is_verified}
          <span class="verified-badge" title="Verified Account">
            <BadgeCheckIcon size={16} />
          </span>
        {/if}
      </div>
      <span class="user-card-username">@{user.username}</span>
      {#if user.bio}
        <p class="user-card-bio">{user.bio}</p>
      {/if}
    </div>
    {#if showFollowButton && $currentUserStore && $currentUserStore.id !== user.id}
      <div class="user-card-action">
        <button class="btn btn-secondary follow-button" on:click|stopPropagation={handleFollowToggle} disabled={isLoadingFollow}>
          <!-- TODO: Change text based on actual follow status -->
          Follow
        </button>
      </div>
    {/if}
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
    @use '../styles/auth-forms.scss';
  
    .verified-badge {
      color: var(--primary-color);
      display: inline-flex;
      align-items: center;
      line-height: 1;
    }

    .user-card {
      display: flex;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      cursor: pointer;
      transition: background-color 0.15s ease-in-out;
  
      &:hover {
        background-color: var(--section-hover-bg);
      }
       &:focus-visible {
           outline: 2px solid var(--primary-color);
           outline-offset: -2px;
           background-color: var(--section-hover-bg);
       }
    }
  
    .user-card-avatar {
      width: 48px;
      height: 48px;
      border-radius: 50%;
      overflow: hidden;
      margin-right: 12px;
      flex-shrink: 0;
      background-color: var(--secondary-text-color);
  
      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }
    .avatar-initials-placeholder-card {
        width: 100%; height: 100%; display: flex;
        align-items: center; justify-content: center;
        font-size: 1.5rem; font-weight: bold; color: var(--background);
    }
  
  
    .user-card-info {
      flex-grow: 1;
      display: flex;
      flex-direction: column;
      justify-content: center;
      overflow: hidden;
    }
  
    .name-group {
        display: flex;
        align-items: center;
        gap: 4px;
    }
    .user-card-name {
      font-weight: bold;
      font-size: 15px;
      color: var(--text-color);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  
    .user-card-username {
      font-size: 15px;
      color: var(--secondary-text-color);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  
    .user-card-bio {
      font-size: 15px;
      color: var(--text-color);
      margin-top: 4px;
      line-height: 1.3;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  
    .user-card-action {
      margin-left: 12px;
      align-self: flex-start;
      padding-top: 4px;
      .follow-button {
          padding: 6px 16px; font-size: 14px; font-weight: bold;
          border-radius: 9999px; cursor: pointer;
          background-color: var(--follow-button-bg);
          color: var(--follow-button-text);
          border: 1px solid var(--follow-button-border);
          &:hover:not(:disabled) { background-color: var(--follow-button-hover-bg); }
           &:disabled { opacity: 0.7; cursor: not-allowed; }
      }
    }
  
  </style>