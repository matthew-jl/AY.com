<script lang="ts">
  import { api, ApiError } from '../lib/api';
  import { navigate } from 'svelte-routing';
  import type { RegisterRequestData, VerifyEmailRequestData } from '../lib/api';
  import { link } from 'svelte-routing';

  let currentStep: 1 | 2 = 1;

  let name = '';
  let username = '';
  let email = '';
  let password = '';
  let confirmPassword = '';
  let gender = '';
  let dateOfBirth = ''; // Expect YYYY-MM-DD
  let securityQuestion = 'pet';
  let securityAnswer = '';

  let verificationCode = '';

  let loading = false;
  let errorStep1: string | null = null;
  let errorStep2: string | null = null;
  let passwordError: string | null = null;
  let successMessage: string | null = null; // after verification

  const securityQuestions = [
    { value: 'pet', label: "What was the name of your first pet?" },
    { value: 'city', label: "What city were you born in?" },
    { value: 'videogame', label: "What is your favorite video game?" },
    { value: 'school', label: "What was the name of your first school?" },
    { value: 'nickname', label: "What was your childhood nickname?" },
  ];

  function validatePasswords() {
    if (password && confirmPassword && password !== confirmPassword) {
      passwordError = "Passwords do not match.";
      return false;
    } else {
      passwordError = null;
      return true;
    }
  }

  function isEmailValid(emailToCheck: string): boolean {
       return /^\S+@\S+\.\S+$/.test(emailToCheck);
  }

  function validateStep1(): boolean {
    errorStep1 = null; // Clear previous error
    if (!validatePasswords()) return false;

    if (!name || !username || !email || !password || !dateOfBirth || !securityQuestion || !securityAnswer) {
      errorStep1 = "Please fill in all required fields.";
      return false;
    }
    if (!isEmailValid(email)) {
      errorStep1 = "Please enter a valid email address.";
      return false;
    }
    return true;
  }

  // Handle submission of Step 1 form
  async function handleStep1Submit() {
    if (!validateStep1()) return;

    loading = true;
    errorStep1 = null; // Clear error before API call

    const userData: RegisterRequestData = {
      name, username, email, password, gender,
      date_of_birth: dateOfBirth,
      security_question: securityQuestion,
      security_answer: securityAnswer,
    };

    try {
      await api.register(userData);
      currentStep = 2;
      errorStep1 = null;
    } catch (err) {
      console.error("Registration Step 1 Error:", err);
      if (err instanceof ApiError) {
        errorStep1 = `Registration failed: ${err.message}`;
      } else if (err instanceof Error) {
        errorStep1 = `An error occurred: ${err.message}`;
      } else {
        errorStep1 = 'An unexpected error occurred during registration.';
      }
    } finally {
      loading = false;
    }
  }

  // Handle submission of Step 2 form (Verification)
  async function handleStep2Submit() {
      if (!verificationCode || verificationCode.length !== 6) {
          errorStep2 = "Please enter the 6-digit verification code.";
          return;
      }
      loading = true;
      errorStep2 = null;
      successMessage = null;

      const verificationData: VerifyEmailRequestData = {
          email: email,
          code: verificationCode
      };

      try {
          await api.verifyEmail(verificationData);
          successMessage = "Account verified successfully! Redirecting to login...";
          setTimeout(() => {
              navigate('/login', { replace: true });
          }, 2000);

      } catch (err) {
           console.error("Verification Step 2 Error:", err);
          if (err instanceof ApiError) {
              errorStep2 = `Verification failed: ${err.message}`;
          } else if (err instanceof Error) {
              errorStep2 = `An error occurred: ${err.message}`;
          } else {
              errorStep2 = 'An unexpected error occurred during verification.';
          }
      } finally {
          loading = false;
      }
  }

  function goToStep1() {
      currentStep = 1;
      errorStep1 = null;
      errorStep2 = null;
      successMessage = null;
  }

  function goToStep2FromLink() {
      if (email && isEmailValid(email)) {
          currentStep = 2;
          errorStep1 = null;
          errorStep2 = null;
          successMessage = null;
      } else {
          errorStep1 = "Please enter a valid email address first to verify.";
          const emailInput = document.getElementById('email');
          if(emailInput) emailInput.focus();
      }
  }
</script>

<div class="auth-container">
  <img src="/logo_light.png" alt="AY Logo" class="logo" />
  {#if currentStep === 1}
    <!-- === Registration Step 1: User Details === -->
    <h2>Create your account</h2>
    <form on:submit|preventDefault={handleStep1Submit}>
      <!-- Name -->
      <div class="form-group">
        <label for="name">Name</label>
        <input type="text" id="name" bind:value={name} required />
      </div>
      <!-- Username -->
      <div class="form-group">
        <label for="username">Username</label>
        <input type="text" id="username" bind:value={username} required />
      </div>
      <!-- Email -->
      <div class="form-group">
        <label for="email">Email</label>
        <input type="email" id="email" bind:value={email} required />
      </div>
      <!-- Password -->
      <div class="form-group">
        <label for="password">Password</label>
        <input type="password" id="password" bind:value={password} on:input={validatePasswords} required />
      </div>
      <!-- Confirm Password -->
      <div class="form-group">
        <label for="confirmPassword">Confirm Password</label>
        <input type="password" id="confirmPassword" bind:value={confirmPassword} on:input={validatePasswords} required />
        {#if passwordError}
          <p class="error-text">{passwordError}</p>
        {/if}
      </div>
      <!-- Date of Birth -->
      <div class="form-group">
        <label for="dob">Date of Birth</label>
        <input type="date" id="dob" bind:value={dateOfBirth} required placeholder="YYYY-MM-DD"/>
      </div>
      <!-- Gender -->
      <div class="form-group">
        <label for="gender">Gender</label>
        <select id="gender" bind:value={gender}>
          <option value="">Select...</option>
          <option value="male">Male</option>
          <option value="female">Female</option>
          <!-- Keep options simple based on backend validation -->
        </select>
      </div>
      <!-- Security Question -->
      <div class="form-group">
          <label for="securityQuestion">Security Question</label>
          <select id="securityQuestion" bind:value={securityQuestion} required>
              {#each securityQuestions as sq}
                  <option value={sq.value}>{sq.label}</option>
              {/each}
          </select>
      </div>
      <!-- Security Answer -->
      <div class="form-group">
          <label for="securityAnswer">Security Answer</label>
          <input type="text" id="securityAnswer" bind:value={securityAnswer} required />
      </div>

      <!-- Step 1 Error Display -->
      {#if errorStep1}
        <p class="error-text api-error">{errorStep1}</p>
      {/if}

      <!-- Step 1 Submit Button -->
      <button type="submit" class="btn btn-primary" disabled={loading}>
        {loading ? 'Processing...' : 'Next'} <!-- Changed label -->
      </button>
    </form>

  {:else if currentStep === 2}
    <!-- === Registration Step 2: Email Verification === -->
    <h2>Verify your email</h2>
    <p class="info-text">We sent a verification code to <strong>{email}</strong>. Please enter the code below.</p>

     <form on:submit|preventDefault={handleStep2Submit}>
        <div class="form-group">
            <label for="verificationCode">Verification Code</label>
            <input
                type="text"
                id="verificationCode"
                bind:value={verificationCode}
                required
                maxlength="6"
                placeholder="6-digit code"
            />
        </div>

        <!-- Step 2 Error Display -->
        {#if errorStep2}
            <p class="error-text api-error">{errorStep2}</p>
        {/if}

         <!-- Step 2 Success Message -->
        {#if successMessage}
            <p class="success-text">{successMessage}</p>
        {/if}

        <!-- Step 2 Action Buttons -->
        <div class="button-row">
             <button type="button" class="btn btn-secondary" on:click={goToStep1} disabled={loading}>
                Previous
             </button>
             <button type="submit" class="btn btn-primary" disabled={loading || !!successMessage}> <!-- Disable if loading or success -->
                {loading ? 'Verifying...' : 'Verify Account'}
             </button>
        </div>
         <!-- TODO: Add "Resend Code" functionality later -->
         <!-- <button type="button" class="btn-link">Resend Code</button> -->
     </form>

  {/if}

  <p class="link-text">
    Already have an account? <a href="/login" use:link>Log in</a>
  </p>
  {#if currentStep === 1 && email && isEmailValid(email)}
    <p class="link-text">
        Already registered? <a href="#verify" on:click|preventDefault={goToStep2FromLink}>Verify here</a>
    </p>
  {/if}
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
    .info-text {
      text-align: center;
      margin-bottom: 1.5rem;
      color: var(--secondary-text-color);
      line-height: 1.4;
       strong {
           color: var(--text-color);
       }
  }
   .success-text {
       color: var(--success-color, #1e8e3e);
       background-color: var(--success-bg, #e6f4ea);
       padding: 0.8rem;
       border-radius: 6px;
       text-align: center;
       margin-top: 1rem;
       font-weight: bold;
   }

  .button-row {
      display: flex;
      gap: 1rem;
      margin-top: 1.5rem;
      // override
       .btn {
           margin-top: 0;
           flex: 1;
       }
  }
</style>