<script lang="ts">
  import { api, ApiError, type HealthResponse } from '../lib/api';
  import { onMount } from 'svelte';

  let healthStatus: string | null = null;
  let error: string | null = null;
  let loading: boolean = true;

  onMount(async () => {
    loading = true;
    error = null;
    healthStatus = null;

    try {
      const response: HealthResponse = await api.getHealth();
      healthStatus = response.status;
    } catch (err) {
      console.error("Error fetching health status:", err);
      if (err instanceof ApiError) {
        error = `API Error (${err.status}): ${err.message}`;
      } else if (err instanceof Error) {
        error = `Error: ${err.message}`;
      } else {
        error = 'An unexpected error occurred.';
      }
    } finally {
      loading = false;
    }
  });
</script>

<div class="home-container">
  <h1>Home Page</h1>

  {#if loading}
    <p class="status-message">Loading health status...</p>
  {:else if healthStatus}
    <p class="status-message success">✅ API Gateway Health Status: {healthStatus}</p>
  {:else if error}
    <p class="status-message error">❌ {error}</p>
    <p>Check the browser console and API Gateway/User Service logs for more details.</p>
  {:else}
    <p class="status-message">Could not retrieve health status.</p>
  {/if}

  <!-- TODO: Add Feed Creation Form and Feed Display Here -->

</div>

<style>
  .home-container {
    padding: 20px 30px;
    width: 100%;
    box-sizing: border-box;
  }

  h1 {
    margin-bottom: 1.5rem;
    border-bottom: 1px solid var(--border-color);
    padding-bottom: 0.5rem;
  }

  .status-message {
    padding: 0.8rem 1rem;
    border-radius: 6px;
    margin-bottom: 1rem;
    border: 1px solid var(--border-color);
    background-color: var(--section-bg);

    &.success {
        border-left: 4px solid var(--success-color);
        background-color: var(--success-bg);
    }
    &.error {
        border-left: 4px solid var(--error-color);
         background-color: var(--error-bg);
    }
  }
</style>