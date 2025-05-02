<script lang="ts">
  import { link } from 'svelte-routing';
</script>

<div class="no-access-container">
  <h2>Access Denied</h2>
  <p>You need to be logged in to view this page.</p>
  <div class="button-group">
    <a href="/login" use:link class="btn btn-primary">Log In</a>
    <a href="/" use:link class="btn btn-secondary">Go to Landing Page</a>
  </div>
</div>

<style lang="scss">
  .no-access-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 80vh;
    text-align: center;
    padding: 2rem;
    background-color: var(--background);
    color: var(--text-color);
  }

  h2 {
    color: var(--error-color);
    margin-bottom: 1rem;
  }

  p {
    margin-bottom: 2rem;
    font-size: 1.1rem;
    color: var(--secondary-text-color);
  }

  .button-group {
    display: flex;
    gap: 1rem;
  }

   .btn {
    display: block;
    padding: 0.8rem 1.5rem;
    border-radius: 9999px;
    text-decoration: none;
    font-weight: bold;
    font-size: 1rem;
    cursor: pointer;
    border: 1px solid transparent;
    transition: background-color 0.2s ease;
  }

  .btn-primary {
    background-color: var(--primary-color);
    color: var(--primary-button-text);
    &:hover {
      background-color: var(--primary-color-hover);
    }
  }

  .btn-secondary {
    background-color: transparent;
    color: var(--primary-color);
    border-color: var(--border-color);
     &:hover {
        background-color: var(--sidebar-hover-bg);
     }
  }
</style>