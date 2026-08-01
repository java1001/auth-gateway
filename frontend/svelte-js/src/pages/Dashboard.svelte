<script>
  import { onMount } from "svelte";
  import { navigate } from "svelte-routing";
  import api from "../lib/api";
  import { auth, clearAuth } from "../lib/store";
  import { LogOut, User, ShieldCheck } from "@lucide/svelte";

  let profile = $auth.user || null;
  let isLoading = false;
  let errorMsg = "";

  onMount(async () => {
    // If we don't have the profile or we just want to fetch the latest
    isLoading = true;
    try {
      const response = await api.get("/me");
      profile = response.data;
    } catch (err) {
      errorMsg = "Failed to load profile. Your session may have expired.";
      console.error(err);
    } finally {
      isLoading = false;
    }
  });

  const handleLogout = async () => {
    try {
      // Best effort logout on server to blocklist token
      await api.post("/logout");
    } catch (err) {
      console.error("Server logout failed, clearing local state anyway.");
    } finally {
      clearAuth();
      navigate("/login", { replace: true });
    }
  };
</script>

<div class="w-full max-w-4xl rounded-3xl bg-white shadow-xl shadow-zinc-200/50 ring-1 ring-zinc-100 overflow-hidden">
  <div class="bg-indigo-600 px-8 py-10 text-white sm:px-12 sm:py-16">
    <div class="flex flex-col items-center sm:flex-row sm:justify-between">
      <div>
        <h1 class="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p class="mt-2 text-indigo-100">Welcome back, you are securely logged in.</p>
      </div>
      <button
        on:click={handleLogout}
        class="mt-6 flex items-center rounded-xl bg-white/10 px-4 py-2.5 text-sm font-semibold text-white shadow-sm ring-1 ring-inset ring-white/20 hover:bg-white/20 sm:mt-0 transition-all duration-200"
      >
        <LogOut class="mr-2 h-4 w-4" />
        Sign out
      </button>
    </div>
  </div>

  <div class="px-8 py-10 sm:px-12">
    {#if isLoading}
      <div class="flex animate-pulse space-x-4">
        <div class="h-12 w-12 rounded-full bg-zinc-200"></div>
        <div class="flex-1 space-y-4 py-1">
          <div class="h-4 rounded bg-zinc-200 w-3/4"></div>
          <div class="h-4 rounded bg-zinc-200 w-1/2"></div>
        </div>
      </div>
    {:else if errorMsg}
      <div class="rounded-xl bg-red-50 p-4 text-sm text-red-600 ring-1 ring-red-500/20">
        {errorMsg}
      </div>
    {:else if profile}
      <div class="grid gap-6 sm:grid-cols-2">
        <div class="rounded-2xl bg-zinc-50 p-6 ring-1 ring-zinc-200">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-full bg-indigo-100 text-indigo-600">
              <User size={24} />
            </div>
            <div>
              <p class="text-sm font-medium text-zinc-500">Account ID</p>
              <p class="font-mono text-sm font-semibold text-zinc-900 truncate max-w-[200px]">{profile.id}</p>
            </div>
          </div>
        </div>
        
        <div class="rounded-2xl bg-zinc-50 p-6 ring-1 ring-zinc-200">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 text-emerald-600">
              <ShieldCheck size={24} />
            </div>
            <div>
              <p class="text-sm font-medium text-zinc-500">Security Status</p>
              <p class="text-sm font-semibold text-zinc-900">
                {profile.verified ? 'Email Verified' : 'Unverified Email'}
              </p>
            </div>
          </div>
        </div>
      </div>
      
      <div class="mt-8">
        <h3 class="text-base font-semibold leading-7 text-zinc-900">Profile Information</h3>
        <p class="mt-1 max-w-2xl text-sm leading-6 text-zinc-500">Details retrieved from the Auth Gateway API.</p>
        
        <div class="mt-6 border-t border-zinc-100">
          <dl class="divide-y divide-zinc-100">
            <div class="px-4 py-6 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-0">
              <dt class="text-sm font-medium leading-6 text-zinc-900">Email address</dt>
              <dd class="mt-1 text-sm leading-6 text-zinc-700 sm:col-span-2 sm:mt-0">{profile.email}</dd>
            </div>
            <div class="px-4 py-6 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-0">
              <dt class="text-sm font-medium leading-6 text-zinc-900">Tenant Site</dt>
              <dd class="mt-1 text-sm leading-6 text-zinc-700 sm:col-span-2 sm:mt-0">{profile.site}</dd>
            </div>
            <div class="px-4 py-6 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-0">
              <dt class="text-sm font-medium leading-6 text-zinc-900">Joined at</dt>
              <dd class="mt-1 text-sm leading-6 text-zinc-700 sm:col-span-2 sm:mt-0">
                {new Date(profile.created_at).toLocaleDateString()}
              </dd>
            </div>
          </dl>
        </div>
      </div>
    {/if}
  </div>
</div>
