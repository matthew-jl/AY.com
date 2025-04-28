<script lang="ts">
  import { link } from 'svelte-routing';
  import { useLocation } from 'svelte-routing';

  const location = useLocation();

  // Placeholder links based on image
  const menuItems = [
    { label: 'Home', path: '/home', icon: '🏠' },
    { label: 'Explore', path: '/explore', icon: '🔍' },
    { label: 'Notifications', path: '/notifications', icon: '🔔' },
    { label: 'Messages', path: '/messages', icon: '✉️' },
    { label: 'Grok', path: '/grok', icon: '🤖' },
    { label: 'Bookmarks', path: '/bookmarks', icon: '🔖' },
    { label: 'Communities', path: '/communities', icon: '👥' },
    { label: 'Premium', path: '/premium', icon: '⭐' },
    { label: 'Verified Orgs', path: '/verified-orgs', icon: '✅' },
    { label: 'Profile', path: '/profile', icon: '👤' },
    { label: 'More', path: '/more', icon: '⋯' },
  ];
</script>

<aside class="left-sidebar">
  <div class="logo">AY</div>
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
  }

  .logo {
    font-size: 28px;
    font-weight: bold;
    margin-bottom: 25px;
    text-align: center;
    padding: 0 10px;
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

    &:hover {
      background-color: var(--sidebar-hover-bg);
    }

    &.active {
      font-weight: bold;
      color: var(--sidebar-active-text);
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

  .left-sidebar::-webkit-scrollbar {
    width: 6px;
  }
  .left-sidebar::-webkit-scrollbar-thumb {
    background-color: var(--scrollbar-thumb-color);
    border-radius: 3px;
  }
  .left-sidebar::-webkit-scrollbar-track {
    background: var(--scrollbar-track-color);
  }
</style>