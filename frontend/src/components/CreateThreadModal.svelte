<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { api, ApiError, type CommunityListItem, type CreateThreadRequestData, type MediaMetadata } from '../lib/api';
  import { user } from '../stores/userStore';
  import { CalendarIcon, ChartColumnBig, ImageIcon } from 'lucide-svelte';
  import { user as currentUserStore } from '../stores/userStore';

  const dispatch = createEventDispatcher();

  let content = '';
  let selectedFiles: FileList | null = null;
  let mediaPreviews: { url: string; file: File }[] = [];
  let isUploading = false;
  let uploadError: string | null = null;
  let createError: string | null = null;
  let isLoading = false;
  let charCount = 0;
  const maxChars = 280;

  let scheduledAtDateTime: string | null = null;
  let postTarget: 'personal' | 'community' = 'personal';
  let selectedCommunityId: number | null = null;
  let replyRestriction: 'EVERYONE' | 'FOLLOWING' | 'VERIFIED' = 'EVERYONE';
  
  let selectedCategories: string[] = [];
  const predefinedCategories = [
      { value: 'world', label: 'World' },
      { value: 'sports', label: 'Sports' },
      { value: 'business', label: 'Business' },
      { value: 'sci_tech', label: 'Sci/Tech' },
  ];
  let aiSuggestedCategoryValue: string | null = null;
  let aiSuggestedCategoryLabel: string | null = null;
  let isLoadingAISuggestion = false;
  let aiSuggestionError: string | null = null;
  let categorySuggestionDebounceTimer: number | undefined;

  let joinedCommunities: CommunityListItem[] = [];
  let isLoadingCommunities = false;

  $: charCount = content.length;
  $: progress = Math.min(100, (charCount / maxChars) * 100);
  $: charsLeft = maxChars - charCount;
  $: isOverLimit = charCount > maxChars;

  // TODO: Add state for reply restriction, schedule, community

  function handleFileSelectionChange() {
    if (!selectedFiles) return;

    mediaPreviews = [];
    uploadError = null;

    const filesArray = Array.from(selectedFiles);

    // TODO: Add validation for file count, size, type here if needed

    filesArray.forEach(file => {
        // Create object URLs for previewing
        const url = URL.createObjectURL(file);
        mediaPreviews = [...mediaPreviews, { url, file }];
    });

    // Optional: Revoke object URLs later in onDestroy or when removing preview
  }

  $: if (selectedFiles) handleFileSelectionChange();

  function removePreview(urlToRemove: string) {
      mediaPreviews = mediaPreviews.filter(p => p.url !== urlToRemove);
       const fileInput = document.getElementById('file-input') as HTMLInputElement;
       if(fileInput) fileInput.value = '';
  }

  function toggleCategory(categoryValue: string) {
    const index = selectedCategories.indexOf(categoryValue);
    if (index > -1) {
        selectedCategories = selectedCategories.filter(c => c !== categoryValue);
    } else {
        if (selectedCategories.length < 3) {
             selectedCategories = [...selectedCategories, categoryValue];
        } else {
            alert("You can select up to 3 categories.");
        }
    }
  }

  // AI Category Suggestion
  function getCategorySuggestion() {
    clearTimeout(categorySuggestionDebounceTimer);
    aiSuggestionError = null;
    if (!content.trim() || content.trim().length < 20) {
      aiSuggestedCategoryLabel = null;
      aiSuggestedCategoryValue = null;
      isLoadingAISuggestion = false;
      return;
    }
    isLoadingAISuggestion = true;
    categorySuggestionDebounceTimer = window.setTimeout(async () => {
      try {
        const response = await api.suggestCategory({ text: content.trim() });
        aiSuggestedCategoryLabel = response.predicted_category_name;
        const foundCategory = predefinedCategories.find(cat => cat.label === response.predicted_category_name);
        aiSuggestedCategoryValue = foundCategory ? foundCategory.value : null;

        console.log("AI Suggested Category:", response.predicted_category_name, "Value:", aiSuggestedCategoryValue);

        // Automatically add the AI suggestion if no categories are selected yet
        if (aiSuggestedCategoryValue && selectedCategories.length === 0) {
            if (!selectedCategories.includes(aiSuggestedCategoryValue)) {
                toggleCategory(aiSuggestedCategoryValue);
            }
        }

      } catch (err) {
        console.error("Error getting category suggestion:", err);
        aiSuggestedCategoryLabel = null;
        aiSuggestedCategoryValue = null;
        if (err instanceof ApiError) aiSuggestionError = `Suggestion error: ${err.message}`;
        else aiSuggestionError = "Could not get category suggestion.";
      } finally {
        isLoadingAISuggestion = false;
      }
    }, 800); // 800ms debounce
  }

  // Call suggestion on content change (debounced)
  $: if (content && typeof window !== 'undefined') getCategorySuggestion();

  async function handleCreateThread() {
    createError = null;
    uploadError = null;
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
                 return response.media.id;
             });
             uploadedMediaIDs = await Promise.all(uploadPromises);
             console.log("All media uploaded. IDs:", uploadedMediaIDs);
             uploadError = null;
        }
        isUploading = false;

        let scheduledAtISO: string | null = null;
        if (scheduledAtDateTime) {
            try {
                scheduledAtISO = new Date(scheduledAtDateTime).toISOString();
            } catch (e) {
                createError = "Invalid schedule date/time format.";
                isLoading = false; return;
            }
        }

        // --- Step 2: Create Thread with Content and Media IDs ---
        const threadData: CreateThreadRequestData = {
            content: content,
            media_ids: uploadedMediaIDs,
            categories: selectedCategories.length > 0 ? selectedCategories : undefined,
            scheduled_at: scheduledAtISO,
            community_id: postTarget === 'community' ? selectedCommunityId : null,
            reply_restriction: replyRestriction,
        };

        console.log("Creating thread with data:", threadData);
        const createdThread = await api.createThread(threadData);
        console.log("Thread created:", createdThread);

        dispatch('close');
        dispatch('threadcreated', createdThread);
        content = ''; selectedFiles = null; mediaPreviews = []; charCount = 0;
        selectedCategories = [];
        scheduledAtDateTime = null; postTarget = 'personal'; selectedCommunityId = null;
        replyRestriction = 'EVERYONE';
        aiSuggestedCategoryLabel = null; aiSuggestedCategoryValue = null;

    } catch (err) {
        console.error("Error during thread creation/upload:", err);
        if (isUploading) {
             if (err instanceof ApiError) { uploadError = `Upload failed: ${err.message}`; }
             else if (err instanceof Error) { uploadError = `Upload error: ${err.message}`; }
             else { uploadError = 'An unexpected upload error occurred.'; }
        } else {
             if (err instanceof ApiError) { createError = `Failed to create thread: ${err.message}`; }
             else if (err instanceof Error) { createError = `An error occurred: ${err.message}`; }
             else { createError = 'An unexpected error occurred while creating thread.'; }
        }
    } finally {
        isLoading = false;
        isUploading = false;
    }
  }

  async function fetchJoinedCommunities(userId: number) {
    isLoadingCommunities = true;
    try {
        const response = await api.getJoinedCommunities(userId);
        joinedCommunities = response.communities || [];
    } catch (err) { console.error("Error fetching joined communities:", err); }
    finally { isLoadingCommunities = false; }
  }

  onMount(async () => {
    if ($currentUserStore) {
        fetchJoinedCommunities($currentUserStore.id);
    }
    // Set min value for datetime-local input to now
    const now = new Date();
    now.setMinutes(now.getMinutes() - now.getTimezoneOffset()); // Adjust for local timezone
    const scheduleInput = document.getElementById('schedule-datetime') as HTMLInputElement;
    if (scheduleInput) {
        scheduleInput.min = now.toISOString().slice(0, 16);
    }
  });
  
  onDestroy(() => {
      clearTimeout(categorySuggestionDebounceTimer);
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
            {#if $user}
                {#if $user.profile_picture}
                    <img src="{$user.profile_picture}" alt="{$user.name}" style="width:100%;height:100%;border-radius:50%;" />
                {:else}
                    {$user.name.charAt(0).toUpperCase()}
                {/if}
            {/if}
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
        <div class="media-preview-grid large-preview">
            {#each mediaPreviews as preview (preview.url)}
                <div class="media-preview-item">
                 <img src={preview.url} alt="Upload preview {preview.file.name}" />
                <button class="remove-media-btn" on:click={() => removePreview(preview.url)}>×</button>
                </div>
            {/each}
        </div>
    {/if}
    {#if uploadError} <p class="error-text">{uploadError}</p> {/if}

    <!-- Compact options layout, but keep old category-pills style -->
    <div class="additional-options compact-options">
      <div class="compact-row">
        <div class="option-group compact">
            <label for="postTarget">Post to:</label>
            <select id="postTarget" bind:value={postTarget} on:change={() => { if (postTarget === 'personal') selectedCommunityId = null; }}>
                <option value="personal">Personal</option>
                <option value="community" disabled={joinedCommunities.length === 0}>Community</option>
            </select>
            {#if postTarget === 'community'}
                <select id="communitySelect" bind:value={selectedCommunityId} required disabled={isLoadingCommunities}>
                    <option value={null} disabled>Select community...</option>
                    {#if isLoadingCommunities} <option disabled>Loading...</option> {/if}
                    {#each joinedCommunities as community (community.id)}
                        <option value={community.id}>{community.name}</option>
                    {/each}
                    {#if !isLoadingCommunities && joinedCommunities.length === 0}
                        <option disabled>No communities</option>
                    {/if}
                </select>
            {/if}
        </div>
        <div class="option-group compact">
            <label for="replyRestriction">Who can reply?</label>
            <select id="replyRestriction" bind:value={replyRestriction}>
                <option value="EVERYONE">Everyone</option>
                <option value="FOLLOWING">Following</option>
                <option value="VERIFIED">Verified</option>
            </select>
        </div>
      </div>
      <div class="compact-row">
        <div class="option-group compact">
            <label for="categories">Categories:</label>
            <!-- Keep the old .category-pills style here -->
            <div class="category-pills">
                {#each predefinedCategories as category (category.value)}
                    <button
                        type="button"
                        class="category-pill"
                        class:selected={selectedCategories.includes(category.value)}
                        on:click={() => toggleCategory(category.value)}
                        disabled={selectedCategories.length >= 3 && !selectedCategories.includes(category.value)}
                    >
                        {category.label}
                    </button>
                {/each}
            </div>
        </div>
        <div class="option-group compact">
            <label for="schedule-datetime">Schedule:</label>
            <input type="datetime-local" id="schedule-datetime" bind:value={scheduledAtDateTime} />
        </div>
      </div>
    </div>


    <div class="compose-actions">
        <div class="action-icons">
            <label for="file-input" class="icon-button" aria-label="Add media">
              <ImageIcon />
              <input id="file-input" type="file" multiple accept="image/*,video/*" bind:files={selectedFiles} hidden/>
            </label>
            <button class="icon-button" aria-label="Add poll" disabled>
              <ChartColumnBig />
            </button>
            <button class="icon-button" aria-label="Schedule post" on:click={() => {
                const scheduleInput = document.getElementById('schedule-datetime') as HTMLInputElement;
                    if (scheduleInput) {
                        if (scheduledAtDateTime) scheduledAtDateTime = null;
                        else scheduleInput.click();
                    }
            }}>
              <CalendarIcon />
            </button>
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
      &.large-preview {
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
        max-height: 260px;
        gap: 12px;
      }
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

  .additional-options {
      border-top: 1px solid var(--border-color);
      margin-top: 15px;
      padding-top: 15px;
      display: flex;
      flex-direction: column;
      gap: 1rem;
  }
  .option-group {
      display: flex;
      flex-direction: column;
      gap: 0.3rem;
      label { font-size: 0.9rem; font-weight: 500; color: var(--secondary-text-color); }
      select, input[type="datetime-local"] {
          padding: 8px 10px;
          border: 1px solid var(--border-color);
          border-radius: 6px;
          background-color: var(--input-bg);
          color: var(--text-color);
          font-size: 0.95rem;
           &:focus { outline: none; border-color: var(--primary-color); }
      }
       select#communitySelect { margin-top: 0.3rem; }
  }
  .category-pills {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      .category-pill {
          background-color: var(--section-hover-bg);
          color: var(--text-color);
          border: 1px solid var(--border-color);
          padding: 5px 12px;
          border-radius: 16px;
          font-size: 0.85rem;
          cursor: pointer;
          transition: background-color 0.2s, border-color 0.2s;
           &:hover { border-color: var(--primary-color); }
           &.selected {
               background-color: var(--primary-color-light, rgba(var(--primary-color-rgb),0.15));
               color: var(--primary-color);
               border-color: var(--primary-color);
               font-weight: 600;
           }
           &:disabled {
               opacity: 0.6; cursor: not-allowed;
               background-color: var(--section-bg);
               color: var(--secondary-text-color);
           }
      }
  }

  .compact-options {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-top: 10px;
    padding-top: 10px;
    border-top: 1px solid var(--border-color);
    .compact-row {
      display: flex;
      gap: 1rem;
      flex-wrap: wrap;
      > .option-group.compact {
        flex: 1 1 0;
        min-width: 120px;
        margin-bottom: 0;
        gap: 0.2rem;
        label {
          font-size: 0.85rem;
          margin-bottom: 2px;
        }
        select,
        input[type="datetime-local"] {
          padding: 4px 7px;
          font-size: 0.93rem;
          min-width: 0;
        }
      }
    }
  }

   .error-text { color: var(--error-color); font-size: 0.85rem; margin-top: 4px; }
   .api-error { margin-top: 1rem; text-align: center; font-weight: bold; }

  @media (max-width: 500px) {
    .modal-overlay {
      padding-top: 0;
      align-items: stretch;
    }
    .modal-content {
      max-width: 100vw;
      width: 100vw;
      min-width: 0;
      border-radius: 0;
      padding: 4px;
      box-shadow: none;
      max-height: 100vh;
    }
    h3 {
      font-size: 1.1rem;
      padding-bottom: 6px;
      margin-bottom: 10px;
    }
    .compose-area {
      padding-bottom: 6px;
      margin-bottom: 6px;
    }
    .media-preview-grid {
      gap: 4px;
      max-height: 80px;
    }
    .media-preview-item {
      border-radius: 4px;
    }
    .close-button {
      font-size: 1.5rem;
      top: 4px;
      left: 4px;
      padding: 0 2px;
    }
    .action-icons {
      gap: 2px;
    }
    .icon-button {
      padding: 5px;
      svg {
        width: 16px;
        height: 16px;
      }
    }
    .progress-container {
      width: 18px;
      height: 18px;
    }
    .post-btn {
      font-size: 13px;
      padding: 6px 8px;
      min-width: 48px;
    }
  }
</style>