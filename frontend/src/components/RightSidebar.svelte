<script lang="ts">
</script>

<aside class="right-sidebar">
  <div class="sticky-container">
    <div class="search-container">
      <div class="search-bar">
        <svg viewBox="0 0 24 24" aria-hidden="true" class="search-icon"><g><path d="M10.25 3.75c-3.59 0-6.5 2.91-6.5 6.5s2.91 6.5 6.5 6.5c1.795 0 3.42-.726 4.596-1.904 1.178-1.177 1.904-2.801 1.904-4.596 0-3.59-2.91-6.5-6.5-6.5zm-8.5 6.5c0-4.694 3.806-8.5 8.5-8.5s8.5 3.806 8.5 8.5c0 1.986-.682 3.815-1.824 5.262l4.781 4.781-1.414 1.414-4.781-4.781c-1.447 1.142-3.276 1.824-5.262 1.824-4.694 0-8.5-3.806-8.5-8.5z"></path></g></svg>
        <input type="text" placeholder="Search" aria-label="Search query" />
      </div>
    </div>

    <section class="content-box premium-box">
      <h3>Subscribe to Premium</h3>
      <p>Subscribe to unlock new features and if eligible, receive a share of ads revenue.</p>
      <button class="premium-button">Subscribe</button>
    </section>

    <section class="content-box trending">
      <h3>What's Happening</h3>
      <ul>
        <li class="list-item trend-item">
          <a href="#">
            <div class="item-context">Politics · Trending</div>
            <div class="item-title">From the Desk of Anthony...</div>
            <div class="item-context">10.5K posts</div>
          </a>
        </li>
        <li class="list-item trend-item">
           <a href="#">
            <div class="item-context">#NCT Dream · Trending</div>
            <div class="item-title">#NCT드림</div>
            <div class="item-context">Trending with #MARK</div>
           </a>
        </li>
        <li class="list-item trend-item">
           <a href="#">
            <div class="item-context">Pagi in Indonesia</div>
            <div class="item-title">Trending Topic Placeholder</div>
            <div class="item-context">9,874 posts</div>
           </a>
        </li>
        <li class="list-item show-more">
           <a href="/explore">Show more</a>
        </li>
      </ul>
    </section>

    <section class="content-box who-to-follow">
      <h3>Who to Follow</h3>
      <ul>
          <li class="list-item follow-item">
             <a href="#" class="follow-link">
              <div class="avatar-placeholder"></div>
              <div class="user-info">
                <span class="user-name">Brawlhalla</span>
                <span class="user-handle">@Brawlhalla</span>
              </div>
            </a>
            <button class="follow-button">Follow</button>
          </li>
          <li class="list-item follow-item">
             <a href="#" class="follow-link">
              <div class="avatar-placeholder"></div>
              <div class="user-info">
                <span class="user-name">Peach</span>
                <span class="user-handle">@Peach</span>
              </div>
             </a>
            <button class="follow-button">Follow</button>
          </li>
          <li class="list-item show-more">
             <a href="/connect_people">Show more</a>
          </li>
      </ul>
    </section>
  </div>
</aside>

<style lang="scss">
  @use '../styles/variables' as *;

  .right-sidebar {
    width: $right-sidebar-width;
    padding-left: 30px;
    padding-right: 15px; 
    position: relative; 
    height: 100vh; 
  }

  .sticky-container {
    position: sticky; 
    top: 10px; 
    height: calc(100vh - 20px); 
    overflow-y: auto;
    scrollbar-width: none; 
    &::-webkit-scrollbar { 
        display: none;
    }
  }

  .search-container {
      position: sticky; 
      top: 0;
      background: var(--background); 
      padding-top: 5px;
      padding-bottom: 5px;
      z-index: 1; 
      margin-left: -10px;
      margin-right: -10px;
      padding-left: 10px;
      padding-right: 10px;

  }

  .search-bar {
    position: relative;
    margin-bottom: 15px;

    input {
      width: 100%;
      padding: 12px 12px 12px 40px;
      border-radius: 9999px;
      border: 1px solid transparent;
      background: var(--search-bg);
      color: var(--text-color);
      font-size: 15px;
      &:focus {
        outline: none;
        border-color: var(--search-border-focus);
        background: var(--background);
        box-shadow: 0 0 0 1px var(--search-border-focus);
      }
    }
    .search-icon {
        position: absolute;
        top: 50%;
        left: 12px;
        transform: translateY(-50%);
        width: 18px;
        height: 18px;
        fill: var(--secondary-text-color);
        pointer-events: none;
    }

  }

  .content-box {
    background: var(--section-bg);
    border-radius: 16px;
    margin-bottom: 15px;
    overflow: hidden;
  }

  .content-box h3 {
    font-size: 20px;
    font-weight: 800;
    padding: 12px 16px;
    margin: 0;
    border-bottom: 1px solid var(--border-color);
  }

  .premium-box {
     padding: 12px 16px;
     h3 {
         font-size: 17px;
         font-weight: bold;
         padding: 0;
         border-bottom: none;
         margin-bottom: 4px;
     }
     p {
         font-size: 14px;
         color: var(--secondary-text-color);
         line-height: 1.3;
         margin-bottom: 12px;
     }
     .premium-button {
         background: var(--follow-button-bg);
         color: var(--follow-button-text);
         border: 1px solid var(--follow-button-border);
         padding: 6px 16px;
         border-radius: 9999px;
         cursor: pointer;
         font-weight: bold;
         font-size: 14px;
         transition: background-color 0.2s ease-in-out;
         &:hover {
             background: var(--follow-button-hover-bg);
         }
     }
  }


  .list-item {
    list-style: none;
    margin: 0;
    padding: 0;
    border-bottom: 1px solid var(--border-color);
     &:last-child {
       border-bottom: none;
     }

    a {
        display: block;
        padding: 12px 16px;
        text-decoration: none;
        color: inherit;
        transition: background-color 0.2s ease-in-out;
         &:hover {
           background-color: var(--section-hover-bg);
           cursor: pointer;
         }
    }
  }


  .trend-item {
    .item-context {
      font-size: 13px;
      color: var(--secondary-text-color);
      margin-bottom: 2px;
    }
    .item-title {
      font-weight: bold;
      font-size: 15px;
      color: var(--text-color);
      line-height: 1.3;
      margin-bottom: 4px;
    }
  }

  .follow-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;

    &:hover {
       background-color: var(--section-hover-bg);
    }

    .follow-link {
      display: flex;
      align-items: center;
      text-decoration: none;
      color: inherit;
      flex-grow: 1;
      padding: 0;
       &:hover {
         background-color: transparent;
       }
    }

     .avatar-placeholder {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        background-color: var(--secondary-text-color);
        margin-right: 12px;
        flex-shrink: 0;
     }
     .user-info {
        display: flex;
        flex-direction: column;
        line-height: 1.2;
        overflow: hidden;
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

    .follow-button {
        background: var(--follow-button-bg);
        color: var(--follow-button-text);
        border: 1px solid var(--follow-button-border);
        padding: 6px 16px;
        border-radius: 9999px;
        cursor: pointer;
        font-weight: bold;
        font-size: 14px;
        flex-shrink: 0;
        margin-left: 10px;
        transition: background-color 0.2s ease-in-out;
         &:hover {
           background: var(--follow-button-hover-bg);
         }
    }
  }

  .show-more {
    padding: 12px 16px;
    a {
        color: var(--primary-color);
        font-size: 15px;
        text-decoration: none;
         &:hover {
           text-decoration: underline;
           background-color: transparent;
         }
    }
  }

  ul {
      margin: 0;
      padding: 0;
  }

</style>