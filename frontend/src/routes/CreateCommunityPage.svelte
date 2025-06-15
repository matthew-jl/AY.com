<script lang="ts">
    import { onMount } from 'svelte';
    import { api, ApiError, type CreateCommunityRequestData, type MediaMetadata } from '../lib/api';
    import { navigate, link } from 'svelte-routing';
    import { user as currentUserStore } from '../stores/userStore';
  
    let name = '';
    let description = '';
    let categoriesInput = ''; // Comma-separated string from user
    let rulesInput = ''; // Newline-separated string from user
  
    let iconFile: File | null = null;
    let bannerFile: File | null = null;
    let iconPreview: string | null = null;
    let bannerPreview: string | null = null;
  
    let isLoading = false;
    let isUploading = false;
    let error: string | null = null;
    let successMessage: string | null = null;
  
    onMount(() => {
      if (!$currentUserStore) {
        navigate('/login', { replace: true }); // User must be logged in
      }
    });
  
    function handleFilePreview(event: Event, type: 'icon' | 'banner') {
      const input = event.target as HTMLInputElement;
      if (input.files && input.files[0]) {
        const file = input.files[0];
        if (file.size > 2 * 1024 * 1024) { // 2MB limit for icon/banner
          alert(`${type === 'icon' ? 'Icon' : 'Banner'} is too large! Max 2MB.`);
          input.value = ''; return;
        }
        const reader = new FileReader();
        reader.onload = (e) => {
          if (type === 'icon') { iconFile = file; iconPreview = e.target?.result as string; }
          else { bannerFile = file; bannerPreview = e.target?.result as string; }
        };
        reader.readAsDataURL(file);
      } else {
        if (type === 'icon') { iconFile = null; iconPreview = null; }
        else { bannerFile = null; bannerPreview = null; }
      }
    }
  
    async function handleCreateCommunity() {
      error = null; successMessage = null;
      if (!name.trim() || !description.trim()) {
        error = "Community name and description are required.";
        return;
      }
      // Basic validation
      if (name.trim().length < 3 || name.trim().length > 100) {
          error = "Community name must be between 3 and 100 characters."; return;
      }
      if (description.trim().length < 10 || description.trim().length > 1000) {
          error = "Description must be between 10 and 1000 characters."; return;
      }
  
  
      isLoading = true; isUploading = false;
  
      let iconUrl: string | null = null;
      let bannerUrl: string | null = null;
  
      try {
        // Step 1: Upload Icon if present
        if (iconFile) {
          isUploading = true;
          const formData = new FormData(); formData.append('media_file', iconFile);
          const response = await api.uploadMedia(formData);
          iconUrl = response.media.public_url;
          console.log("Icon uploaded:", iconUrl);
        }
  
        // Step 2: Upload Banner if present
        if (bannerFile) {
          isUploading = true;
          const formData = new FormData(); formData.append('media_file', bannerFile);
          const response = await api.uploadMedia(formData);
          bannerUrl = response.media.public_url;
          console.log("Banner uploaded:", bannerUrl);
        }
        isUploading = false;
  
        // Step 3: Create Community
        const categoriesArray = categoriesInput.split(',').map(c => c.trim()).filter(c => c);
        const rulesArray = rulesInput.split('\n').map(r => r.trim()).filter(r => r);
  
        const communityData: CreateCommunityRequestData = {
          name: name.trim(),
          description: description.trim(),
          icon_url: iconUrl,
          banner_url: bannerUrl,
          categories: categoriesArray,
          rules: rulesArray,
        };
  
        console.log("Creating community with data:", communityData);
        const createdCommunity = await api.createCommunity(communityData);
        successMessage = `Community "${createdCommunity.name}" created successfully! It's pending approval.`;
        // Redirect to the new community page (or communities list) after a delay
        setTimeout(() => {
          // TODO: The route for a specific community page needs to be defined, e.g., /community/:id or /community/:name
          navigate(`/communities`); // For now, back to communities list
        }, 3000);
  
        // Reset form (optional, as navigating away)
        // name = ''; description = ''; categoriesInput = ''; rulesInput = '';
        // iconFile = null; bannerFile = null; iconPreview = null; bannerPreview = null;
  
      } catch (err) {
        console.error("Create Community Error:", err);
        if (isUploading) {
           if (err instanceof ApiError) { error = `Image upload failed: ${err.message}`; }
           else { error = 'An unexpected image upload error occurred.'; }
        } else {
            if (err instanceof ApiError) { error = `Failed to create community: ${err.message}`; }
            else { error = 'An unexpected error occurred while creating community.'; }
        }
      } finally {
        isLoading = false;
        isUploading = false;
      }
    }
  
  </script>
  
  <div class="page-container create-community-page">
    <header class="page-header-simple">
      <button class="back-button" on:click={() => navigate('/communities')} aria-label="Back to communities">
          ←
      </button>
      <h2>Create Community</h2>
    </header>
  
    <div class="form-content-wrapper">
      <form class="auth-container" style="margin-top: 1rem; max-width: 600px;" on:submit|preventDefault={handleCreateCommunity}>
          <p class="form-instruction">Fill in the details to propose your new community.</p>
  
          <div class="form-group">
              <label for="commName">Community Name</label>
              <input type="text" id="commName" bind:value={name} required minlength="3" maxlength="100" />
          </div>
  
          <div class="form-group">
              <label for="commDescription">Description</label>
              <textarea id="commDescription" bind:value={description} rows="4" required minlength="10" maxlength="1000"></textarea>
          </div>
  
          <div class="form-group">
              <label for="commIcon">Icon (Optional, max 2MB)</label>
              <input type="file" id="commIcon" accept="image/*" on:change={(e) => handleFilePreview(e, 'icon')} />
              {#if iconPreview} <img src={iconPreview} alt="Icon preview" class="image-preview small-preview" /> {/if}
          </div>
  
          <div class="form-group">
              <label for="commBanner">Banner (Optional, max 2MB)</label>
              <input type="file" id="commBanner" accept="image/*" on:change={(e) => handleFilePreview(e, 'banner')} />
              {#if bannerPreview} <img src={bannerPreview} alt="Banner preview" class="image-preview banner-preview-form" /> {/if}
          </div>
  
          <div class="form-group">
              <label for="commCategories">Categories (comma-separated, e.g., gaming, webdev, anime)</label>
              <input type="text" id="commCategories" bind:value={categoriesInput} placeholder="gaming, webdev, anime" />
          </div>
  
          <div class="form-group">
              <label for="commRules">Rules (one rule per line, without numbers)</label>
              <textarea id="commRules" bind:value={rulesInput} rows="5" placeholder="Be respectful.
No spam."></textarea>
          </div>
  
          {#if error}
              <p class="error-text api-error">{error}</p>
          {/if}
          {#if successMessage}
              <p class="success-text">{successMessage}</p>
          {/if}
  
          <button type="submit" class="btn btn-primary" disabled={isLoading}>
              {isLoading ? (isUploading ? 'Uploading Images...' : 'Submitting...') : 'Create Community'}
          </button>
      </form>
    </div>
  
  </div>
  
  <style lang="scss">
  @use '../styles/variables' as *;
  @use '../styles/auth-forms.scss';

  .page-container.create-community-page {
    width: 100%;
    display: flex;
    flex-direction: column;
    min-height: 100vh;
  }

  .page-header-simple {
    display: flex;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-color);
    background-color: var(--background);
    position: sticky;
    top: 0;
    z-index: 10;

    .back-button {
      background: none; 
      border: none; 
      font-size: 1.5rem;
      margin-right: 16px; 
      cursor: pointer; 
      color: var(--text-color);
      padding: 4px;
      display: flex;
      align-items: center;
      justify-content: center;
      &:hover { 
        background-color: var(--section-hover-bg); 
        border-radius: 50%;
      }
    }
    
    h2 { 
      font-size: 1.25rem; 
      font-weight: bold; 
      margin: 0; 
    }
  }

  .form-content-wrapper {
    padding: 16px;
    overflow-y: auto;
    flex-grow: 1;
    display: flex;
    justify-content: center;
  }

  .form-instruction {
    text-align: center;
    color: var(--secondary-text-color);
    margin-bottom: 1.5rem;
    font-size: 0.95rem;
  }

  .image-preview {
    max-width: 80px;
    max-height: 80px;
    border-radius: 8px; 
    margin-top: 8px; 
    object-fit: cover;
    border: 1px solid var(--border-color);
  }
  
  .banner-preview-form {
    max-width: 100%; 
    max-height: 120px;
  }
  
  .small-preview {
    border-radius: 50%;
  }

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

  textarea {
    resize: vertical;
    min-height: 80px;
  }

  /* Responsive styles */
  @media (max-width: 768px) {
    .form-content-wrapper {
      padding: 12px;
    }
    
    :global(.auth-container) {
      margin-top: 0.5rem !important;
      padding: 1.5rem !important;
      width: 100% !important;
      max-width: 550px !important;
    }
    
    .page-header-simple {
      padding: 10px 14px;
      
      h2 {
        font-size: 1.2rem;
      }
    }
    
    .form-instruction {
      font-size: 0.9rem;
      margin-bottom: 1.2rem;
    }
    
    :global(.form-group) {
      margin-bottom: 1rem;
    }
    
    :global(.btn) {
      font-size: 15px;
    }
  }

  @media (max-width: 576px) {
    .form-content-wrapper {
      padding: 10px;
    }
    
    :global(.auth-container) {
      padding: 1.2rem !important;
      margin-top: 0 !important;
    }
    
    .page-header-simple {
      padding: 8px 12px;
      
      .back-button {
        font-size: 1.3rem;
        margin-right: 12px;
      }
      
      h2 {
        font-size: 1.1rem;
      }
    }
    
    .banner-preview-form {
      max-height: 100px;
    }
    
    .form-instruction {
      font-size: 0.85rem;
      margin-bottom: 1rem;
    }
    
    :global(.form-group) {
      margin-bottom: 0.9rem;
      
      label {
        font-size: 0.9rem;
        margin-bottom: 4px;
      }
      
      input,
      textarea {
        padding: 8px 10px;
        font-size: 14px;
      }
    }
    
    .success-text {
      padding: 0.7rem;
      font-size: 14px;
    }
    
    .error-text.api-error {
      font-size: 14px;
    }
    
    :global(.btn) {
      padding: 10px 14px;
      font-size: 14px;
    }
  }

  @media (max-width: 430px) {
    .form-content-wrapper {
      padding: 8px;
    }
    
    :global(.auth-container) {
      padding: 1rem 0.9rem !important;
      border: none !important;
      box-shadow: none !important;
      background-color: transparent !important;
    }
    
    .page-header-simple {
      padding: 8px 10px;
      
      .back-button {
        font-size: 1.2rem;
        margin-right: 10px;
      }
      
      h2 {
        font-size: 1rem;
      }
    }
    
    .image-preview {
      max-width: 70px;
      max-height: 70px;
    }
    
    .banner-preview-form {
      max-height: 80px;
    }
    
    :global(.form-group) {
      margin-bottom: 0.8rem;
      
      label {
        font-size: 0.85rem;
      }
      
      input,
      textarea {
        padding: 7px 9px;
        font-size: 13px;
      }
    }
    
    .form-instruction {
      font-size: 0.8rem;
      margin-bottom: 0.9rem;
    }
    
    .success-text {
      padding: 0.6rem;
      font-size: 13px;
    }
    
    .error-text.api-error {
      font-size: 13px;
    }
    
    :global(.btn) {
      padding: 9px 12px;
      font-size: 13px;
    }
  }
</style>