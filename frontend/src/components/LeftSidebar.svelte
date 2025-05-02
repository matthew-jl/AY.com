<script lang="ts">
  import { onMount } from 'svelte';
  import { link, navigate } from 'svelte-routing';
  import { useLocation } from 'svelte-routing';
  import { clearTokens } from '../lib/api';
  import { setAuthState } from '../stores/authStore';

  
  let theme: 'light' | 'dark' = 'light';

  $: logoPath = theme === 'light' ? '/logo_light.png' : '/logo_dark.png';
  
  onMount(() => {
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark';
    if (savedTheme) {
      theme = savedTheme;
      document.documentElement.setAttribute('data-theme', theme);
    }
  });
  
  function toggleTheme() {
    theme = theme === 'light' ? 'dark' : 'light';
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
  }
  
  const location = useLocation();
  const menuItems = [
    { label: 'Home', path: '/home', icon: '🏠' },
    { label: 'Explore', path: '/explore', icon: '🔍' },
    { label: 'Notifications', path: '/notifications', icon: '🔔' },
    { label: 'Messages', path: '/messages', icon: '✉️' },
    { label: 'Bookmarks', path: '/bookmarks', icon: '🔖' },
    { label: 'Communities', path: '/communities', icon: '👥' },
    { label: 'Premium', path: '/premium', icon: '⭐' },
    { label: 'Profile', path: '/profile', icon: '👤' },
    { label: 'Settings', path: '/settings', icon: '⚙' },
  ];

  function handleLogout() {
    console.log("Logging out...");
    clearTokens();
    setAuthState(false);
    navigate('/login', { replace: true });
  }
</script>

<aside class="left-sidebar">
  <div class="logo">
      <img src={logoPath} alt="Logo" />
  </div>
  <nav>
    {#each menuItems as item}
      <a class="menu-item"
        class:active={$location.pathname === item.path}
        href={item.path}
        use:link
        role="menuitem">
        <span class="icon">{item.icon}</span>
        <span>{item.label}</span>
      </a>
    {/each}
  </nav>
  <button class="post-button">Post</button>
  <button class="theme-toggle" on:click={toggleTheme}>
    {theme === 'light' ? '🌙' : '☀️'}
  </button>
  <button class="logout-button" on:click={handleLogout}>
        Logout
  </button>
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
      color: var(--sidebar-active-text);
      background-color: var(--sidebar-hover-bg);
    }
  }

  .icon {
    margin-right: 15px;
    font-size: 22px;
    width: 24px;
    text-align: center;
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
</style>