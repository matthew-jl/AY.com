<script lang="ts">
    import type { CommunityListItem } from '../lib/api';
    import { link, navigate } from 'svelte-routing';
    import { api, CommunityStatusNumbers } from '../lib/api';
    import { user as currentUserStore } from '../stores/userStore';
  
    export let community: CommunityListItem;
    export let onJoinRequested: (() => void) | undefined = undefined; // Callback after join request
  
    let isLoadingJoin = false;
  
    $: isJoined = community.is_joined_by_requester ?? false; // Needs this prop from ListCommunities
    $: hasPendingRequest = community.has_pending_request_by_requester ?? false;
  
    async function handleRequestToJoin() {
      if (!$currentUserStore || isJoined) return;
      isLoadingJoin = true;
      try {
        await api.requestToJoinCommunity(community.id);
        // Optionally update UI immediately to "Pending" or rely on parent to refetch
        alert(`Join request sent for ${community.name}!`);
        if (onJoinRequested) onJoinRequested();
      } catch (err) {
        console.error(`Error requesting to join community ${community.id}:`, err);
        alert(`Failed to send join request: ${typeof err === 'object' && err !== null && 'message' in err ? (err as any).message : 'Unknown error'}`);
      } finally {
        isLoadingJoin = false;
      }
    }
  </script>
  
  <div
    class="community-card"
    on:click={() => navigate(`/community/${community.id}`)}
    on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); navigate(`/community/${community.id}`); } }}
    role="link"
    tabindex="0"
  >
    <div class="card-header">
      {#if community.icon_url}
        <img src={community.icon_url} alt="{community.name} icon" class="community-icon-small" />
      {:else}
        <div class="community-icon-placeholder">{community.name?.charAt(0)?.toUpperCase() ?? 'C'}</div>
      {/if}
      <h4 class="community-name-card">{community.name}</h4>
    </div>
    {#if community.description_snippet}
      <p class="community-description-card">{community.description_snippet}</p>
    {/if}

    {#if community.categories && community.categories.length > 0}
      <div class="community-card-categories">
          {#each community.categories.slice(0, 3) as category (category)} <!-- Show max 3 -->
              <span class="category-tag-card">#{category}</span>
          {/each}
      </div>
    {/if}

    <div class="card-footer">
      <span class="member-count">{community.member_count} member{community.member_count !== 1 ? 's' : ''}</span>
      {#if $currentUserStore && community.status === CommunityStatusNumbers.ACTIVE}
        {#if isJoined}
          <span class="joined-badge">Joined</span>
        {:else if hasPendingRequest}
          <button class="btn btn-secondary join-btn" disabled>Request Pending</button>
        {:else}
          <button class="btn btn-secondary join-btn" on:click|stopPropagation={handleRequestToJoin} disabled={isLoadingJoin}>
            {isLoadingJoin ? 'Sending...' : 'Request to Join'}
          </button>
        {/if}
      {/if}
    </div>
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .community-card {
      background-color: var(--section-bg);
      border: 1px solid var(--border-color);
      border-radius: 12px;
      padding: 16px;
      margin-bottom: 12px;
      cursor: pointer;
      transition: box-shadow 0.2s ease-in-out, transform 0.2s ease-in-out;
  
      &:hover {
        box-shadow: 0 4px 12px rgba(0,0,0,0.08);
        transform: translateY(-2px);
         [data-theme="dark"] & {
             box-shadow: 0 4px 12px rgba(255,255,255,0.05);
         }
      }
    }

    .community-card-categories {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
      margin: 8px 0;
      .category-tag-card {
          background-color: var(--section-hover-bg);
          color: var(--secondary-text-color);
          padding: 3px 8px;
          border-radius: 12px;
          font-size: 0.75rem;
          font-weight: 500;
      }
  }
  
    .card-header {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 8px;
    }
  
    .community-icon-small, .community-icon-placeholder {
      width: 40px;
      height: 40px;
      border-radius: 8px;
      object-fit: cover;
      background-color: var(--secondary-text-color);
      flex-shrink: 0;
    }
    .community-icon-placeholder {
        display: flex; align-items: center; justify-content: center;
        font-size: 1.2rem; font-weight: bold; color: var(--background);
    }
  
    .community-name-card {
      font-size: 1.1rem;
      font-weight: 700;
      color: var(--text-color);
      margin: 0;
      white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    }
  
    .community-description-card {
      font-size: 0.9rem;
      color: var(--secondary-text-color);
      line-height: 1.4;
      margin: 0 0 12px 0;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
      text-overflow: ellipsis;
      min-height: 2.8em;
    }
  
    .card-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-size: 0.85rem;
      color: var(--secondary-text-color);
  
      .join-btn {
        padding: 4px 12px;
        font-size: 0.85rem;
        font-weight: 600;
        background-color: transparent;
        color: var(--primary-color);
        border: 1px solid var(--primary-color);
        border-radius: 9999px;
         &:hover:not(:disabled) { background-color: rgba(var(--primary-color-rgb), 0.1); }
      }
      .joined-badge {
          font-weight: bold;
          color: var(--success-color);
      }
    }
  </style>