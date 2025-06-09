<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import type { MediaMetadata } from '../lib/api'; // If needed for preview struct
  import { Paperclip } from 'lucide-svelte';
  
    export let currentContent = ''; // Two-way bind this from parent
    export let isSending = false;   // To disable input/button while sending
  
    // For file previews
    export let selectedFiles: FileList | null = null; // Parent will manage the actual FileList
    export let mediaPreviews: { url: string; file: File; type: 'image' | 'video' | 'other' }[] = []; // Parent passes this for display
  
    const dispatch = createEventDispatcher<{
      send: { content: string; filesToUpload: File[] }; // Pass File objects
      filesselected: FileList; // Dispatch raw FileList
      removepreview: string; // Dispatch url to remove
    }>();
  
    function handleInput(event: Event) {
      const textarea = event.target as HTMLTextAreaElement;
      currentContent = textarea.value; // Update parent's bound variable
      // Auto-resize textarea
      textarea.style.height = 'auto';
      textarea.style.height = `${textarea.scrollHeight}px`;
    }
  
    function sendMessage() {
      if (isSending || (currentContent.trim() === '' && mediaPreviews.length === 0)) return;
      // Parent (MessagesPage) will handle actual file uploads and API call.
      // This component just gathers the data.
      const filesToPass: File[] = mediaPreviews.map(p => p.file);
      dispatch('send', { content: currentContent.trim(), filesToUpload: filesToPass });
      // Parent should clear currentContent and mediaPreviews after successful send
    }
  
    function handleFileSelect(event: Event) {
        const input = event.target as HTMLInputElement;
        if (input.files) {
            dispatch('filesselected', input.files); // Dispatch FileList to parent
            input.value = ''; // Clear the input so same file can be selected again
        }
    }
  
    function removePreview(urlToRemove: string) {
      // Dispatch an event so parent can update mediaPreviews and selectedFiles
      dispatch('removepreview', urlToRemove);
    }
  
  </script>
  
  <footer class="message-input-area">
    <!-- Media Previews (above input) -->
    {#if mediaPreviews.length > 0}
      <div class="input-media-previews">
        {#each mediaPreviews as preview (preview.url)}
          <div class="input-media-item">
            {#if preview.type === 'image'}
              <img src={preview.url} alt="Upload preview {preview.file.name}" />
            {:else if preview.type === 'video'}
              <div class="video-preview-icon">▶️<span>Video</span></div>
            {:else}
              <div class="file-preview-icon">📄<span>{preview.file.name}</span></div>
            {/if}
            <button class="remove-preview-btn" on:click={() => removePreview(preview.url)}>×</button>
          </div>
        {/each}
      </div>
    {/if}
  
    <div class="input-row">
      <label for="message-file-input-comp" class="attach-btn" title="Attach file">
        <Paperclip size={20} />
        <input type="file" id="message-file-input-comp" multiple accept="image/*,video/*" on:change={handleFileSelect} hidden />
      </label>
      <textarea
        bind:value={currentContent}
        placeholder="Send a message..."
        rows="1"
        on:input={handleInput}
        on:keydown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage(); } }}
        disabled={isSending}
      ></textarea>
      <button class="send-btn" on:click={sendMessage} disabled={isSending || (currentContent.trim() === '' && mediaPreviews.length === 0)}>
        <!-- Send Icon -->
        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"></path></svg>
      </button>
    </div>
  </footer>
  
  <style lang="scss">
    @use '../styles/variables' as *;
  
    .message-input-area {
      display: flex;
      flex-direction: column; /* Stack previews above input row */
      padding: 8px 12px; /* Reduced horizontal padding */
      border-top: 1px solid var(--border-color);
      background-color: var(--background);
    }
  
    .input-media-previews {
      display: flex;
      gap: 8px;
      padding: 0px 0px 8px 40px;
      overflow-x: auto;
      /* Simple scrollbar for previews */
      scrollbar-width: thin;
      scrollbar-color: var(--scrollbar-thumb-color) var(--background);
       &::-webkit-scrollbar { height: 6px; }
       &::-webkit-scrollbar-thumb { background-color: var(--scrollbar-thumb-color); border-radius: 3px;}
    }
  
    .input-media-item {
      position: relative;
      width: 60px; height: 60px;
      border-radius: 8px;
      overflow: hidden;
      background-color: var(--section-bg);
      flex-shrink: 0; /* Prevent shrinking */
      display: flex; align-items: center; justify-content: center;
      img { width: 100%; height: 100%; object-fit: cover; }
      .video-preview-icon, .file-preview-icon {
          display: flex; flex-direction: column; align-items: center;
          font-size: 1.5rem; color: var(--secondary-text-color);
          span { font-size: 0.7rem; margin-top: 2px; max-width: 50px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;}
      }
      .remove-preview-btn { /* From CreateThreadModal */
          position: absolute; top: 2px; right: 2px; background: rgba(0,0,0,0.6); color: white;
          border: none; border-radius: 50%; width: 18px; height: 18px; font-size: 12px;
          line-height: 16px; text-align: center; cursor: pointer; padding: 0;
           &:hover { background: rgba(0,0,0,0.8); }
      }
    }
  
  
    .input-row {
        display: flex;
        align-items: flex-end; /* Align items to bottom for better multi-line textarea experience */
        gap: 8px;
    }
  
    textarea {
      flex-grow: 1;
      padding: 10px;
      border: 1px solid var(--border-color);
      border-radius: 20px;
      resize: none;
      min-height: 20px; /* Match line-height + padding */
      max-height: 100px; /* Example: approx 5 lines */
      font-size: 15px;
      line-height: 20px; /* For better auto-resize */
      background-color: var(--input-bg);
      color: var(--text-color);
      &:focus { outline: none; border-color: var(--primary-color); }
      scrollbar-width: none; /* Hide scrollbar Firefox */
       &::-webkit-scrollbar { display: none; }
    }
  
    .attach-btn, .send-btn {
      background: none; border: none; cursor: pointer; padding: 8px;
      display: flex; align-items: center; justify-content: center;
      color: var(--primary-color);
      font-size: 1.3rem;
      border-radius: 50%;
      width: 40px; /* Fixed size */
      height: 40px;
      flex-shrink: 0;
      align-self: flex-end; /* Align with bottom of textarea */
      margin-bottom: 1px; /* Fine-tune vertical alignment */
      //  &:hover:not(:disabled) { background-color: var(--section-hover-bg); }
       &:disabled { color: var(--secondary-text-color); cursor: default; }
    }
    .send-btn svg {
        width: 20px; height: 20px;
    }
  </style>