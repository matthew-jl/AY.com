<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { user as currentUserStore, setUser } from '../stores/userStore';
    import type { UserProfileBasic, UpdateUserProfileRequestData } from '../lib/api';
    import { api, ApiError } from '../lib/api';
    import { tick } from 'svelte';
  
    const dispatch = createEventDispatcher<{ close: void; updated: UserProfileBasic }>();
    export let initialUser: UserProfileBasic;
  
    let name: string;
    let bio: string;
    let currentPassword = '';
    let newPassword = '';
    let confirmNewPassword = '';
    let gender: string;
    let dateOfBirth: string;
    let accountPrivacy: 'public' | 'private';
    let subscribedToNewsletter: boolean;
  
    let profilePictureFile: File | null = null;
    let bannerFile: File | null = null;
    let profilePicturePreview: string | null = null;
    let bannerPreview: string | null = null;
  
    let isLoading = false;
    let isUploading = false;
    let error: string | null = null;
    let passwordError: string | null = null;
  
  
    function initializeForm() {
        name = initialUser.name || '';
        bio = initialUser.bio || '';
        gender = initialUser.gender || '';
        dateOfBirth = initialUser.date_of_birth || '';
        accountPrivacy = initialUser.account_privacy === 'private' ? 'private' : 'public';
        subscribedToNewsletter = initialUser.subscribed_to_newsletter || false;
        profilePicturePreview = initialUser.profile_picture || null;
        bannerPreview = initialUser.banner || null;
        currentPassword = ''; newPassword = ''; confirmNewPassword = ''; passwordError = null;
        profilePictureFile = null; bannerFile = null;
    }
  
    $: if (initialUser) {
        initializeForm();
    }
  
    function handleFilePreview(event: Event, type: 'profile' | 'banner') {
        const input = event.target as HTMLInputElement;
        if (input.files && input.files[0]) {
            const file = input.files[0];
            if (file.size > 5 * 1024 * 1024) { alert(`${type === 'profile' ? 'Profile picture' : 'Banner'} is too large! Max 5MB.`); input.value = ''; return; }
            const reader = new FileReader();
            reader.onload = (e) => {
                if (type === 'profile') { profilePictureFile = file; profilePicturePreview = e.target?.result as string; }
                else { bannerFile = file; bannerPreview = e.target?.result as string; }
            };
            reader.readAsDataURL(file);
        } else {
            if (type === 'profile') { profilePictureFile = null; profilePicturePreview = initialUser.profile_picture || null; }
            else { bannerFile = null; bannerPreview = initialUser.banner || null; }
        }
    }
  
    function validateNewPasswords() {
      if (newPassword || confirmNewPassword) {
          if (!currentPassword) { passwordError = "Current password is required to change password."; return false;}
          if (newPassword.length < 8) { passwordError = "New password must be at least 8 characters."; return false; }
          if (newPassword !== confirmNewPassword) { passwordError = "New passwords do not match."; return false; }
      }
      passwordError = null;
      return true;
    }
  
  
    async function handleSaveChanges() {
      error = null; passwordError = null;
      if (!validateNewPasswords()) return;
  
      if (name.length <= 4) { error = "Name must be longer than 4 characters."; return; }
  
      isLoading = true; isUploading = false;
  
      let uploadedProfilePicUrl: string | undefined | null = initialUser.profile_picture;
      let uploadedBannerUrl: string | undefined | null = initialUser.banner;
  
      const updateData: UpdateUserProfileRequestData = {};
  
      try {
          if (profilePictureFile) {
              isUploading = true;
              const formData = new FormData(); formData.append('media_file', profilePictureFile);
              const response = await api.uploadMedia(formData);
              uploadedProfilePicUrl = response.media.public_url;
          }
          if (bannerFile) {
              isUploading = true;
              const formData = new FormData(); formData.append('media_file', bannerFile);
              const response = await api.uploadMedia(formData);
              uploadedBannerUrl = response.media.public_url;
          }
          isUploading = false;
  
          if (name !== initialUser.name) updateData.name = name;
          if (bio !== (initialUser.bio || '')) updateData.bio = bio;
          if (gender !== initialUser.gender) updateData.gender = gender;
          if (dateOfBirth !== initialUser.date_of_birth) updateData.date_of_birth = dateOfBirth;
          if (accountPrivacy !== initialUser.account_privacy) updateData.account_privacy = accountPrivacy;
          if (subscribedToNewsletter !== initialUser.subscribed_to_newsletter) updateData.subscribed_to_newsletter = subscribedToNewsletter;
          if (uploadedProfilePicUrl !== initialUser.profile_picture) updateData.profile_picture_url = uploadedProfilePicUrl;
          if (uploadedBannerUrl !== initialUser.banner) updateData.banner_url = uploadedBannerUrl;
  
          if (newPassword) {
              updateData.current_password = currentPassword;
              updateData.new_password = newPassword;
          }
  
          if (Object.keys(updateData).length === 0) {
              dispatch('close');
              return;
          }
  
          const updatedUser = await api.updateUserProfile(updateData);
          setUser(updatedUser);
          dispatch('updated', updatedUser);
          dispatch('close');
  
      } catch (err) {
          console.error("Update Profile Error:", err);
          if (err instanceof ApiError) { error = `Update failed: ${err.message}`; }
          else if (err instanceof Error) { error = `An error occurred: ${err.message}`; }
          else { error = 'An unexpected error occurred.'; }
      } finally {
          isLoading = false; isUploading = false;
      }
    }
  
  </script>
  
  <div class="modal-overlay" on:click={() => dispatch('close')}>
    <div class="modal-content edit-profile-modal" on:click|stopPropagation>
      <header class="modal-header">
          <button class="close-btn-header" on:click={() => dispatch('close')} aria-label="Close">×</button>
          <h3>Edit Profile</h3>
          <button class="btn btn-primary save-btn" on:click={handleSaveChanges} disabled={isLoading}>
              {isLoading ? (isUploading ? 'Uploading...' : 'Saving...') : 'Save'}
          </button>
      </header>
  
      <form class="modal-form">
          <!-- Banner Upload -->
          <div class="banner-upload-container">
              {#if bannerPreview}
                  <img src={bannerPreview} alt="Banner preview" class="banner-editor-preview" />
              {:else}
                  <div class="banner-editor-placeholder"></div>
              {/if}
              <label for="bannerFileEdit" class="file-upload-overlay-btn edit-banner-btn">📷</label>
              <input type="file" id="bannerFileEdit" accept="image/*" on:change={(e) => handleFilePreview(e, 'banner')} hidden />
          </div>
  
          <!-- Profile Picture Upload -->
          <div class="profile-pic-upload-container">
              <div class="avatar-editor-wrapper">
                  {#if profilePicturePreview}
                      <img src={profilePicturePreview} alt="Profile preview" class="profile-editor-avatar" />
                  {:else}
                       <div class="profile-editor-avatar-placeholder">{initialUser?.name?.charAt(0)?.toUpperCase() ?? '?'}</div>
                  {/if}
                   <label for="profilePictureFileEdit" class="file-upload-overlay-btn edit-avatar-btn">📷</label>
                   <input type="file" id="profilePictureFileEdit" accept="image/*" on:change={(e) => handleFilePreview(e, 'profile')} hidden />
              </div>
          </div>
  
  
          <!-- Text Fields -->
          <div class="form-group">
              <label for="editName">Name</label>
              <input type="text" id="editName" bind:value={name} required />
          </div>
          <div class="form-group">
              <label for="editBio">Bio</label>
              <textarea id="editBio" bind:value={bio} rows="3" maxlength="160"></textarea>
          </div>
           <!-- DOB, Gender -->
           <div class="form-group">
              <label for="editDob">Date of Birth</label>
              <input type="date" id="editDob" bind:value={dateOfBirth} required />
          </div>
           <div class="form-group">
              <label for="editGender">Gender</label>
              <select id="editGender" bind:value={gender}>
                  <option value="">Select...</option><option value="male">Male</option><option value="female">Female</option>
              </select>
          </div>
  
          <!-- Account Settings -->
           <div class="form-group">
              <label for="editAccountPrivacy">Account Privacy</label>
              <select id="editAccountPrivacy" bind:value={accountPrivacy}>
                  <option value="public">Public</option><option value="private">Private</option>
              </select>
          </div>
           <div class="form-group-checkbox">
              <input type="checkbox" id="editNewsletter" bind:checked={subscribedToNewsletter} />
              <label for="editNewsletter">Subscribe to newsletter</label>
          </div>
  
  
          <!-- Password Change Section (Optional) -->
          <h4 class="section-title">Change Password (optional)</h4>
          <div class="form-group">
              <label for="currentPassword">Current Password</label>
              <input type="password" id="currentPassword" bind:value={currentPassword} placeholder="Leave blank if not changing" />
          </div>
          <div class="form-group">
              <label for="newPassword">New Password</label>
              <input type="password" id="newPassword" bind:value={newPassword} on:input={validateNewPasswords} placeholder="Leave blank if not changing" />
          </div>
          <div class="form-group">
              <label for="confirmNewPassword">Confirm New Password</label>
              <input type="password" id="confirmNewPassword" bind:value={confirmNewPassword} on:input={validateNewPasswords} placeholder="Leave blank if not changing" />
              {#if passwordError} <p class="error-text">{passwordError}</p> {/if}
          </div>
  
          {#if error} <p class="error-text api-error">{error}</p> {/if}
      </form>
    </div>
  </div>
  
  <style lang="scss">
    @use '../styles/variables' as *;
    @use '../styles/auth-forms.scss';
  
    .modal-overlay { 
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.6);
        display: flex;
        justify-content: center;
        align-items: flex-start; 
        padding-top: 5vh; 
        z-index: 1000;
    }

    .modal-content.edit-profile-modal {
      background: var(--background);
      color: var(--text-color);
      padding: 0;
      border-radius: 16px;
      width: 90%;
      max-width: 600px;
      box-shadow: 0 5px 15px rgba(0,0,0,0.2);
      max-height: 90vh;
      display: flex;
      flex-direction: column;
    }
  
    .modal-header {
      display: flex;
      align-items: center;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border-color);
      h3 { flex-grow: 1; text-align: left; margin: 0; font-size: 1.2rem; font-weight: bold; }
      .close-btn-header {
          background: transparent; border: none; font-size: 1.8rem; cursor: pointer; color: var(--text-color);
          padding: 0 8px; margin-right: 15px;
      }
      .save-btn {
          padding: 6px 16px; font-size: 0.9rem; margin-top: 0; width: auto;
          background-color: var(--follow-button-bg); color: var(--follow-button-text);
          border-color: var(--follow-button-border);
           &:hover:not(:disabled) { background-color: var(--follow-button-hover-bg); }
      }
    }
  
    .modal-form {
        padding: 0 16px 16px 16px;
        overflow-y: auto;
        flex-grow: 1;
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }
  
    .banner-upload-container {
        position: relative;
        height: 150px;
        background-color: var(--secondary-text-color);
        margin: 0 -16px 0px -16px;
        .banner-editor-preview { width: 100%; height: 100%; object-fit: cover; }
        .banner-editor-placeholder {}
    }
  
    .profile-pic-upload-container {
      margin-top: -50px;
      padding-left: 16px;
      margin-bottom: 1rem;
      .avatar-editor-wrapper { position: relative; width: 100px; height: 100px; }
      .profile-editor-avatar, .profile-editor-avatar-placeholder {
          width: 100px; height: 100px; border-radius: 50%;
          border: 4px solid var(--background);
          object-fit: cover; background-color: var(--secondary-text-color);
          display: flex; align-items: center; justify-content: center;
          font-size: 2.5rem; font-weight: bold; color: var(--background);
      }
    }
  
    .file-upload-overlay-btn {
        position: absolute;
        background-color: rgba(15, 20, 25, 0.75);
        color: white;
        border: none; border-radius: 50%;
        cursor: pointer;
        display: flex; align-items: center; justify-content: center;
        font-size: 1.2rem;
         &:hover { background-color: rgba(39, 44, 48, 0.75); }
    }
    .edit-banner-btn {
        top: 50%; left: 50%; transform: translate(-50%, -50%);
        width: 40px; height: 40px;
    }
    .edit-avatar-btn {
        bottom: 5px; right: 5px;
        width: 32px; height: 32px; font-size: 1rem;
    }
  
    .form-group-checkbox {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        margin-top: 0.5rem;
        input[type="checkbox"] {
            width: auto;
            accent-color: var(--primary-color);
        }
        label {
            margin-bottom: 0;
            font-weight: normal;
            font-size: 0.9rem;
            color: var(--secondary-text-color);
        }
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
    .section-title {
        font-weight: bold;
        margin-top: 1.5rem;
        margin-bottom: 0.5rem;
        padding-bottom: 0.5rem;
        border-bottom: 1px solid var(--border-color);
    }
  
  </style>