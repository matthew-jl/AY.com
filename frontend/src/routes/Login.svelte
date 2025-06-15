<script lang="ts">
  import { api, ApiError, clearTokens, saveTokens } from '../lib/api';
  import { setAuthState } from '../stores/authStore';
  import { navigate } from 'svelte-routing';
  import type { LoginRequestData } from '../lib/api';
  import { link } from 'svelte-routing';
  import { onDestroy, onMount } from 'svelte';
  import { clearUser, setUser } from '../stores/userStore';

  let email = '';
  let password = '';
  let loading = false;
  let error: string | null = null;

  let recaptchaToken: string | null = null;
  let recaptchaWidgetId: number | null = null;
  let isRecaptchaScriptLoaded = false;

  const recaptchaSiteKey = import.meta.env.VITE_RECAPTCHA_SITE_KEY;

   // --- reCAPTCHA Callbacks ---
  function onRecaptchaSuccess(token: string) {
    console.log("Login reCAPTCHA solved");
    recaptchaToken = token;
    error = null; // Clear potential reCAPTCHA errors
  }
  function onRecaptchaExpired() {
    console.log("Login reCAPTCHA expired");
    recaptchaToken = null;
  }
  function onRecaptchaError() {
     console.error("Login reCAPTCHA error callback");
     error = "reCAPTCHA challenge failed. Please try again.";
     recaptchaToken = null;
  }

  // --- Lifecycle & Script Loading ---
  onMount(() => {
    if (!recaptchaSiteKey) {
        console.error("VITE_RECAPTCHA_SITE_KEY is not set!");
        error = "reCAPTCHA configuration error.";
        return;
    }

    const renderWidget = () => {
      const container = document.getElementById('recaptcha-container-login');
      if (container && window.grecaptcha && window.grecaptcha.render) {
         try {
             console.log("Rendering Login reCAPTCHA widget...");
             recaptchaWidgetId = window.grecaptcha.render(container, {
                 sitekey: recaptchaSiteKey,
                 callback: onRecaptchaSuccess,
                 'expired-callback': onRecaptchaExpired,
                 'error-callback': onRecaptchaError,
             });
         } catch (renderError) {
             console.error("Error rendering Login reCAPTCHA:", renderError);
             onRecaptchaError();
         }
      } else {
          console.warn("Login renderWidget called but container or grecaptcha not ready yet.");
      }
    }

    // Define the global callback function only once across the app
    if (!window.onloadRecaptchaCallback) {
        console.log("Defining onloadRecaptchaCallback (Login)");
        window.onloadRecaptchaCallback = () => {
            console.log("reCAPTCHA script loaded via callback (Login).");
            isRecaptchaScriptLoaded = true;
            renderWidget(); // Render login widget now
        };
    } else {
        // If callback exists, script might already be loaded
        if (window.grecaptcha) {
            console.log("reCAPTCHA script potentially already loaded (Login).");
            isRecaptchaScriptLoaded = true;
            setTimeout(renderWidget, 0); // Render after DOM might update
        }
    }

    // Load the script only if it doesn't exist
    if (!document.getElementById('recaptcha-script')) {
        console.log("Loading reCAPTCHA script (Login)...");
        const script = document.createElement('script');
        script.id = 'recaptcha-script';
        script.src = 'https://www.google.com/recaptcha/api.js?onload=onloadRecaptchaCallback&render=explicit';
        script.async = true; script.defer = true; script.onerror = onRecaptchaError;
        document.body.appendChild(script);
    } else {
        // If script tag exists, check if object is ready and render
         if (window.grecaptcha) {
             isRecaptchaScriptLoaded = true;
             setTimeout(renderWidget, 0);
         } else {
              console.log("Login: reCAPTCHA script tag exists but grecaptcha object not ready, waiting.");
         }
    }
  });

  onDestroy(() => {
    // Optional cleanup
    console.log("Login component destroyed");
  });

   // --- Form Validation & Submission ---
   function validateLogin(): boolean {
       error = null;
       if (!email || !password) {
           error = "Please enter both email and password."; return false;
       }
       if (!recaptchaToken) { // Check reCAPTCHA token
           error = "Please complete the reCAPTCHA verification."; return false;
       }
       return true;
   }

  async function handleLogin() {
    if (!validateLogin()) {
      return;
    }
    error = null;
    loading = true;

    const credentials: LoginRequestData = { email, password, recaptchaToken: recaptchaToken as string };

    try {
      // 1. Login and get tokens
      const loginResponse = await api.login(credentials);
      saveTokens(loginResponse.access_token, loginResponse.refresh_token);

      // 2. Fetch user profile using the new token
      try {
          const userProfileApiResponse = await api.getOwnUserProfile();
          setUser(userProfileApiResponse.user); // Update user store
      } catch (profileError) {
          console.error("Login successful, but failed to fetch profile:", profileError);
          setUser(null);
      }

      // 3. Update global auth state
      setAuthState(true);

      // 4. Navigate to home
      navigate('/home', { replace: true });

    } catch (err) {
      console.error("Login Error:", err);
      // Reset reCAPTCHA if used
      recaptchaToken = null;
      clearTokens();
      clearUser();
      setAuthState(false);
      if (err instanceof ApiError) { error = `Login failed: ${err.message}`; }
      else if (err instanceof Error) { error = `An error occurred: ${err.message}`; }
      else { error = 'An unexpected error occurred during login.'; }
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

    <!-- reCAPTCHA Container -->
    <div class="form-group recaptcha-container">
        {#if recaptchaSiteKey}
             <div id="recaptcha-container-login">
             </div>
        {:else}
            <p class="error-text">reCAPTCHA Site Key not configured.</p>
        {/if}
    </div>

    {#if error}
      <p class="error-text api-error">{error}</p>
    {/if}

    <button type="submit" class="btn btn-primary" disabled={loading}>
      {loading ? 'Logging in...' : 'Log in'}
    </button>
  </form>

  <p class="link-text">
    <a href="/" use:link>Go back to Landing page</a>
  </p>
   <p class="link-text">
    Don't have an account? <a href="/register" use:link>Sign up</a>
  </p>
   <p class="link-text"><a href="/forgot-password" use:link>Forgot password?</a></p>

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

  /* Media queries for responsive layout */
  @media (max-width: 480px) {
    :global(.auth-container) {
      margin: 2rem auto;
      padding: 1.5rem;
      max-width: 90%;
      border-radius: 12px;
    }
    
    :global(.recaptcha-container) {
      display: flex;
      justify-content: center;
      transform: scale(0.9);
      transform-origin: center;
    }
  }

  @media (max-width: 350px) {
    :global(.auth-container) {
      margin: 1rem auto;
      padding: 1rem;
      box-shadow: none;
      background-color: var(--sidebar-bg);
      border-width: 0;
    }

    :global(h2) {
      font-size: 1.5rem;
      margin-bottom: 1rem;
    }

    :global(.form-group) {
      gap: 0.5rem;
    }

    :global(.form-group input) {
      padding: 0.6rem 0.7rem;
      font-size: 0.95rem;
    }
    
    :global(.recaptcha-container) {
      transform: scale(0.85);
      margin-left: -20px;
      margin-right: -20px;
    }
  }
</style>