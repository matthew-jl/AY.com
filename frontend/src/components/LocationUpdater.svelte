<script lang="ts">
  import { useLocation } from 'svelte-routing';
  import { currentPathname } from '../stores/locationStore';

  const location = useLocation();

  // Reactive statement: Whenever the location from the router changes...
  $: {
    // Check if location is available before updating the store
    if ($location) {
      console.log("LocationUpdater: Path changed to", $location.pathname);
      // Update our custom store with the new pathname
      currentPathname.set($location.pathname);
    }
  }
</script>