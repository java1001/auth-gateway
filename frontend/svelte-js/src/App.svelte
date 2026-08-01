<script>
  import { Router, Route } from "svelte-routing";
  import { auth } from "./lib/store";
  
  import Login from "./pages/Login.svelte";
  import Signup from "./pages/Signup.svelte";
  import VerifyEmail from "./pages/VerifyEmail.svelte";
  import ForgotPassword from "./pages/ForgotPassword.svelte";
  import Dashboard from "./pages/Dashboard.svelte";
  import Callback from "./pages/Callback.svelte";
  
  export let url = "";
</script>

<div class="min-h-screen bg-zinc-50 font-sans text-zinc-900 antialiased selection:bg-indigo-500 selection:text-white">
  <Router {url}>
    <main class="flex min-h-screen items-center justify-center p-4">
      <div class="w-full max-w-md">
        <Route path="/" component={Login} />
        <Route path="/login" component={Login} />
        <Route path="/signup" component={Signup} />
        <Route path="/verify-email" component={VerifyEmail} />
        <Route path="/forgot-password" component={ForgotPassword} />
        <Route path="/callback" component={Callback} />
        
        <!-- Protected Route -->
        <Route path="/dashboard">
          {#if $auth.isAuthenticated}
            <Dashboard />
          {:else}
            <Login />
          {/if}
        </Route>
      </div>
    </main>
  </Router>
</div>
