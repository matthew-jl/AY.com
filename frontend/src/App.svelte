<script lang="ts">
  import { Router, Route, navigate, link } from "svelte-routing";
  import { onDestroy, onMount } from 'svelte';
  import { isAuthenticated, setAuthState } from './stores/authStore';
  import Landing from "./routes/Landing.svelte";
  import Login from "./routes/Login.svelte";
  import Register from "./routes/Register.svelte";
  import Home from "./routes/Home.svelte";
  import LeftSidebar from "./components/LeftSidebar.svelte";
  import RightSidebar from "./components/RightSidebar.svelte";
  import NoAccess from "./routes/NoAccess.svelte";
  import { currentPathname } from "./stores/locationStore";
  import LocationUpdater from "./components/LocationUpdater.svelte";
  import ForgotPassword from "./routes/ForgotPassword.svelte";
  import { closeCreateThreadModal, isCreateThreadModalOpen } from "./stores/modalStore";
  import CreateThreadModal from "./components/CreateThreadModal.svelte";
  import { api, clearTokens, getAccessToken } from "./lib/api";
  import { clearUser, setUser } from "./stores/userStore";

  export let url = "";

  const sidebarLayoutRoutes = [
    '/home', '/explore', '/notifications', '/messages',
    '/bookmarks', '/communities', '/premium', '/profile', '/settings',
    '/'
  ];

   $: showSidebars = isAuth && pathFromStore !== null && sidebarLayoutRoutes.includes(pathFromStore);
  
  // --- State Management ---
  let isAuth = false;
  let pathFromStore: string | null = null;
  let authUnsubscribe: (() => void) | null = null;
  let pathUnsubscribe: (() => void) | null = null;

  // --- Lifecycle ---
  onMount(async () => {
    console.log("App Mounted");

    // --- Rehydrate User Session on Load ---
    const token = getAccessToken();
    if (token) {
        console.log("Token found on mount, attempting to fetch profile...");
        try {
            const userProfile = await api.getUserProfile();
            setUser(userProfile); // Populate user store
            setAuthState(true);   // Confirm auth state
            console.log("User profile rehydrated:", userProfile);
        } catch (err) {
            console.error("Failed to rehydrate user profile on mount:", err);
            // If profile fetch fails (e.g., token expired, server error), treat as logged out
            clearTokens();
            clearUser();
            setAuthState(false);
            // Optionally navigate to login if not on a public page
            if (pathFromStore && !['/login', '/register', '/'].includes(pathFromStore)) {
                navigate('/login', { replace: true });
            }
        }
    } else {
        // No token, ensure logged out state
        clearUser();
        setAuthState(false);
        console.log("No token found on mount, user is logged out.");
    }

    authUnsubscribe = isAuthenticated.subscribe(value => {
      const authChanged = isAuth !== value;
      isAuth = value;
      console.log("Auth state updated:", isAuth);
      if (authChanged && pathFromStore !== null) {
          checkNavigation(pathFromStore, isAuth);
      }
    });

    pathUnsubscribe = currentPathname.subscribe(value => {
        pathFromStore = value;
        console.log("Path store updated:", pathFromStore);
        checkNavigation(pathFromStore, isAuth);
    });
  });

  onDestroy(() => {
    console.log("App Unmounted, unsubscribing.");
    if (authUnsubscribe) authUnsubscribe();
    if (pathUnsubscribe) pathUnsubscribe();
  });

  // --- Navigation Logic ---
  function checkNavigation(path: string | null, authStatus: boolean) {
    if (path === null) {
      console.log("NAV CHECK: Path from store not ready yet.");
      return;
    }

    const isGuestRoute = ['/login', '/register', '/forgot-password', '/'].includes(path);
    const isProtectedRoute = !isGuestRoute;

    console.log(`NAV CHECK (Store): Path=${path}, IsAuth=${authStatus}, IsGuestRoute=${isGuestRoute}, IsProtectedRoute=${isProtectedRoute}`);

    if (authStatus && isGuestRoute && path !== '/') {
      console.log("Redirecting authenticated user from guest route to /home");
      setTimeout(() => navigate('/home', { replace: true }), 0);
    } else if (!authStatus && isProtectedRoute) {
      console.log("Redirecting unauthenticated user from protected route to /login");
      setTimeout(() => navigate('/login', { replace: true }), 0);
    }
  }
</script>

<Router {url}>
  <LocationUpdater />

  <div class="app-container" class:sidebar-layout={showSidebars}>
    {#if showSidebars}
      <LeftSidebar />
    {/if}

    <main class="main-content-area">
      <!-- Guest Routes -->
      <Route path="/"> {#if isAuth} <Home /> {:else} <Landing /> {/if} </Route>
      <Route path="/login"> {#if isAuth} <Home /> {:else} <Login /> {/if} </Route>
      <Route path="/register"> {#if isAuth} <Home /> {:else} <Register /> {/if} </Route>
      <Route path="/forgot-password"> {#if isAuth} <Home /> {:else} <ForgotPassword /> {/if} </Route>

      <!-- Protected Routes -->
      <Route path="/home">
        {#if isAuth} <Home /> {:else} <NoAccess /> {/if}
      </Route>
      <Route path="/explore">
         {#if isAuth} <Home /> <!-- Replace with Explore later --> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/notifications">
         {#if isAuth} <Home /> <!-- Replace with Notifications later --> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/messages">
         {#if isAuth} <Home /> <!-- Replace with Messages later --> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/bookmarks">
         {#if isAuth} <Home /> <!-- Replace with Bookmarks later --> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/communities">
         {#if isAuth} <Home /> <!-- Replace with Communities later --> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/premium">
         {#if isAuth} <Home /> <!-- Replace with Premium later --> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/profile">
         {#if isAuth} <Home /> <!-- Replace with Profile later --> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/settings">
         {#if isAuth} <Home /> <!-- Replace with Settings later --> {:else} <NoAccess /> {/if}
      </Route>

      <Route path="/*">
          {#if isAuth}
            <div class="not-found">
              <h1>404 - Not Found</h1>
              <p>The page you requested could not be found.</p>
              <a href="/home" use:link>Go Home</a>
            </div>
          {:else}
            <NoAccess />
          {/if}
      </Route>
    </main>

    {#if showSidebars}
      <RightSidebar />
    {/if}

    {#if $isCreateThreadModalOpen}
      <CreateThreadModal on:close={closeCreateThreadModal} on:threadcreated={() => console.log('Maybe refresh feed?')} />
    {/if}

  </div>
</Router>

<style lang="scss">
  @use "styles/variables.scss" as *;

  .app-container {
    display: flex;
    min-height: 100vh;
    background-color: var(--background);
    color: var(--text-color);
  }

  .main-content-area {
    flex-grow: 1;
    width: 100%;
  }

  .sidebar-layout .main-content-area {
    margin: 0 auto;
    border-left: 1px solid var(--border-color);
    border-right: 1px solid var(--border-color);
    margin-left: $left-sidebar-width;
  }

  .not-found {
      padding: 30px;
      text-align: center;
      a {
          color: var(--primary-color);
          text-decoration: none;
           &:hover { text-decoration: underline; }
      }
  }

</style>