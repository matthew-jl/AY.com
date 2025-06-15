<script lang="ts">
  import { onMount } from 'svelte';
  import { 
    Shield, 
    Palette, 
    User, 
    UserX, 
    Bell, 
    Heart, 
    Repeat, 
    UserPlus, 
    AtSign, 
    Users, 
    Moon, 
    Sun, 
    Check, 
    Info, 
    AlertTriangle 
  } from 'lucide-svelte';
  import { api } from '../lib/api';
  import { user } from '../stores/userStore';

  type SettingsTab = 'security' | 'display' | 'account' | 'blocked' | 'notifications';
  let activeTab: SettingsTab = 'security';

  // Form states
  let isPrivateAccount = false;
  let fontSize = 'medium';
  let colorTheme = 'light';
  let blockedUsers: any[] = [];
  let isLoadingBlockedUsers = false;
  let notificationPreferences = {
    like: true,
    repost: true,
    follow: true,
    mention: true,
    community: true
  };
  let loadingStates = {
    privateAccount: false,
    fontSize: false,
    colorTheme: false,
    notification: false,
    blockList: false
  };

  // Settings loading state
  let isLoadingSettings = true;
  let error: string | null = null;
  let confirmDeactivate = false;
  let deactivateReason = '';
  let deactivatePassword = '';

  // Account deactivation validation
  let reasonError = '';
  let passwordError = '';
  let isDeactivating = false;

  onMount(async () => {
    try {
      // TODO: Fetch user settings from API
      // const settings = await api.getUserSettings();
      
      // Simulate API delay
      await new Promise(resolve => setTimeout(resolve, 800));
      
      // Placeholder data
      isPrivateAccount = false;
      fontSize = 'medium';
      colorTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
      
      // Load notification preferences
      notificationPreferences = {
        like: true,
        repost: true,
        follow: true,
        mention: true,
        community: true
      };
      
      // Load blocked users
    //   await loadBlockedUsers();
      
      isLoadingSettings = false;
    } catch (err) {
      console.error('Failed to load settings:', err);
      error = 'Failed to load your settings. Please try again.';
      isLoadingSettings = false;
    }
  });

  function setActiveTab(tab: SettingsTab) {
    activeTab = tab;
  }

  // Security tab functions
  async function togglePrivateAccount() {
    if (loadingStates.privateAccount) return;
    loadingStates.privateAccount = true;
    
    try {
      // TODO: Update privacy setting via API
      // await api.updatePrivacySetting(!isPrivateAccount);
      
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 500));
      isPrivateAccount = !isPrivateAccount;
      
      // Use alert instead of toast
      alert(`Your account is now ${isPrivateAccount ? 'private' : 'public'}`);
    } catch (err) {
      console.error('Failed to update privacy setting:', err);
      alert('Failed to update privacy setting. Please try again.');
    } finally {
      loadingStates.privateAccount = false;
    }
  }

  // Display tab functions
  async function updateFontSize(size: string) {
    if (loadingStates.fontSize) return;
    loadingStates.fontSize = true;
    
    try {
      // TODO: Update font size via API
      // await api.updateDisplaySettings({ fontSize: size });
      
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 300));
      fontSize = size;
      
      // Apply font size to document root
      document.documentElement.style.fontSize = 
        size === 'small' ? '14px' : 
        size === 'large' ? '18px' : '16px';
      
      alert('Font size updated');
    } catch (err) {
      console.error('Failed to update font size:', err);
      alert('Failed to update font size. Please try again.');
    } finally {
      loadingStates.fontSize = false;
    }
  }

  async function updateColorTheme(theme: string) {
    if (loadingStates.colorTheme) return;
    loadingStates.colorTheme = true;
    
    try {
      // TODO: Update color theme via API
      // await api.updateDisplaySettings({ colorTheme: theme });
      
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 300));
      colorTheme = theme;
      
      // Apply theme to document
      document.documentElement.setAttribute('data-theme', theme);
      
      alert('Theme updated');
    } catch (err) {
      console.error('Failed to update color theme:', err);
      alert('Failed to update color theme. Please try again.');
    } finally {
      loadingStates.colorTheme = false;
    }
  }

  // Account tab functions
  function showDeactivateConfirmation() {
    confirmDeactivate = true;
  }

  function cancelDeactivate() {
    confirmDeactivate = false;
    deactivateReason = '';
    deactivatePassword = '';
    reasonError = '';
    passwordError = '';
  }

  async function deactivateAccount() {
    // Validate inputs
    reasonError = !deactivateReason.trim() ? 'Please tell us why you\'re leaving' : '';
    passwordError = !deactivatePassword.trim() ? 'Password is required' : '';
    
    if (reasonError || passwordError) return;
    
    isDeactivating = true;
    
    try {
      // TODO: Call deactivate account API
      // await api.deactivateAccount(deactivatePassword, deactivateReason);
      
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      alert('Your account has been deactivated. You will be logged out.');
      
      // Simulate logout
      setTimeout(() => {
        window.location.href = '/login';
      }, 2000);
    } catch (err) {
      console.error('Failed to deactivate account:', err);
      alert('Failed to deactivate your account. Please check your password and try again.');
    } finally {
      isDeactivating = false;
    }
  }

  // Blocked accounts functions
  async function unblockUser(userId: number) {
    try {
      // TODO: Unblock user via API
      // await api.unblockUser(userId);
      
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // Update local state
      blockedUsers = blockedUsers.filter(user => user.id !== userId);
      
      alert('User has been unblocked');
    } catch (err) {
      console.error('Failed to unblock user:', err);
      alert('Failed to unblock user. Please try again.');
    }
  }

  // Notification preferences tab functions
  async function toggleNotification(type: keyof typeof notificationPreferences) {
    if (loadingStates.notification) return;
    loadingStates.notification = true;
    
    const newValue = !notificationPreferences[type];
    
    try {
      // TODO: Update notification preference via API
      // await api.updateNotificationPreference(type, newValue);
      
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 300));
      
      // Update local state
      notificationPreferences = {
        ...notificationPreferences,
        [type]: newValue
      };
      
      alert(`${type.charAt(0).toUpperCase() + type.slice(1)} notifications ${newValue ? 'enabled' : 'disabled'}`);
    } catch (err) {
      console.error('Failed to update notification preference:', err);
      alert('Failed to update notification setting. Please try again.');
    } finally {
      loadingStates.notification = false;
    }
  }
</script>

<div class="settings-page">
  <header class="settings-header">
    <h1>Settings</h1>
  </header>

  {#if isLoadingSettings}
    <div class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading your settings...</p>
    </div>
  {:else if error}
    <div class="error-container">
      <AlertTriangle size={24} />
      <p>{error}</p>
      <button class="btn btn-primary" on:click={() => window.location.reload()}>
        Try Again
      </button>
    </div>
  {:else}
    <div class="settings-container">
      <!-- Tabs navigation -->
      <div class="settings-tabs">
        <button 
          class="tab-button" 
          class:active={activeTab === 'security'} 
          on:click={() => setActiveTab('security')}
        >
          <Shield size={18} />
          <span class="tab-text">Security</span>
        </button>
        
        <button 
          class="tab-button" 
          class:active={activeTab === 'display'} 
          on:click={() => setActiveTab('display')}
        >
          <Palette size={18} />
          <span class="tab-text">Display</span>
        </button>
        
        <button 
          class="tab-button" 
          class:active={activeTab === 'account'} 
          on:click={() => setActiveTab('account')}
        >
          <User size={18} />
          <span class="tab-text">Your Account</span>
        </button>
        
        <button 
          class="tab-button" 
          class:active={activeTab === 'blocked'} 
          on:click={() => setActiveTab('blocked')}
        >
          <UserX size={18} />
          <span class="tab-text">Blocked Accounts</span>
        </button>
        
        <button 
          class="tab-button" 
          class:active={activeTab === 'notifications'} 
          on:click={() => setActiveTab('notifications')}
        >
          <Bell size={18} />
          <span class="tab-text">Notification Preferences</span>
        </button>
      </div>

      <!-- Tab content -->
      <div class="tab-content">
        <!-- Security Tab -->
        {#if activeTab === 'security'}
          <div class="tab-panel">
            <h2>
              <Shield size={22} />
              <span>Security Settings</span>
            </h2>
            
            <div class="setting-card">
              <div class="setting-header">
                <h3>Account Privacy</h3>
                <p>Choose who can view your activity and content</p>
              </div>
              
              <div class="setting-option">
                <div class="option-info">
                  <span class="option-label">Private Account</span>
                  <p class="option-description">
                    When enabled, only users you approve can follow you and see your content
                  </p>
                </div>
                <button 
                  class="toggle-switch" 
                  class:active={isPrivateAccount}
                  disabled={loadingStates.privateAccount} 
                  on:click={togglePrivateAccount}
                  aria-pressed={isPrivateAccount}
                >
                  <span class="toggle-slider"></span>
                </button>
              </div>
            </div>
            
            <div class="card-info">
              <Info size={16} />
              <p>When your account is private, users must request to follow you</p>
            </div>
          </div>
        {/if}
        
        <!-- Display Tab -->
        {#if activeTab === 'display'}
          <div class="tab-panel">
            <h2>
              <Palette size={22} />
              <span>Display Settings</span>
            </h2>
            
            <div class="setting-card">
              <div class="setting-header">
                <h3>Font Size</h3>
                <p>Change the font size for a comfortable reading experience</p>
              </div>
              
              <div class="font-size-options">
                <button 
                  class="font-size-option" 
                  class:active={fontSize === 'small'}
                  disabled={loadingStates.fontSize}
                  on:click={() => updateFontSize('small')}
                >
                  <span class="size-text small">Aa</span>
                  <span class="size-label">Small</span>
                  {#if fontSize === 'small'}<Check size={16} />{/if}
                </button>
                
                <button 
                  class="font-size-option" 
                  class:active={fontSize === 'medium'}
                  disabled={loadingStates.fontSize}
                  on:click={() => updateFontSize('medium')}
                >
                  <span class="size-text medium">Aa</span>
                  <span class="size-label">Medium</span>
                  {#if fontSize === 'medium'}<Check size={16} />{/if}
                </button>
                
                <button 
                  class="font-size-option" 
                  class:active={fontSize === 'large'}
                  disabled={loadingStates.fontSize}
                  on:click={() => updateFontSize('large')}
                >
                  <span class="size-text large">Aa</span>
                  <span class="size-label">Large</span>
                  {#if fontSize === 'large'}<Check size={16} />{/if}
                </button>
              </div>
            </div>
            
            <div class="setting-card">
              <div class="setting-header">
                <h3>Color Theme</h3>
                <p>Choose your preferred appearance</p>
              </div>
              
              <div class="theme-options">
                <button 
                  class="theme-option" 
                  class:active={colorTheme === 'light'}
                  disabled={loadingStates.colorTheme}
                  on:click={() => updateColorTheme('light')}
                >
                  <div class="theme-icon light">
                    <Sun size={24} />
                  </div>
                  <span class="theme-label">Light</span>
                  {#if colorTheme === 'light'}<Check size={16} />{/if}
                </button>
                
                <button 
                  class="theme-option" 
                  class:active={colorTheme === 'dark'}
                  disabled={loadingStates.colorTheme}
                  on:click={() => updateColorTheme('dark')}
                >
                  <div class="theme-icon dark">
                    <Moon size={24} />
                  </div>
                  <span class="theme-label">Dark</span>
                  {#if colorTheme === 'dark'}<Check size={16} />{/if}
                </button>
              </div>
            </div>
          </div>
        {/if}
        
        <!-- Account Tab -->
        {#if activeTab === 'account'}
          <div class="tab-panel">
            <h2>
              <User size={22} />
              <span>Your Account</span>
            </h2>
            
            {#if !confirmDeactivate}
              <div class="setting-card danger-zone">
                <div class="setting-header">
                  <h3>Deactivate Account</h3>
                  <p>Temporarily disable your account</p>
                </div>
                
                <div class="setting-info">
                  <AlertTriangle size={18} />
                  <p>
                    Account deactivation will hide your profile, posts, and replies from others. 
                    You can reactivate anytime by logging back in.
                  </p>
                </div>
                
                <button class="btn btn-danger" on:click={showDeactivateConfirmation}>
                  Deactivate Account
                </button>
              </div>
            {:else}
              <div class="setting-card confirmation-card">
                <div class="setting-header">
                  <h3>Are you sure?</h3>
                  <p>Confirm account deactivation</p>
                </div>
                
                <form on:submit|preventDefault={deactivateAccount}>
                  <div class="form-group">
                    <label for="deactivate-reason">Why are you deactivating?</label>
                    <textarea 
                      id="deactivate-reason" 
                      bind:value={deactivateReason}
                      placeholder="Please tell us why you're leaving..." 
                      rows="3"
                      disabled={isDeactivating}
                    ></textarea>
                    {#if reasonError}
                      <span class="input-error">{reasonError}</span>
                    {/if}
                  </div>
                  
                  <div class="form-group">
                    <label for="deactivate-password">Enter your password to confirm</label>
                    <input 
                      type="password" 
                      id="deactivate-password" 
                      bind:value={deactivatePassword} 
                      placeholder="Your password"
                      disabled={isDeactivating}
                    />
                    {#if passwordError}
                      <span class="input-error">{passwordError}</span>
                    {/if}
                  </div>
                  
                  <div class="button-group">
                    <button 
                      type="button" 
                      class="btn btn-outline" 
                      on:click={cancelDeactivate}
                      disabled={isDeactivating}
                    >
                      Cancel
                    </button>
                    <button 
                      type="submit" 
                      class="btn btn-danger" 
                      disabled={isDeactivating}
                    >
                      {isDeactivating ? 'Processing...' : 'Confirm Deactivation'}
                    </button>
                  </div>
                </form>
              </div>
            {/if}
          </div>
        {/if}
        
        <!-- Blocked Accounts Tab -->
        {#if activeTab === 'blocked'}
          <div class="tab-panel">
            <h2>
              <UserX size={22} />
              <span>Blocked Accounts</span>
            </h2>
            
            <div class="setting-card">
              <div class="setting-header">
                <h3>Manage Blocked Users</h3>
                <p>View and unblock users</p>
              </div>
              
              {#if isLoadingBlockedUsers}
                <div class="loading-section">
                  <div class="loading-spinner small"></div>
                  <p>Loading blocked accounts...</p>
                </div>
              {:else if blockedUsers.length === 0}
                <div class="empty-state">
                  <UserX size={40} />
                  <p>You haven't blocked any accounts</p>
                </div>
              {:else}
                <div class="blocked-users-list">
                  {#each blockedUsers as user (user.id)}
                    <div class="blocked-user">
                      <div class="user-info">
                        <img 
                          src={user.profile_picture} 
                          alt={user.name} 
                          class="user-avatar"
                          loading="lazy"
                        />
                        <div class="user-details">
                          <h4>{user.name}</h4>
                          <span class="username">@{user.username}</span>
                        </div>
                      </div>
                      <button 
                        class="btn btn-outline btn-sm" 
                        on:click={() => unblockUser(user.id)}
                      >
                        Unblock
                      </button>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
            
            <div class="card-info">
              <Info size={16} />
              <p>Blocked users cannot see your posts, message you, or find you in search</p>
            </div>
          </div>
        {/if}
        
        <!-- Notifications Tab -->
        {#if activeTab === 'notifications'}
          <div class="tab-panel">
            <h2>
              <Bell size={22} />
              <span>Notification Preferences</span>
            </h2>
            
            <div class="setting-card">
              <div class="setting-header">
                <h3>Manage Notifications</h3>
                <p>Choose which notifications you receive</p>
              </div>
              
              <div class="notification-options">
                <div class="notification-option">
                  <div class="notification-info">
                    <Heart size={18} />
                    <span class="option-label">Likes</span>
                    <p class="option-description">When someone likes your post</p>
                  </div>
                  <button 
                    class="toggle-switch" 
                    class:active={notificationPreferences.like}
                    disabled={loadingStates.notification}
                    on:click={() => toggleNotification('like')}
                    aria-pressed={notificationPreferences.like}
                  >
                    <span class="toggle-slider"></span>
                  </button>
                </div>
                
                <div class="notification-option">
                  <div class="notification-info">
                    <Repeat size={18} />
                    <span class="option-label">Reposts</span>
                    <p class="option-description">When someone reposts your content</p>
                  </div>
                  <button 
                    class="toggle-switch" 
                    class:active={notificationPreferences.repost}
                    disabled={loadingStates.notification}
                    on:click={() => toggleNotification('repost')}
                    aria-pressed={notificationPreferences.repost}
                  >
                    <span class="toggle-slider"></span>
                  </button>
                </div>
                
                <div class="notification-option">
                  <div class="notification-info">
                    <UserPlus size={18} />
                    <span class="option-label">Follows</span>
                    <p class="option-description">When someone follows you</p>
                  </div>
                  <button 
                    class="toggle-switch" 
                    class:active={notificationPreferences.follow}
                    disabled={loadingStates.notification}
                    on:click={() => toggleNotification('follow')}
                    aria-pressed={notificationPreferences.follow}
                  >
                    <span class="toggle-slider"></span>
                  </button>
                </div>
                
                <div class="notification-option">
                  <div class="notification-info">
                    <AtSign size={18} />
                    <span class="option-label">Mentions</span>
                    <p class="option-description">When someone mentions you</p>
                  </div>
                  <button 
                    class="toggle-switch" 
                    class:active={notificationPreferences.mention}
                    disabled={loadingStates.notification}
                    on:click={() => toggleNotification('mention')}
                    aria-pressed={notificationPreferences.mention}
                  >
                    <span class="toggle-slider"></span>
                  </button>
                </div>
                
                <div class="notification-option">
                  <div class="notification-info">
                    <Users size={18} />
                    <span class="option-label">Community</span>
                    <p class="option-description">Updates from communities you're in</p>
                  </div>
                  <button 
                    class="toggle-switch" 
                    class:active={notificationPreferences.community}
                    disabled={loadingStates.notification}
                    on:click={() => toggleNotification('community')}
                    aria-pressed={notificationPreferences.community}
                  >
                    <span class="toggle-slider"></span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style lang="scss">
  @use '../styles/variables' as *;
  
  .settings-page {
    width: 100%;
    max-width: 800px;
    margin: 0 auto;
    padding: 0px 20px 40px;
    
    @media (max-width: 930px) {
      max-width: 100%;
      padding: 0px 16px 40px;
    }
    
    @media (max-width: 480px) {
      padding: 0px 12px 32px;
    }
  }
    .settings-header {
    padding: 16px 0;
    border-bottom: 1px solid var(--border-color);
    margin-bottom: 24px;
    margin-left: -4px;
    
    h1 {
      font-size: 20px;
      font-weight: 800;
      margin: 0;
    }
  }
  
  .loading-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 0;
    
    .loading-spinner {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      border: 3px solid var(--border-color);
      border-top: 3px solid var(--primary-color);
      animation: spin 1s linear infinite;
      margin-bottom: 16px;
      
      &.small {
        width: 20px;
        height: 20px;
        border-width: 2px;
      }
    }
    
    p {
      color: var(--secondary-text-color);
    }
    
    @keyframes spin {
      0% { transform: rotate(0deg); }
      100% { transform: rotate(360deg); }
    }
  }
  
  .error-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    text-align: center;
    
    :global(svg) {
      color: var(--error-color);
      margin-bottom: 16px;
    }
    
    p {
      color: var(--text-color);
      margin-bottom: 20px;
    }
  }
  
  .settings-container {
    display: grid;
    grid-template-columns: 220px 1fr;
    gap: 24px;
    
    @media (max-width: 680px) {
      grid-template-columns: 1fr;
    }
  }
  
  .settings-tabs {
    display: flex;
    flex-direction: column;
    gap: 8px;
    position: sticky;
    top: 20px;
    align-self: start;
    
    @media (max-width: 680px) {
      position: static;
      flex-direction: row;
      overflow-x: auto;
      padding-bottom: 12px;
      gap: 4px;      margin: -8px -16px 12px -16px;
      padding: 0 16px 12px;
      
      &::-webkit-scrollbar {
        height: 2px;
      }
      
      &::-webkit-scrollbar-thumb {
        background-color: var(--border-color);
      }
    }
    
    .tab-button {
      display: flex;
      align-items: center;
      gap: 12px;
      background: none;
      border: none;
      padding: 12px 16px;
      border-radius: 9999px;
      cursor: pointer;
      color: var(--text-color);
      text-align: left;
      transition: background-color 0.2s;
      font-weight: 600;
      font-size: 15px;
      white-space: nowrap;
        @media (max-width: 680px) {
        padding: 8px 12px;
        font-size: 14px;
        flex-shrink: 0;
      }
      
      @media (max-width: 480px) {
        padding: 6px 10px;
      }
      
      &:hover {
        background-color: var(--section-hover-bg);
      }
      
      &.active {
        background-color: var(--section-hover-bg);
        color: var(--primary-color);
        
        :global(svg) {
          color: var(--primary-color);
        }
      }
      
      :global(svg) {
        flex-shrink: 0;
      }
    }
  }
  
  .tab-content {
    .tab-panel {
      h2 {
        display: flex;
        align-items: center;
        gap: 12px;
        font-size: 20px;
        margin: 0 0 20px;
        
        :global(svg) {
          color: var(--text-color);
        }
      }
    }
  }
  
  .setting-card {
    background-color: var(--section-bg);
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 20px;
    border: 1px solid var(--border-color);
    
    &.danger-zone {
      border-color: rgba(255, 0, 0, 0.1);
      
      .setting-info {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        background-color: rgba(255, 0, 0, 0.05);
        padding: 12px;
        border-radius: 8px;
        margin: 16px 0;
        
        :global(svg) {
          color: var(--error-color);
          flex-shrink: 0;
          margin-top: 2px;
        }
        
        p {
          margin: 0;
          font-size: 14px;
        }
      }
    }
    
    &.confirmation-card {
      background-color: rgba(255, 0, 0, 0.02);
      
      form {
        .button-group {
          display: flex;
          gap: 12px;
          justify-content: flex-end;
          margin-top: 24px;
        }
      }
    }
    
    .setting-header {
      margin-bottom: 16px;
      
      h3 {
        font-size: 18px;
        margin: 0 0 4px;
      }
      
      p {
        color: var(--secondary-text-color);
        margin: 0;
        font-size: 14px;
      }
    }
    
    .setting-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 0;
      
      .option-info {
        .option-label {
          font-weight: 600;
          font-size: 16px;
          display: block;
          margin-bottom: 4px;
        }
        
        .option-description {
          color: var(--secondary-text-color);
          font-size: 14px;
          margin: 0;
        }
      }
    }
  }
  
  .toggle-switch {
    width: 50px;
    height: 26px;
    background-color: var(--border-color);
    border-radius: 13px;
    position: relative;
    cursor: pointer;
    border: none;
    transition: background-color 0.2s;
    
    &.active {
      background-color: var(--primary-color);
    }
    
    .toggle-slider {
      position: absolute;
      top: 3px;
      left: 3px;
      width: 20px;
      height: 20px;
      background-color: white;
      border-radius: 50%;
      transition: transform 0.2s;
    }
    
    &.active .toggle-slider {
      transform: translateX(24px);
    }
    
    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }
  
  .font-size-options {
    display: flex;
    gap: 16px;
    margin: 16px 0;
    
    @media (max-width: 480px) {
      gap: 8px;
    }
    
    .font-size-option {
      display: flex;
      flex-direction: column;
      align-items: center;
      padding: 16px;
      background: none;
      border: 1px solid var(--border-color);
      border-radius: 12px;
      cursor: pointer;
      flex: 1;
      transition: all 0.2s;
      
      &.active {
        border-color: var(--primary-color);
        background-color: rgba(var(--primary-color-rgb), 0.05);
        color: var(--primary-color);
      }
      
      &:hover:not(.active):not(:disabled) {
        background-color: var(--section-hover-bg);
      }
      
      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
      
      .size-text {
        font-weight: bold;
        margin-bottom: 8px;
        
        &.small { font-size: 16px; }
        &.medium { font-size: 20px; }
        &.large { font-size: 24px; }
      }
      
      .size-label {
        font-size: 14px;
        margin-bottom: 4px;
      }
      
      :global(svg) {
        color: var(--primary-color);
      }
    }
  }
  
  .theme-options {
    display: flex;
    gap: 16px;
    margin: 16px 0;
    
    .theme-option {
      display: flex;
      flex-direction: column;
      align-items: center;
      padding: 16px;
      background: none;
      border: 1px solid var(--border-color);
      border-radius: 12px;
      cursor: pointer;
      flex: 1;
      transition: all 0.2s;
      
      &.active {
        border-color: var(--primary-color);
        background-color: rgba(var(--primary-color-rgb), 0.05);
        color: var(--primary-color);
      }
      
      &:hover:not(.active):not(:disabled) {
        background-color: var(--section-hover-bg);
      }
      
      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
      
      .theme-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 48px;
        height: 48px;
        border-radius: 24px;
        margin-bottom: 12px;
        
        &.light {
          background-color: #f8f8f8;
          color: #ff9500;
        }
        
        &.dark {
          background-color: #222;
          color: #8e8e93;
        }
      }
      
      .theme-label {
        font-size: 14px;
        margin-bottom: 4px;
      }
      
      :global(svg) {
        color: var(--primary-color);
      }
    }
  }
  
  .notification-options {
    display: flex;
    flex-direction: column;
    gap: 16px;
    
    .notification-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 0;
      
      .notification-info {
        display: flex;
        flex-direction: column;
        
        :global(svg) {
          margin-bottom: 8px;
          color: var(--secondary-text-color);
        }
        
        .option-label {
          font-weight: 600;
          font-size: 16px;
          display: block;
          margin-bottom: 4px;
        }
        
        .option-description {
          color: var(--secondary-text-color);
          font-size: 14px;
          margin: 0;
        }
      }
    }
  }
  
  .blocked-users-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
    max-height: 400px;
    overflow-y: auto;
    
    .blocked-user {
      display: flex;
      justify-content: space-between;
      align-items: center;
      
      .user-info {
        display: flex;
        align-items: center;
        gap: 12px;
        
        .user-avatar {
          width: 40px;
          height: 40px;
          border-radius: 50%;
          object-fit: cover;
        }
        
        .user-details {
          h4 {
            font-size: 16px;
            margin: 0 0 2px;
          }
          
          .username {
            font-size: 14px;
            color: var(--secondary-text-color);
          }
        }
      }
    }
  }
  
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 32px 16px;
    text-align: center;
    
    :global(svg) {
      color: var(--secondary-text-color);
      margin-bottom: 16px;
      opacity: 0.7;
    }
    
    p {
      color: var(--secondary-text-color);
      margin: 0;
    }
  }
  
  .loading-section {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 32px 16px;
    
    p {
      color: var(--secondary-text-color);
      margin: 16px 0 0;
    }
  }
  
  .card-info {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    margin-bottom: 24px;
    
    :global(svg) {
      color: var(--secondary-text-color);
      flex-shrink: 0;
      margin-top: 3px;
    }
    
    p {
      color: var(--secondary-text-color);
      font-size: 14px;
      margin: 0;
    }
  }
  
  .form-group {
    margin-bottom: 16px;
    
    label {
      display: block;
      font-weight: 600;
      margin-bottom: 8px;
    }
    
    input, textarea {
      width: 100%;
      padding: 12px;
      border: 1px solid var(--border-color);
      border-radius: 8px;
      background-color: var(--background);
      color: var(--text-color);
      
      &:focus {
        outline: none;
        border-color: var(--primary-color);
      }
      
      &:disabled {
        opacity: 0.7;
        cursor: not-allowed;
      }
    }
    
    .input-error {
      color: var(--error-color);
      font-size: 14px;
      margin-top: 4px;
      display: block;
    }
  }
  
  .btn {
    display: inline-block;
    padding: 10px 16px;
    border-radius: 9999px;
    font-weight: 600;
    font-size: 15px;
    text-align: center;
    cursor: pointer;
    border: 1px solid transparent;
    transition: background-color 0.2s;
    
    &.btn-primary {
      background-color: var(--primary-color);
      color: white;
      
      &:hover:not(:disabled) {
        background-color: var(--primary-color-hover);
      }
    }
    
    &.btn-outline {
      background: none;
      border-color: var(--border-color);
      color: var(--text-color);
      
      &:hover:not(:disabled) {
        background-color: var(--section-hover-bg);
      }
    }
    
    &.btn-danger {
      background-color: var(--error-color);
      color: white;
      
      &:hover:not(:disabled) {
        background-color: #d63031;
      }
    }
    
    &.btn-sm {
      font-size: 14px;
      padding: 6px 12px;
    }
    
    &:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
  }
</style>