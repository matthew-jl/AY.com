<script lang="ts">
  import { onMount } from 'svelte';
  import { Badge, CheckCircle, Shield, Star, Upload, AlertCircle } from 'lucide-svelte';
  import { api } from '../lib/api';
  import { user as currentUserStore } from '../stores/userStore';
  import { link } from 'svelte-routing';
  
  // State for premium status
  let isPremium = false;
  let isPendingVerification = false;
  let isLoading = true;
  let isSubmitting = false;
  let error: string | null = null;
  let successMessage: string | null = null;
  
  // Form data
  let identityCardNumber = '';
  let verificationReason = '';
  let faceImage: File | null = null;
  let faceImagePreview: string | null = null;
  
  // Form validation
  let formErrors = {
    identityCardNumber: '',
    verificationReason: '',
    faceImage: ''
  };

  onMount(async () => {
    try {
      // TODO: API call to check user premium status
      // const response = await api.getUserPremiumStatus();
      // isPremium = response.isPremium;
      // isPendingVerification = response.isPendingVerification;
      
      // For demo purposes
      isPremium = false;
      isPendingVerification = false;
    } catch (err) {
      console.error("Error fetching premium status:", err);
      error = "Failed to load premium status. Please try again.";
    } finally {
      isLoading = false;
    }
  });

  function handleFileUpload(event: Event) {
    const target = event.target as HTMLInputElement;
    const files = target.files;
    
    if (files && files[0]) {
      const file = files[0];
      
      // Validate file is an image and under 5MB
      if (!file.type.startsWith('image/')) {
        formErrors.faceImage = 'Please upload an image file.';
        return;
      }
      
      if (file.size > 5 * 1024 * 1024) {
        formErrors.faceImage = 'Image must be under 5MB.';
        return;
      }
      
      faceImage = file;
      formErrors.faceImage = '';
      
      // Create preview
      const reader = new FileReader();
      reader.onload = (e) => {
        faceImagePreview = e.target?.result as string;
      };
      reader.readAsDataURL(file);
    }
  }
  
  function validateForm(): boolean {
    let isValid = true;
    
    // Reset errors
    formErrors = {
      identityCardNumber: '',
      verificationReason: '',
      faceImage: ''
    };
    
    // Validate ID number (simple length check - more validation would be done on server)
    if (!identityCardNumber || identityCardNumber.trim().length < 8) {
      formErrors.identityCardNumber = 'Please enter a valid identity card number (min. 8 characters).';
      isValid = false;
    }
    
    // Validate reason
    if (!verificationReason || verificationReason.trim().length < 10) {
      formErrors.verificationReason = 'Please provide a reason (min. 10 characters).';
      isValid = false;
    }
    
    // Validate image
    if (!faceImage) {
      formErrors.faceImage = 'Please upload an identity verification photo.';
      isValid = false;
    }
    
    return isValid;
  }
  
  async function handleSubmit() {
    if (!validateForm()) return;
    
    isSubmitting = true;
    error = null;
    successMessage = null;
    
    try {
      // TODO: API call to submit premium verification request
      // const formData = new FormData();
      // formData.append('identityCardNumber', identityCardNumber);
      // formData.append('verificationReason', verificationReason);
      // if (faceImage) formData.append('faceImage', faceImage);
      // await api.submitPremiumVerification(formData);
      
      // For demo purposes
      await new Promise(resolve => setTimeout(resolve, 1500)); // Simulate API delay
      
      successMessage = "Your premium verification has been submitted successfully! We'll review it shortly.";
      isPendingVerification = true;
      
      // Reset form
      identityCardNumber = '';
      verificationReason = '';
      faceImage = null;
      faceImagePreview = null;
      
    } catch (err) {
      console.error("Error submitting premium verification:", err);
      error = "Failed to submit verification. Please try again.";
    } finally {
      isSubmitting = false;
    }
  }
</script>

<div class="premium-page">
  <header class="premium-header">
    <!-- <div class="premium-icon"><Star size={28} /></div> -->
    <h1>Premium</h1>
  </header>
  
  {#if isLoading}
    <div class="loading-state">
      <div class="loading-spinner"></div>
      <p>Loading your premium status...</p>
    </div>
  {:else if isPremium}
    <div class="premium-status verified">
      <CheckCircle size={48} />
      <h2>You're verified!</h2>
      <p>Thank you for being a premium member of AY.com</p>
      
      <div class="premium-benefits">
        <h3>Your Premium Benefits</h3>
        <ul>
          <li>
            <Badge size={20} />
            <span>Blue verification checkmark on your profile</span>
          </li>
          <li>
            <CheckCircle size={20} />
            <span>Verified status across the platform</span>
          </li>
          <li>
            <Shield size={20} />
            <span>Enhanced profile security and support</span>
          </li>
        </ul>
      </div>
    </div>
  {:else if isPendingVerification}
    <div class="premium-status pending">
      <AlertCircle size={48} />
      <h2>Verification In Progress</h2>
      <p>We're reviewing your premium verification request. This process typically takes 1-3 business days.</p>
      <div class="premium-benefits">
        <h3>What's Next?</h3>
        <p>Once approved, you'll receive:</p>
        <ul>
          <li>
            <Badge size={20} />
            <span>Blue verification checkmark on your profile</span>
          </li>
          <li>
            <CheckCircle size={20} />
            <span>Verified status across the platform</span>
          </li>
          <li>
            <Shield size={20} />
            <span>Enhanced profile security and support</span>
          </li>
        </ul>
      </div>
    </div>
  {:else}
    <div class="premium-intro">
      <div class="premium-benefits">
        <h2>Get Verified on AY.com</h2>
        <p class="intro-text">Join our premium members and get the blue checkmark for increased recognition and trust on the platform.</p>
        
        <div class="benefits-grid">
          <div class="benefit-card">
            <Badge size={24} />
            <h3>Blue Verification Badge</h3>
            <p>Stand out with an official verification mark on your profile</p>
          </div>
          
          <div class="benefit-card">
            <CheckCircle size={24} />
            <p>Get more visibility and credibility in the community</p>
          </div>
          
          <div class="benefit-card">
            <Shield size={24} />
            <p>Enhanced account security and dedicated support</p>
          </div>
        </div>
      </div>
      
      <div class="verification-form">
        <h2>Submit Verification Request</h2>
        
        {#if error}
          <p class="error-text api-error">{error}</p>
        {/if}
        
        {#if successMessage}
          <p class="success-text">{successMessage}</p>
        {/if}
        
        <form on:submit|preventDefault={handleSubmit}>
          <div class="form-group">
            <label for="identityCardNumber">National Identity Card Number</label>
            <input 
              type="text" 
              id="identityCardNumber"
              bind:value={identityCardNumber}
              placeholder="Enter your ID number"
              disabled={isSubmitting}
            />
            {#if formErrors.identityCardNumber}
              <p class="error-text">{formErrors.identityCardNumber}</p>
            {/if}
            <p class="security-note">
              <Shield size={14} />
              <span>Your ID number will be encrypted and stored securely</span>
            </p>
          </div>
          
          <div class="form-group">
            <label for="verificationReason">Why do you want to be verified?</label>
            <textarea 
              id="verificationReason"
              bind:value={verificationReason}
              placeholder="Explain why you're requesting verification (min. 10 characters)"
              rows="3"
              disabled={isSubmitting}
            ></textarea>
            {#if formErrors.verificationReason}
              <p class="error-text">{formErrors.verificationReason}</p>
            {/if}
          </div>
          
          <div class="form-group">
            <label for="faceImage">Upload verification photo</label>
            <div class="image-upload-container">
              {#if faceImagePreview}
                <div class="image-preview-container">
                  <img src={faceImagePreview} alt="Face verification preview" class="image-preview" />
                  <button 
                    type="button" 
                    class="remove-image-btn" 
                    on:click={() => {faceImage = null; faceImagePreview = null;}}
                    disabled={isSubmitting}
                  >×</button>
                </div>
              {:else}
                <label for="face-upload" class="upload-label" class:disabled={isSubmitting}>
                  <Upload size={24} />
                  <span>Upload a clear photo of your face</span>
                </label>
                <input 
                  type="file"
                  id="face-upload"
                  accept="image/*"
                  on:change={handleFileUpload}
                  disabled={isSubmitting}
                  style="display: none;"
                />
              {/if}
            </div>
            {#if formErrors.faceImage}
              <p class="error-text">{formErrors.faceImage}</p>
            {/if}
          </div>
          
          <button type="submit" class="btn btn-primary submit-btn" disabled={isSubmitting}>
            {isSubmitting ? 'Submitting...' : 'Submit Verification Request'}
          </button>
        </form>
      </div>
    </div>
  {/if}
  
  <div class="premium-faq">
    <h3>Frequently Asked Questions</h3>
    <div class="faq-item">
      <h4>How long does verification take?</h4>
      <p>The verification process typically takes 1-3 business days once we receive your submission.</p>
    </div>
    <div class="faq-item">
      <h4>Is my personal information secure?</h4>
      <p>Yes! Your identity card number is encrypted before storage, and all submitted verification documents are handled securely according to our privacy policy.</p>
    </div>
    <div class="faq-item">
      <h4>What happens if my verification is rejected?</h4>
      <p>If your verification is rejected, you'll receive an email with the reason and instructions on how to reapply if applicable.</p>
    </div>
  </div>
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  .premium-page {
    width: 100%;
    max-width: 800px;
    margin: 0 auto;
    padding: 0 16px 40px;
  }

  .premium-header {
    display: flex;
    align-items: center;
    border-bottom: 1px solid var(--border-color);
    padding: 16px 0;
    margin-bottom: 24px;
    
    .premium-icon {
      color: var(--primary-color);
      margin-right: 12px;
    }
    
    h1 {
      font-size: 20px;
      font-weight: 800;
      margin: 0;
    }
  }
  
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 200px;
    
    .loading-spinner {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      border: 3px solid var(--border-color);
      border-top: 3px solid var(--primary-color);
      animation: spin 1s linear infinite;
      margin-bottom: 16px;
    }
    
    @keyframes spin {
      0% { transform: rotate(0deg); }
      100% { transform: rotate(360deg); }
    }
  }
  
  .premium-status {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 24px 16px;
    border-radius: 16px;
    background-color: var(--section-bg);
    margin-bottom: 28px;
    
    h2 {
      margin: 16px 0 8px;
      font-size: 24px;
      font-weight: bold;
    }
    
    p {
      margin: 0 0 24px;
      color: var(--secondary-text-color);
      max-width: 500px;
    }
    
    &.verified {
      :global(svg) {
        color: var(--primary-color);
      }
    }
    
    &.pending {
      :global(svg) {
        color: orange;
      }
    }
  }
  
  .premium-benefits {
    width: 100%;
    
    h3 {
      font-size: 18px;
      margin: 0 0 16px;
      text-align: left;
    }
    
    ul {
      list-style: none;
      padding: 0;
      margin: 0;
      text-align: left;
      
      li {
        display: flex;
        align-items: center;
        margin-bottom: 16px;
        
        :global(svg) {
          color: var(--primary-color);
          margin-right: 12px;
          flex-shrink: 0;
        }
      }
    }
  }
  
  .premium-intro {
    display: grid;
    grid-template-columns: 1fr;
    gap: 40px;
    margin-bottom: 40px;
    
    @media (min-width: 768px) {
      grid-template-columns: 1fr 1fr;
    }
    
    .intro-text {
      font-size: 16px;
      line-height: 1.5;
      color: var(--secondary-text-color);
      margin-bottom: 24px;
    }
  }
  
  .benefits-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 16px;
    
    @media (min-width: 500px) {
      grid-template-columns: repeat(2, 1fr);
    }
    
    .benefit-card {
      padding: 16px;
      background-color: var(--section-bg);
      border-radius: 12px;
      display: flex;
      flex-direction: column;
      align-items: flex-start;
      
      &:first-child {
        grid-column: 1 / -1;
      }
      
      :global(svg) {
        color: var(--primary-color);
        margin-bottom: 8px;
      }
      
      h3 {
        font-size: 16px;
        margin: 8px 0;
      }
      
      p {
        font-size: 14px;
        margin: 0;
        color: var(--secondary-text-color);
      }
    }
  }
  
  .verification-form {
    background-color: var(--section-bg);
    padding: 20px;
    border-radius: 16px;
    
    h2 {
      font-size: 20px;
      margin: 0 0 20px;
    }
    
    .form-group {
      margin-bottom: 20px;
      
      label {
        display: block;
        font-weight: 600;
        margin-bottom: 8px;
        font-size: 15px;
      }
      
      input, textarea {
        width: 100%;
        padding: 12px;
        border: 1px solid var(--border-color);
        border-radius: 8px;
        background-color: var(--input-bg);
        color: var(--text-color);
        font-size: 15px;
        
        &:focus {
          border-color: var(--primary-color);
          outline: none;
        }
        
        &:disabled {
          opacity: 0.7;
          cursor: not-allowed;
        }
      }
    }
    
    .security-note {
      display: flex;
      align-items: center;
      color: var(--secondary-text-color);
      font-size: 12px;
      margin-top: 8px;
      
      :global(svg) {
        margin-right: 4px;
      }
    }
    
    .image-upload-container {
      border: 2px dashed var(--border-color);
      border-radius: 8px;
      padding: 16px;
      text-align: center;
      
      .upload-label {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 120px;
        cursor: pointer;
        
        :global(svg) {
          margin-bottom: 12px;
          color: var(--secondary-text-color);
        }
        
        span {
          font-size: 14px;
          color: var(--secondary-text-color);
        }
        
        &.disabled {
          opacity: 0.7;
          cursor: not-allowed;
        }
      }
      
      .image-preview-container {
        position: relative;
        
        .image-preview {
          max-width: 100%;
          max-height: 200px;
          border-radius: 8px;
          object-fit: cover;
        }
        
        .remove-image-btn {
          position: absolute;
          top: 8px;
          right: 8px;
          width: 28px;
          height: 28px;
          border-radius: 50%;
          background: rgba(0, 0, 0, 0.5);
          color: #fff;
          border: none;
          font-size: 20px;
          line-height: 1;
          padding: 0;
          cursor: pointer;
          
          &:hover {
            background: rgba(0, 0, 0, 0.7);
          }
          
          &:disabled {
            opacity: 0.5;
            cursor: not-allowed;
          }
        }
      }
    }
    
    .submit-btn {
      width: 100%;
      padding: 12px;
      font-size: 16px;
      font-weight: bold;
      margin-top: 16px;
    }
  }
  
  .premium-faq {
    margin-top: 40px;
    padding-top: 24px;
    border-top: 1px solid var(--border-color);
    
    h3 {
      font-size: 18px;
      margin: 0 0 20px;
    }
    
    .faq-item {
      margin-bottom: 24px;
      
      h4 {
        font-size: 16px;
        margin: 0 0 8px;
        font-weight: 600;
      }
      
      p {
        margin: 0;
        color: var(--secondary-text-color);
        font-size: 14px;
        line-height: 1.5;
        
        a {
          color: var(--primary-color);
          text-decoration: none;
          
          &:hover {
            text-decoration: underline;
          }
        }
      }
    }
  }
  
  .error-text {
    color: var(--error-color);
    font-size: 14px;
    margin: 8px 0 0;
  }
  
  .success-text {
    background-color: var(--success-bg);
    color: var(--success-color);
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 20px;
    font-weight: 500;
  }
  
  .api-error {
    background-color: var(--error-bg);
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 20px;
  }
  
  .btn {
    display: inline-block;
    padding: 0.8rem 1rem;
    border-radius: 9999px;
    text-decoration: none;
    font-weight: bold;
    font-size: 1rem;
    cursor: pointer;
    border: 1px solid transparent;
    transition: background-color 0.2s ease;
    text-align: center;
  }
  
  .btn-primary {
    background-color: var(--primary-color);
    color: var(--primary-button-text);
    border: 1px solid var(--border-color);
    
    &:hover:not(:disabled) {
      background-color: var(--primary-color-hover);
    }
    
    &:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
  }
</style>