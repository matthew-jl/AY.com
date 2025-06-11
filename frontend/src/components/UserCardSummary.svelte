<script lang="ts">
  import { navigate } from "svelte-routing";

    export let user: {
      id: number;
      name: string;
      username: string;
      profile_picture_url?: string | null;
    };

    function navigateToProfile(event: MouseEvent | KeyboardEvent) {
      // if (event.target instanceof HTMLElement && event.target.closest('button.follow-button')) {
      //     return;
      // }
      navigate(`/profile/${user.username}`);
    }
  </script>
  
  <div class="user-card-simple" on:click={navigateToProfile} on:keydown={(e) => e.key === 'Enter' && navigateToProfile(e)} role="link" tabindex="0" aria-label="View profile for {user.name}">
    {#if user.profile_picture_url}
      <img class="avatar" src={user.profile_picture_url} alt={user.name} />
    {:else}
      <div class="avatar avatar-initial">{user.name?.charAt(0)?.toUpperCase() ?? "U"}</div>
    {/if}
    <div class="user-info">
      <span class="name">{user.name}</span>
      <span class="username">@{user.username}</span>
    </div>
  </div>
  
  <style lang="scss">
  .user-card-simple {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 0;
    cursor: pointer;

    &:hover {
        background-color: var(--section-hover-bg);
      }
    &:focus-visible {
        outline: 2px solid var(--primary-color);
        outline-offset: -2px;
        background-color: var(--section-hover-bg);
    }
  
    .avatar, .avatar-initial {
      width: 38px;
      height: 38px;
      border-radius: 50%;
      object-fit: cover;
      background: var(--secondary-text-color);
      color: var(--background);
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: bold;
      font-size: 1.1rem;
    }
    .avatar-initial {
      font-size: 1.2rem;
    }
    .user-info {
      display: flex;
      flex-direction: column;
      .name {
        font-weight: 600;
        font-size: 15px;
        color: var(--text-color);
      }
      .username {
        font-size: 13px;
        color: var(--secondary-text-color);
      }
    }
  }
  </style>