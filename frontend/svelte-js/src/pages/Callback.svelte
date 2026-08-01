<script>
  import { onMount } from "svelte";
  import { navigate } from "svelte-routing";
  import { setAuth } from "../lib/store";
  import api from "../lib/api";
  import { Loader2 } from "@lucide/svelte";

  let errorMsg = "";

  onMount(async () => {
    // The gateway redirects to /callback#access_token=...&refresh_token=...
    const hash = window.location.hash.substring(1);
    const params = new URLSearchParams(hash);
    
    const accessToken = params.get("access_token");
    const refreshToken = params.get("refresh_token");

    if (accessToken && refreshToken) {
      try {
        // We have the tokens, let's fetch the user profile right away
        // to complete the auth state. We'll manually set the token in Axios for this call.
        const response = await api.get("/me", {
          headers: {
            Authorization: `Bearer ${accessToken}`
          }
        });
        
        setAuth({
          accessToken,
          refreshToken,
          user: response.data
        });

        // Clear the hash from the URL for security/cleanliness
        window.history.replaceState(null, "", window.location.pathname);
        navigate("/dashboard", { replace: true });

      } catch (err) {
        errorMsg = "Failed to fetch user profile after social login.";
        console.error(err);
      }
    } else {
      errorMsg = "Authentication failed. No token received.";
    }
  });
</script>

<div class="rounded-3xl bg-white p-8 shadow-xl shadow-zinc-200/50 ring-1 ring-zinc-100 text-center">
  {#if errorMsg}
    <div class="mb-4 rounded-lg bg-red-50 p-4 text-sm text-red-600 ring-1 ring-red-500/20">
      {errorMsg}
    </div>
    <button on:click={() => navigate("/login")} class="text-indigo-600 font-medium hover:underline">
      Return to Login
    </button>
  {:else}
    <Loader2 class="mx-auto h-8 w-8 animate-spin text-indigo-600" />
    <h2 class="mt-4 text-lg font-semibold text-zinc-900">Completing sign in...</h2>
    <p class="mt-2 text-sm text-zinc-500">Please wait while we redirect you.</p>
  {/if}
</div>
