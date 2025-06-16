<script lang="ts">
  import { onMount } from 'svelte';
  import { link, navigate } from 'svelte-routing';
  import { useLocation } from 'svelte-routing';
  import { clearTokens } from '../lib/api';
  import { setAuthState } from '../stores/authStore';
  import { clearUser, user } from '../stores/userStore';
  import { openCreateThreadModal } from '../stores/modalStore';

  import { Home, Search, Bell, Mail, Bookmark, Users, Star, User, Settings, Moon, Sun, BadgeCheckIcon } from 'lucide-svelte';

  let theme: 'light' | 'dark' = 'light';
  let showLogout = false;
  let isCompact = false;

  $: logoPath = theme === 'light' ? '/logo_light.png' : '/logo_dark.png';

  // Responsive: update isCompact on resize
  function updateCompact() {
    isCompact = window.innerWidth < 900;
  }
  onMount(() => {
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark';
    if (savedTheme) {
      theme = savedTheme;
      document.documentElement.setAttribute('data-theme', theme);
    }
    updateCompact();
    window.addEventListener('resize', updateCompact);
    return () => window.removeEventListener('resize', updateCompact);
  });

  function toggleTheme() {
    theme = theme === 'light' ? 'dark' : 'light';
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
  }

  function toggleLogout() {
    showLogout = !showLogout;
  }

  const location = useLocation();
  const menuItems = [
    { label: 'Home', path: '/home', icon: Home },
    { label: 'Explore', path: '/explore', icon: Search },
    { label: 'Notifications', path: '/notifications', icon: Bell },
    { label: 'Messages', path: '/messages', icon: Mail },
    { label: 'Bookmarks', path: '/bookmarks', icon: Bookmark },
    { label: 'Communities', path: '/communities', icon: Users },
    { label: 'Premium', path: '/premium', icon: Star },
    { label: 'Profile', path: $user ? `/profile/${$user.username}` : '/profile', icon: User },
    { label: 'Settings', path: '/settings', icon: Settings },
  ];

  function handleLogout() {
    console.log("Logging out...");
    clearUser();
    clearTokens();
    setAuthState(false);
    navigate('/login', { replace: true });
  }
</script>

<aside class="left-sidebar {isCompact ? 'compact' : ''}">
  <div class="logo">
      <img src={logoPath} alt="Logo" />
  </div>
  <nav>
    {#each menuItems as item}
      <a class="menu-item"
        class:active={$location.pathname === item.path}
        href={item.path}
        use:link
        role="menuitem"
        aria-label={item.label}>
        <span class="icon"><svelte:component this={item.icon} size="1.6em" /></span>
        {#if !isCompact}
          <span>{item.label}</span>
        {/if}
      </a>
    {/each}
  </nav>
  <button class="post-button" on:click={openCreateThreadModal}>
    {#if isCompact}
      +
    {:else}
      Post
    {/if}
  </button>
  <button class="theme-toggle" on:click={toggleTheme} aria-label="Toggle theme">
    {#if theme === 'light'}
      <Moon size="1.4em" />
    {:else}
      <Sun size="1.4em" />
    {/if}
  </button>
  {#if $user}
    <div class="user-info-container" role="button" tabindex="0" aria-label="Account options" on:click={toggleLogout} on:keydown={(e) => e.key === 'Enter' && toggleLogout()}>
        <div class="user-avatar-placeholder">
            {#if $user.profile_picture}
              <img src="{$user.profile_picture}" alt="{$user.name}" style="width:100%;height:100%;border-radius:50%;" />
            {:else}
              {$user.name.charAt(0).toUpperCase()}
            {/if}
        </div>
        {#if !isCompact}
        <div class="user-details">
          <div class="user-name-line">
            <span class="user-name">{$user.name}</span>
              {#if $user.is_verified}
                  <span class="verified-badge" title="Verified Account">
                      <BadgeCheckIcon size={16} />
                  </span>
              {/if}
          </div>
            <span class="user-handle">@{$user.username}</span>
        </div>
        {/if}
    </div>
    {#if showLogout}
      <button class="logout-button" on:click={handleLogout}>
          Logout
      </button>
    {/if}
   {/if}
</aside>

<style lang="scss">
  @use '../styles/variables' as *;

  .left-sidebar {
    width: $left-sidebar-width;
    background: var(--sidebar-bg);
    color: var(--sidebar-text);
    position: fixed;
    top: 0;
    left: 0;
    height: 100vh;
    overflow-y: auto;
    padding: 15px 10px;
    border-right: 1px solid var(--border-color);
    display: flex;
    flex-direction: column;
    scrollbar-width: none;
    transition: width 0.2s;
    z-index: 10;
    &.compact {
      width: 70px;
      .menu-item span:not(.icon),
      .post-button,
      .user-details,
      .logout-button {
        display: none !important;
      }
      .icon {
        margin-right: 0;
        justify-content: center;
      }
      .post-button {
        width: 40px;
        height: 40px;
        padding: 0;
        font-size: 1.5em;
        border-radius: 50%;
        margin-top: 20px;
      }
      .theme-toggle {
        width: 40px;
        height: 40px;
        padding: 0;
        border-radius: 50%;
      }
      .user-avatar-placeholder {
        margin-right: 0;
      }
    }
  }

  .logo {
    margin-bottom: 5px;
    padding: 0 10px;

    img {
       width: 40px;
       height: auto;
       display: block;
    }
  }

  nav {
     flex-grow: 1;
  }

  .menu-item {
    display: flex;
    align-items: center;
    padding: 12px 15px;
    margin-bottom: 5px;
    cursor: pointer;
    border-radius: 25px;
    transition: background-color 0.2s ease-in-out;
    font-weight: 600;

    &:hover {
      background-color: var(--sidebar-hover-bg);
    }

    &.active {
      font-weight: 700;
      color: var(--primary-color);
      background-color: var(--sidebar-hover-bg);
    }
  }

  .icon {
    margin-right: 15px;
    font-size: 22px;
    width: 24px;
    text-align: center;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .post-button {
    background: var(--primary-color);
    color: var(--primary-button-text);
    border: none;
    padding: 14px 20px;
    border-radius: 25px;
    margin-top: 20px;
    width: 90%;
    align-self: center;
    cursor: pointer;
    font-size: 16px;
    font-weight: bold;
    transition: background-color 0.2s ease-in-out;

    &:hover {
       background: var(--primary-color-hover);
    }
  }

  .theme-toggle {
    background: #444;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 20px;
    margin: 10px 0;
    width: 90%;
    align-self: center;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .logout-button {
      background: var(--logout-button-bg);
      color: var(--logout-button-text);
      border: 1px solid var(--logout-button-border);
      padding: 10px 20px;
      border-radius: 9999px;
      width: 90%;
      align-self: center;
      cursor: pointer;
      font-size: 15px;
      font-weight: 600;
      transition: background-color 0.2s ease-in-out, border-color 0.2s ease-in-out, color 0.2s ease-in-out;

       &:hover {
           background-color: var(--logout-button-hover-bg);
           border-color: var(--logout-button-hover-border);
           color: var(--text-color);
       }
  }

  .user-info-container {
      display: flex;
      align-items: center;
      padding: 8px 12px;
      margin: 5px 0;
      border-radius: 9999px;
      cursor: pointer;
      transition: background-color 0.2s ease-in-out;
      width: 100%;
      box-sizing: border-box;
      background: transparent;
      border: none;
      color: var(--sidebar-text);
      text-align: left;

      &:hover {
          background-color: var(--sidebar-hover-bg);
      }
       &:focus-visible {
            outline: 2px solid var(--primary-color);
            outline-offset: 2px;
       }
  }

  .user-avatar-placeholder {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      background-color: var(--secondary-text-color);
      color: var(--background);
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: bold;
      font-size: 1.1rem;
      margin-right: 10px;
      flex-shrink: 0;
  }

  .user-details {
      flex-grow: 1;
      display: flex;
      flex-direction: column;
      overflow: hidden;
      line-height: 1.3;
  }

  .user-name-line {
      display: flex;
      align-items: center;
      gap: 6px;
  }
  
  .verified-badge {
      color: var(--primary-color);
      display: inline-flex;
      align-items: center;
  }

  .user-name {
      font-weight: bold;
      font-size: 15px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
  }

  .user-handle {
      font-size: 14px;
      color: var(--secondary-text-color);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
  }

  .user-logout-icon {
      margin-left: auto;
      padding-left: 10px;
      flex-shrink: 0;
       svg {
           width: 20px;
           height: 20px;
           fill: currentColor;
       }
  }

  @media (max-width: 900px) {
    .left-sidebar {
      width: 70px;
    }
    .left-sidebar:not(.compact) {
      .menu-item span:not(.icon),
      .post-button,
      .user-details,
      .logout-button {
        display: none !important;
      }
      .icon {
        margin-right: 0;
        justify-content: center;
      }
      .post-button {
        width: 40px;
        height: 40px;
        padding: 0;
        font-size: 1.5em;
        border-radius: 50%;
        margin-top: 20px;
      }
      .theme-toggle {
        width: 40px;
        height: 40px;
        padding: 0;
        border-radius: 50%;
      }
      .user-avatar-placeholder {
        margin-right: 0;
      }
    }
  }
</style>