<script lang="ts">
  import { link } from 'svelte-routing';
</script>

<div class="landing-container">
  <div class="landing-content">
    <div class="logo-section">
      <img src="/logo_light.png" alt="AY Logo" class="logo" />
    </div>
    
    <div class="headline-section">
      <h1>Happening now</h1>
      <h2>Join today.</h2>
    </div>

    <div class="button-section">
      <a href="/register" use:link class="btn btn-primary">Create account</a>
      <p>Already have an account?</p>
      <a href="/login" use:link class="btn btn-secondary">Sign in</a>
    </div>
  </div>
</div>

<style lang="scss">
  .landing-container {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    width: 100%;
    padding: 1rem;
    background-color: var(--sidebar-bg);
    box-sizing: border-box;
  }

  .landing-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 100%;
    max-width: 500px;
    text-align: center;
    padding: 2rem;
    // border-radius: 16px;
    // background-color: var(--background);
    // box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  }

  .logo-section {
    margin-bottom: 2rem;
  }

  .logo {
    width: 80px;
    height: auto;
  }

  .headline-section {
    margin-bottom: 2.5rem;
  }

  h1 {
    font-size: clamp(2rem, 5vw, 3rem);
    font-weight: bold;
    margin-bottom: 0.5rem;
    line-height: 1.1;
  }

  h2 {
    font-size: clamp(1.3rem, 3vw, 1.8rem);
    margin-bottom: 0;
    line-height: 1.3;
  }

  .button-section {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    width: 100%;
    max-width: 300px;
  }

  p {
    margin: 0.5rem 0 0.2rem;
    font-weight: bold;
    font-size: clamp(0.9rem, 2vw, 1rem);
  }

  .btn {
    display: block;
    padding: clamp(0.6rem, 2vw, 0.8rem) 1rem;
    border-radius: 9999px;
    text-decoration: none;
    font-weight: bold;
    font-size: clamp(0.9rem, 2vw, 1rem);
    cursor: pointer;
    border: 1px solid transparent;
    transition: background-color 0.2s ease, transform 0.1s ease;
    
    &:active {
      transform: scale(0.98);
    }
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
    border-color: var(--primary-color);
    &:hover {
      background-color: var(--sidebar-hover-bg);
    }
  }

  /* Media queries for responsive layout */
  @media (max-width: 600px) {
    .landing-content {
      padding: 1.5rem;
      max-width: 90%;
    }
    
    .logo {
      width: 60px;
    }
    
    .headline-section {
      margin-bottom: 2rem;
    }
  }
  
  @media (max-width: 400px) {
    .landing-content {
      padding: 1.2rem;
      max-width: 95%;
      box-shadow: none;
      background-color: var(--sidebar-bg);
    }
    
    .logo-section {
      margin-bottom: 1.5rem;
    }
    
    .button-section {
      gap: 0.8rem;
    }
  }
  
  @media (max-height: 500px) {
    .landing-container {
      padding: 0.5rem;
    }
    
    .landing-content {
      padding: 1rem;
    }
    
    .logo-section {
      margin-bottom: 1rem;
    }
    
    .logo {
      width: 50px;
    }
    
    .headline-section {
      margin-bottom: 1rem;
    }
    
    .button-section {
      gap: 0.5rem;
    }
    
    h1 {
      margin-bottom: 0.3rem;
    }
  }
</style>