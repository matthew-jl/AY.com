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
  import ProfilePage from "./routes/ProfilePage.svelte";
  import ExplorePage from "./routes/ExplorePage.svelte";
  import BookmarksPage from "./routes/BookmarksPage.svelte";
  import NotificationsPage from "./routes/NotificationsPage.svelte";
  import MessagesPage from "./routes/MessagesPage.svelte";
  import CommunitiesPage from "./routes/CommunitiesPage.svelte";
  import CreateCommunityPage from "./routes/CreateCommunityPage.svelte";
  import CommunityDetailPage from "./routes/CommunityDetailPage.svelte";
  import PremiumPage from "./routes/PremiumPage.svelte";
  import ThreadDetailPage from "./components/ThreadDetailPage.svelte";
  import SettingsPage from "./routes/SettingsPage.svelte";

  export let url = "";

  const sidebarLayoutRoutes = [
    '/home', '/explore', '/notifications', '/messages',
    '/bookmarks', '/communities', '/premium', '/settings', '/'
  ];

  function matchesSidebarRoute(path: string | null): boolean {
    if (!path) return false;
    if (sidebarLayoutRoutes.includes(path)) return true;
    // Match /profile/:username
    if (/^\/profile\/[^/]+$/.test(path)) return true;
    // Match /community/:id
    if (/^\/community\/[^/]+$/.test(path)) return true;
    // Match /thread/:id
    if (/^\/thread\/[^/]+$/.test(path)) return true;
    return false;
  }

  $: showSidebars = isAuth && matchesSidebarRoute(pathFromStore);
  
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
            const userProfileApiResponse = await api.getOwnUserProfile();
            setUser(userProfileApiResponse.user);
            setAuthState(true);
            console.log("User profile rehydrated:", userProfileApiResponse.user);
        } catch (err) {
            console.error("Failed to rehydrate user profile on mount:", err);
            clearTokens();
            clearUser();
            setAuthState(false);
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
         {#if isAuth} <ExplorePage /> {:else} <NoAccess /> {/if}
      </Route>
      <Route path="/thread/:id" let:params>
          <ThreadDetailPage threadId={params.id} />
      </Route>
       <Route path="/notifications">
         {#if isAuth} <NotificationsPage /> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/messages">
         {#if isAuth} <MessagesPage /> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/bookmarks">
         {#if isAuth} <BookmarksPage /> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/communities">
         {#if isAuth} <CommunitiesPage /> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/communities/create">
         {#if isAuth} <CreateCommunityPage /> {:else} <NoAccess /> {/if}
      </Route>
      <Route path="/community/:id" let:params>
          <CommunityDetailPage id={params.id} />
      </Route>
       <Route path="/premium">
         {#if isAuth} <PremiumPage /> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/profile/:username">
         {#if isAuth} <ProfilePage /> {:else} <NoAccess /> {/if}
      </Route>
       <Route path="/settings">
         {#if isAuth} <SettingsPage /> {:else} <NoAccess /> {/if}
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
    min-width: 0;
    // padding: 0 16px;
    box-sizing: border-box;
  }

  .sidebar-layout .main-content-area {
    margin: 0 auto;
    border-left: 1px solid var(--border-color);
    border-right: 1px solid var(--border-color);
    margin-left: $left-sidebar-width;
    margin-right: 0;
    max-width: 700px;
    min-width: 0;
    // padding: 0 24px;
  }

  @media (max-width: 1200px) {
    .sidebar-layout .main-content-area {
      margin-left: $left-sidebar-width;
      margin-right: 0;
      max-width: 100vw;
      // padding: 0 12px;
    }
  }

  @media (max-width: 900px) {
    .sidebar-layout .main-content-area {
      margin-left: 70px;
      margin-right: 0;
      max-width: 100vw;
      // padding: 0 6px;
    }
  }

  @media (max-width: 600px) {
    .main-content-area,
    .sidebar-layout .main-content-area {
      margin-left: 70px;
      // padding: 0 2vw;
      border-left: none;
      border-right: none;
      max-width: 100vw;
    }
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