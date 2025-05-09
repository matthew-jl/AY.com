<script lang="ts">
  import { createEventDispatcher, onDestroy } from 'svelte';
  import { api, ApiError, type CreateThreadRequestData, type MediaMetadata } from '../lib/api';
  import { user } from '../stores/userStore'; // Get current user for avatar

  const dispatch = createEventDispatcher();

  let content = '';
  let selectedFiles: FileList | null = null;
  let mediaPreviews: { url: string; file: File }[] = [];
  let isUploading = false;
  let uploadError: string | null = null;
  let createError: string | null = null;
  let isLoading = false;
  let charCount = 0;
  const maxChars = 280; // Example limit

  $: charCount = content.length;
  $: progress = Math.min(100, (charCount / maxChars) * 100);
  $: charsLeft = maxChars - charCount;
  $: isOverLimit = charCount > maxChars;

  // TODO: Add state for reply restriction, schedule, community

  // --- Handle File Selection for Previews ---
  function handleFileSelectionChange() {
    if (!selectedFiles) return;

    // Clear previous previews if selecting new files
    mediaPreviews = [];
    uploadError = null; // Clear previous upload errors

    const filesArray = Array.from(selectedFiles);

    // TODO: Add validation for file count, size, type here if needed

    filesArray.forEach(file => {
        // Create object URLs for previewing
        const url = URL.createObjectURL(file);
        mediaPreviews = [...mediaPreviews, { url, file }];
    });

    // Optional: Revoke object URLs later in onDestroy or when removing preview
  }

  // Trigger preview generation when selectedFiles changes
  $: if (selectedFiles) handleFileSelectionChange();

  function removePreview(urlToRemove: string) {
      mediaPreviews = mediaPreviews.filter(p => p.url !== urlToRemove);
      // Also update selectedFiles if needed, though it's usually simpler
      // to just re-select files if the user removes one this way.
      // For simplicity, we'll upload all files currently in mediaPreviews on submit.
       const fileInput = document.getElementById('file-input') as HTMLInputElement;
       if(fileInput) fileInput.value = ''; // Clear input value so user can re-select same file
  }


  async function handleCreateThread() {
    createError = null;
    uploadError = null; // Clear upload error on submit attempt
    if (isOverLimit) {
        createError = "Thread content exceeds maximum length.";
        return;
    }
    if (!content && mediaPreviews.length === 0) {
      createError = "Please add content or media to your thread.";
      return;
    }

    isLoading = true;
    isUploading = true;

     let uploadedMediaIDs: number[] = [];

     try {
        // --- Step 1: Upload Media (if any previews exist) ---
        if (mediaPreviews.length > 0) {
             console.log("Starting media uploads...");
             const uploadPromises = mediaPreviews.map(async (preview) => {
                 console.log(`Uploading ${preview.file.name}...`);
                 const formData = new FormData();
                 formData.append('media_file', preview.file);
                 const response = await api.uploadMedia(formData);
                 console.log(`Uploaded ${preview.file.name}:`, response.media);
                 return response.media.id; // Return only the ID
             });
             uploadedMediaIDs = await Promise.all(uploadPromises);
             console.log("All media uploaded. IDs:", uploadedMediaIDs);
             uploadError = null; // Clear upload error if successful
        }
        isUploading = false; // Upload phase finished

        // --- Step 2: Create Thread with Content and Media IDs ---
        const threadData: CreateThreadRequestData = {
            content: content,
            media_ids: uploadedMediaIDs,
            // TODO: Add reply_restriction, etc.
        };

        console.log("Creating thread with data:", threadData);
        const createdThread = await api.createThread(threadData);
        console.log("Thread created:", createdThread);

        // Success
        dispatch('close');
        dispatch('threadcreated', createdThread);
        // Reset form
        content = ''; selectedFiles = null; mediaPreviews = []; charCount = 0;

    } catch (err) {
        console.error("Error during thread creation/upload:", err);
        if (isUploading) { // Check if error happened during upload phase
             if (err instanceof ApiError) { uploadError = `Upload failed: ${err.message}`; }
             else if (err instanceof Error) { uploadError = `Upload error: ${err.message}`; }
             else { uploadError = 'An unexpected upload error occurred.'; }
        } else { // Error happened during thread creation API call
             if (err instanceof ApiError) { createError = `Failed to create thread: ${err.message}`; }
             else if (err instanceof Error) { createError = `An error occurred: ${err.message}`; }
             else { createError = 'An unexpected error occurred while creating thread.'; }
        }
        // Don't reset form on error so user can retry/fix
    } finally {
        isLoading = false;
        isUploading = false;
    }
  }

  // --- Cleanup ---
  onDestroy(() => {
      // Revoke object URLs to prevent memory leaks
      mediaPreviews.forEach(p => URL.revokeObjectURL(p.url));
  });

</script>

<div class="modal-overlay" 
    role="button" 
    tabindex="0" 
    on:click={() => dispatch('close')}
    on:keydown={(e) => (e.key === 'Enter' || e.key === 'Space') && dispatch('close')}
    >
  <div class="modal-content" 
    role="dialog"
    aria-labelledby="modal-title"
    tabindex="0"
    on:click|stopPropagation
    on:keydown={(e) => e.key === 'Escape' && dispatch('close')}
    >
    <button class="close-button" on:click={() => dispatch('close')} aria-label="Close">×</button>
    <h3>Create Thread</h3>

    <div class="compose-area">
        <div class="avatar-placeholder-small">
             {$user?.name?.charAt(0)?.toUpperCase() ?? '?'}
             <!-- TODO: Replace with actual <img src={$user.profile_picture}> -->
        </div>
        <textarea
            bind:value={content}
            placeholder="What's happening?!"
            rows="4"
            maxlength={maxChars + 50}
        ></textarea>
    </div>

    <!-- Media Previews -->
    {#if mediaPreviews.length > 0}
        <div class="media-preview-grid">
            {#each mediaPreviews as preview (preview.url)} <!-- Key by unique URL -->
                <div class="media-preview-item">
                     <!-- Use preview.url for image src -->
                     <img src={preview.url} alt="Upload preview {preview.file.name}" />
                     <!-- Remove button uses preview.url -->
                    <button class="remove-media-btn" on:click={() => removePreview(preview.url)}>×</button>
                </div>
            {/each}
        </div>
    {/if}
    {#if uploadError} <p class="error-text">{uploadError}</p> {/if}


    <div class="compose-actions">
        <div class="action-icons">
            <label for="file-input" class="icon-button" aria-label="Add media">
                <!-- Simple Image Icon Placeholder -->
                <svg viewBox="0 0 24 24"><g><path d="M19.75 2H4.25C3.01 2 2 3.01 2 4.25v15.5C2 20.99 3.01 22 4.25 22h15.5c1.24 0 2.25-1.01 2.25-2.25V4.25C22 3.01 20.99 2 19.75 2zM4.25 3.5h15.5c.41 0 .75.34.75.75v9.69l-3.31-3.29c-.3-.3-.77-.3-1.06 0l-2.77 2.79-1.94-1.93c-.3-.3-.77-.3-1.06 0L6.56 17H4.25V4.25c0-.41.34-.75.75-.75zm15.5 17H4.25c-.41 0-.75-.34-.75-.75V19h16.5v.75c0 .41-.34.75-.75.75zM8.5 8.5c.83 0 1.5-.67 1.5-1.5S9.33 5.5 8.5 5.5 7 6.17 7 7s.67 1.5 1.5 1.5z"></path></g></svg>
                <input id="file-input" type="file" multiple accept="image/*,video/*" bind:files={selectedFiles} hidden/>
            </label>
             <!-- Placeholder icons for Poll, Emoji, Schedule -->
             <button class="icon-button" aria-label="Add poll" disabled>📊</button>
             <button class="icon-button" aria-label="Add emoji" disabled>😀</button>
             <button class="icon-button" aria-label="Schedule post" disabled>📅</button>
        </div>
        <div class="post-controls">
            <!-- Circular Progress Counter -->
            <div class="progress-container" class:warning={charsLeft <= 20 && charsLeft >= 0} class:over-limit={isOverLimit}>
                {#if charsLeft < 0}
                    <span class="char-count over">{charsLeft}</span>
                {/if}
                 <svg viewBox="0 0 36 36" class="circular-chart">
                    <path class="circle-bg"
                        d="M18 2.0845
                        a 15.9155 15.9155 0 0 1 0 31.831
                        a 15.9155 15.9155 0 0 1 0 -31.831"
                    />
                    <path class="circle"
                        stroke-dasharray="{progress}, 100"
                        d="M18 2.0845
                        a 15.9155 15.9155 0 0 1 0 31.831
                        a 15.9155 15.9155 0 0 1 0 -31.831"
                    />
                </svg>
                 {#if charsLeft <= 20 && charsLeft >=0}
                     <span class="char-count">{charsLeft}</span>
                 {/if}
            </div>
            <button class="btn btn-primary post-btn" on:click={handleCreateThread} disabled={isLoading || isUploading || isOverLimit || (!content && mediaPreviews.length === 0)}>
                {isLoading ? 'Posting...' : 'Post'}
            </button>
        </div>
    </div>
     {#if createError} <p class="error-text api-error">{createError}</p> {/if}

  </div>
</div>

<style lang="scss">
  @use '../styles/variables' as *;

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

  .modal-content {
    background: var(--background);
    color: var(--text-color);
    padding: 20px;
    border-radius: 16px;
    width: 90%;
    max-width: 600px;
    position: relative;
    box-shadow: 0 5px 15px rgba(0, 0, 0, 0.2);
    max-height: 90vh;
    display: flex;
    flex-direction: column;
  }

  .close-button {
    position: absolute;
    top: 10px;
    left: 10px;
    background: transparent;
    border: none;
    font-size: 2rem;
    line-height: 1;
    cursor: pointer;
    color: var(--secondary-text-color);
    padding: 0 5px;
    border-radius: 50%;
     &:hover {
         background-color: var(--section-hover-bg);
     }
  }

  h3 {
      text-align: center;
      margin-bottom: 20px;
      padding-bottom: 10px;
      border-bottom: 1px solid var(--border-color);
      font-weight: bold;
  }

  .compose-area {
      display: flex;
      gap: 10px;
      width: 100%;
       border-bottom: 1px solid var(--border-color);
       padding-bottom: 15px;
       margin-bottom: 10px;
  }

  .avatar-placeholder-small {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      background-color: var(--secondary-text-color);
      color: var(--background);
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: bold;
      flex-shrink: 0;
  }

  textarea {
      flex-grow: 1;
      border: none;
      outline: none;
      resize: none;
      font-size: 1.1rem;
      background: transparent;
      color: var(--text-color);
      font-family: inherit;
      padding: 8px 0;
       &::placeholder {
           color: var(--secondary-text-color);
       }
  }

  .media-preview-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
      gap: 8px;
      margin-bottom: 10px;
      max-height: 180px;
      overflow-y: auto;
  }

  .media-preview-item {
      position: relative;
      aspect-ratio: 1 / 1;
      border-radius: 8px;
      overflow: hidden;
      background-color: var(--section-bg);
      display: flex;
      align-items: center;
      justify-content: center;

      img {
          width: 100%;
          height: 100%;
          object-fit: cover;
      }
      span {
          font-size: 0.8rem;
          color: var(--secondary-text-color);
          font-weight: bold;
      }

      .remove-media-btn {
          position: absolute;
          top: 4px;
          right: 4px;
          background: rgba(0, 0, 0, 0.6);
          color: white;
          border: none;
          border-radius: 50%;
          width: 20px;
          height: 20px;
          font-size: 14px;
          line-height: 18px;
          text-align: center;
          cursor: pointer;
          padding: 0;
           &:hover {
               background: rgba(0, 0, 0, 0.8);
           }
      }
  }

  .compose-actions {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-top: 10px;
  }

  .action-icons {
      display: flex;
      gap: 5px;
  }

  .icon-button {
      background: none;
      border: none;
      padding: 8px;
      border-radius: 50%;
      cursor: pointer;
      color: var(--primary-color);
      display: flex;
      align-items: center;
      justify-content: center;
       &:hover:not(:disabled) {
           background-color: rgba(var(--primary-color-rgb, 29, 155, 240), 0.1);
       }
       &:disabled {
           opacity: 0.5;
           cursor: default;
       }
       svg {
           width: 20px;
           height: 20px;
           fill: currentColor;
       }
       font-size: 1.2rem;
       line-height: 1;
  }


  .post-controls {
      display: flex;
      align-items: center;
      gap: 15px;
  }

  .progress-container {
      position: relative;
      width: 30px;
      height: 30px;
      display: flex;
      align-items: center;
      justify-content: center;
      margin-right: -5px;

      &.warning .circle {
          stroke: orange;
      }
       &.over-limit .circle {
          stroke: var(--error-color);
      }
       &.over-limit .char-count.over {
          color: var(--error-color);
       }
       &.warning .char-count {
            display: block;
       }
  }

  .circular-chart {
      display: block;
      width: 100%;
      height: 100%;
  }

  .circle-bg {
      fill: none;
      stroke: var(--border-color);
      stroke-width: 2.5;
  }

  .circle {
      fill: none;
      stroke: var(--primary-color);
      stroke-width: 2.5;
      stroke-linecap: round;
      transition: stroke-dasharray 0.3s ease, stroke 0.3s ease;
      transform: rotate(-90deg);
      transform-origin: 50% 50%;
  }

  .char-count {
      position: absolute;
      font-size: 11px;
      font-weight: bold;
      color: var(--secondary-text-color);
      display: none;

       &.over {
           display: block;
       }
  }

.btn {
  display: block;
  padding: 0.8rem 1rem;
  border-radius: 9999px;
  text-decoration: none;
  font-weight: bold;
  font-size: 1rem;
  cursor: pointer;
  border: 1px solid transparent;
  transition: background-color 0.2s ease;
  margin-top: 1rem;
//   width: 100%;
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

  .post-btn {
      padding: 8px 16px;
      font-size: 15px;
      min-width: 70px;
      margin-top: 0;
  }

   .error-text { color: var(--error-color); font-size: 0.85rem; margin-top: 4px; }
   .api-error { margin-top: 1rem; text-align: center; font-weight: bold; }


</style>