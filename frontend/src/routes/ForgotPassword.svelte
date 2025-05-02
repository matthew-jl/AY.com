<script lang="ts">
    import { api, ApiError, type GetSecurityQuestionRequestData, type ResetPasswordRequestData } from '../lib/api';
    import { link, navigate } from 'svelte-routing';
    import { tick } from 'svelte';

    type Step = 'email' | 'question' | 'reset' | 'success';

    const securityQuestions = [
      { value: 'pet', label: "What was the name of your first pet?" },
      { value: 'city', label: "What city were you born in?" },
      { value: 'videogame', label: "What is your favorite video game?" },
      { value: 'school', label: "What was the name of your first school?" },
      { value: 'nickname', label: "What was your childhood nickname?" },
    ];

    let currentStep: Step = 'email';
    let email = '';
    let securityQuestionValue: string | null = null;
    let securityAnswer = '';
    let newPassword = '';
    let confirmNewPassword = '';

    let loading = false;
    let error: string | null = null;
    let passwordError: string | null = null;

    $: securityQuestionLabel = securityQuestionValue
        ? securityQuestions.find(q => q.value === securityQuestionValue)?.label ?? 'Unknown security question. Please contact support.'
        : 'Loading question...';

    async function handleEmailSubmit() {
        if (!email || !/^\S+@\S+\.\S+$/.test(email)) {
            error = "Please enter a valid email address."; return;
        }
        loading = true; error = null; securityQuestionValue = null;

        const requestData: GetSecurityQuestionRequestData = { email };

        try {
            const response = await api.getSecurityQuestion(requestData);
            securityQuestionValue = response.security_question;
            currentStep = 'question';
        } catch (err) {
            console.error("Error fetching security question:", err);
            if (err instanceof ApiError) { error = err.message; }
            else if (err instanceof Error) { error = `An error occurred: ${err.message}`; }
            else { error = 'An unexpected error occurred.'; }
        } finally {
            loading = false;
        }
    }

    function validateResetPasswords() { /* ... */ return true; }

    async function handleResetSubmit() {
        error = null; passwordError = null;
        if (!validateResetPasswords()) return;
        if (!securityAnswer || !newPassword) {
            error = "Please provide the security answer and a new password."; return;
        }
        loading = true;

        const requestData: ResetPasswordRequestData = {
            email, security_answer: securityAnswer, new_password: newPassword
        };

        try {
            await api.resetPassword(requestData);
            currentStep = 'success';
        } catch (err) {
            // ... error handling ...
            console.error("Error resetting password:", err);
             if (err instanceof ApiError) { error = err.message; }
             else if (err instanceof Error) { error = `An error occurred: ${err.message}`; }
             else { error = 'An unexpected error occurred.'; }
        } finally {
            loading = false;
        }
    }

    function goBackToEmail() {
        currentStep = 'email'; error = null; passwordError = null;
        securityQuestionValue = null; // Reset value
        securityAnswer = ''; newPassword = ''; confirmNewPassword = '';
    }

    $: if (currentStep === 'question') {
        tick().then(() => {
            const answerInput = document.getElementById('securityAnswer');
            if (answerInput) answerInput.focus();
        });
    }
</script>

<div class="auth-container">
  <img src="/logo_light.png" alt="AY Logo" class="logo" />

  {#if currentStep === 'email'}
    <!-- Step 1: Enter Email -->
    <h2>Find your AY account</h2>
    <p class="info-text">Enter the email address associated with your account.</p>
    <form on:submit|preventDefault={handleEmailSubmit}>
        <!-- ... email input ... -->
         <div class="form-group">
            <label for="email">Email</label>
            <input type="email" id="email" bind:value={email} required />
        </div>
        {#if error} <p class="error-text api-error">{error}</p> {/if}
        <button type="submit" class="btn btn-primary" disabled={loading}>
            {loading ? 'Searching...' : 'Next'}
        </button>
    </form>

  {:else if currentStep === 'question'}
    <!-- Step 2: Answer Security Question -->
     <h2>Answer your security question</h2>
     <!-- Display the mapped LABEL using the reactive variable -->
     <p class="info-text question-text">{securityQuestionLabel}</p>
     <form on:submit|preventDefault={() => currentStep = 'reset'}>
        <!-- ... answer input ... -->
         <div class="form-group">
            <label for="securityAnswer">Your Answer</label>
            <input type="text" id="securityAnswer" bind:value={securityAnswer} required />
        </div>
        <div class="button-row">
            <button type="button" class="btn btn-secondary" on:click={goBackToEmail} disabled={loading}>Back</button>
            <button type="submit" class="btn btn-primary" disabled={loading || !securityAnswer}>Next</button>
        </div>
     </form>

  {:else if currentStep === 'reset'}
    <!-- Step 3: Set New Password -->
    <h2>Choose a new password</h2>
    <p class="info-text">Create a strong password that you don't use elsewhere.</p>
     <form on:submit|preventDefault={handleResetSubmit}>
         <!-- ... password inputs ... -->
          <div class="form-group">
            <label for="newPassword">New Password</label>
            <input type="password" id="newPassword" bind:value={newPassword} on:input={validateResetPasswords} required />
        </div>
         <div class="form-group">
            <label for="confirmNewPassword">Confirm New Password</label>
            <input type="password" id="confirmNewPassword" bind:value={confirmNewPassword} on:input={validateResetPasswords} required />
            {#if passwordError} <p class="error-text">{passwordError}</p> {/if}
        </div>
         {#if error} <p class="error-text api-error">{error}</p> {/if}

         <div class="button-row">
             <button type="submit" class="btn btn-primary" disabled={loading}>
                 {loading ? 'Resetting...' : 'Reset Password'}
             </button>
         </div>
     </form>

  {:else if currentStep === 'success'}
     <!-- Step 4: Success Message -->
     <!-- ... success content ... -->
      <h2>Password Reset</h2>
     <p class="success-text">Your password has been successfully reset.</p>
     <a href="/login" use:link class="btn btn-primary" style="text-align:center;">Log In</a>

  {/if}

  <!-- Links remain the same -->
   {#if currentStep !== 'success'}
    <p class="link-text">
        Remembered your password? <a href="/login" use:link>Log in</a>
    </p>
  {/if}
   <p class="link-text">
        <a href="/" use:link>Back to Landing Page</a>
    </p>

</div>

<style lang="scss">
  @use '../styles/auth-forms.scss';
.error-text {
    color: var(--error-color, #d93025);
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
}

.success-text {
    color: var(--success-color, #1e8e3e);
    background-color: var(--success-bg, #e6f4ea);
    padding: 1rem;
    border-radius: 8px;
    text-align: center;
    margin: 1.5rem 0;
    font-weight: bold;
}

.button-row {
    display: flex;
    gap: 1rem;
    margin-top: 1.5rem;

    .btn {
        margin-top: 0;
        flex: 1;
    }
}

.question-text {
    font-style: italic;
    font-size: 1.1rem;
    color: var(--text-color);
    margin-bottom: 1.5rem;
    border-left: 3px solid var(--primary-color);
    padding-left: 1rem;
    min-height: 1.5em;
}
</style>