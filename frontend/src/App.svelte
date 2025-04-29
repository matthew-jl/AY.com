<script lang="ts">
  import { Router, Route } from "svelte-routing";
  import Landing from "./routes/Landing.svelte";
  import Login from "./routes/Login.svelte";
  import Register from "./routes/Register.svelte";
  import Home from "./routes/Home.svelte";
  import LeftSidebar from "./components/LeftSidebar.svelte";
  import RightSidebar from "./components/RightSidebar.svelte";

  export let url = "";
</script>

<Router {url}>
  <div class="app">
    <LeftSidebar />
    <main>
        <Route path="/"><Landing /></Route>
        <Route path="/login"><Login /></Route>
        <Route path="/register"><Register /></Route>
        <Route path="/home"><Home /></Route>
        <Route path="/*">
          <h1>404 - Page Not Found</h1>
          <a href="/">Go to Landing</a>
        </Route>
      </main>
      <RightSidebar />
  </div>
</Router>

<style lang="scss">
  @use './styles/variables' as *;

  .app {
    display: flex;
    width: 100%;
  }

  main {
    flex: 1;
    margin-left: $left-sidebar-width;
    margin-right: $right-sidebar-width;
    padding: 20px;
  }
</style>