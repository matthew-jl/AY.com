<script lang="ts">
  import { api, ApiError, saveTokens } from '../lib/api';
  import { setAuthState } from '../stores/authStore';
  import { navigate } from 'svelte-routing';
  import type { LoginRequestData } from '../lib/api';
  import { link } from 'svelte-routing';

  let email = '';
  let password = '';
  let loading = false;
  let error: string | null = null;

  async function handleLogin() {
    error = null;
    if (!email || !password) {
        error = "Please enter both email and password.";
        return;
    }
    loading = true;

    const credentials: LoginRequestData = { email, password };

    try {
      const response = await api.login(credentials);
      saveTokens(response.access_token, response.refresh_token);
      setAuthState(true);
      navigate('/home', { replace: true });
    } catch (err) {
      console.error("Login Error:", err);
      if (err instanceof ApiError) {
        error = `Login failed: ${err.message}`;
      } else if (err instanceof Error) {
        error = `An error occurred: ${err.message}`;
      } else {
        error = 'An unexpected error occurred during login.';
      }
    } finally {
      loading = false;
    }
  }
</script>

<div class="auth-container">
  <img src="/logo_light.png" alt="AY Logo" class="logo" />
  <h2>Sign in to AY</h2>

  <form on:submit|preventDefault={handleLogin}>
    <div class="form-group">
      <label for="email">Email</label>
      <input type="email" id="email" bind:value={email} required />
    </div>
    <div class="form-group">
      <label for="password">Password</label>
      <input type="password" id="password" bind:value={password} required />
    </div>

    {#if error}
      <p class="error-text api-error">{error}</p>
    {/if}

    <button type="submit" class="btn btn-primary" disabled={loading}>
      {loading ? 'Logging in...' : 'Log in'}
    </button>
  </form>

   <p class="link-text">
    Don't have an account? <a href="/register" use:link>Sign up</a>
  </p>
   <!-- Add link to forgotten password later -->
   <!-- <p class="link-text"><a href="/forgot-password" use:link>Forgot password?</a></p> -->

</div>

<style lang="scss">
  @use '../styles/auth-forms.scss';

  .error-text {
       color: var(--error-color);
       font-size: 0.85rem;
       margin-top: 4px;
   }
    .api-error {
        margin-top: 1rem;
        text-align: center;
        font-weight: bold;
    }
</style>