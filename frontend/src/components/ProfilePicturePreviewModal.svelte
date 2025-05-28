<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    export let imageUrl: string | null;
    export let altText: string = "Profile Picture";
  
    const dispatch = createEventDispatcher<{ close: void }>();
  </script>
  
  {#if imageUrl}
  <div class="modal-overlay preview-overlay" on:click={() => dispatch('close')} role="dialog" aria-modal="true" aria-labelledby="preview-title">
    <div class="modal-content preview-content" on:click|stopPropagation>
      <h3 id="preview-title" class="visually-hidden">{altText} Preview</h3>
      <img src={imageUrl} alt="{altText}" class="preview-image" />
      <button class="close-preview-btn" on:click={() => dispatch('close')} aria-label="Close preview">×</button>
    </div>
  </div>
  {/if}
  
  <style lang="scss">
    .modal-overlay.preview-overlay {
      position: fixed; top: 0; left: 0; width: 100%; height: 100%;
      background-color: rgba(0, 0, 0, 0.85);
      display: flex; justify-content: center; align-items: center;
      z-index: 1100; 
    }
  
    .modal-content.preview-content {
      background: transparent; 
      padding: 0;
      border-radius: 0; 
      max-width: 90vw; 
      max-height: 90vh; 
      position: relative;
      box-shadow: none; 
      display: flex; 
      justify-content: center;
      align-items: center;
    }
  
    .preview-image {
      max-width: 100%;
      max-height: 100%;
      object-fit: contain;
      border-radius: 8px; 
    }
  
    .close-preview-btn {
      position: absolute;
      top: 15px;
      right: 15px;
      background: rgba(0,0,0,0.5);
      color: white;
      border: none;
      border-radius: 50%;
      width: 36px;
      height: 36px;
      font-size: 1.8rem;
      line-height: 36px;
      text-align: center;
      cursor: pointer;
      padding: 0;
      transition: background-color 0.2s ease;
       &:hover { background-color: rgba(0,0,0,0.8); }
    }
  
    .visually-hidden {
      position: absolute;
      width: 1px;
      height: 1px;
      margin: -1px;
      padding: 0;
      overflow: hidden;
      clip: rect(0, 0, 0, 0);
      border: 0;
    }
  </style>