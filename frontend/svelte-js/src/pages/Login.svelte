<script>
  import { Link, navigate } from "svelte-routing";
  import api from "../lib/api";
  import { setAuth } from "../lib/store";
  import { Mail, Lock, Loader2, ArrowRight } from "@lucide/svelte";

  let email = "";
  let password = "";
  let isLoading = false;
  let errorMsg = "";

  const handleLogin = async (e) => {
    e.preventDefault();
    isLoading = true;
    errorMsg = "";

    try {
      const response = await api.post("/login", { email, password });
      setAuth({
        accessToken: response.data.access_token,
        refreshToken: response.data.refresh_token,
        user: response.data.user,
      });
      navigate("/dashboard");
    } catch (err) {
      errorMsg = err.response?.data?.error || "Login failed. Please try again.";
    } finally {
      isLoading = false;
    }
  };

  const handleSocialLogin = (provider) => {
    const site = window.location.hostname;
    // Assuming backend is at http://localhost:8080 during dev. 
    // In prod this would just be /auth/${provider}/${site}/login
    window.location.href = `http://localhost:8080/auth/${provider}/${site}/login`;
  };
</script>

<div class="rounded-3xl bg-white p-8 shadow-xl shadow-zinc-200/50 ring-1 ring-zinc-100">
  <div class="mb-8 text-center">
    <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-50 text-indigo-600">
      <Lock size={24} />
    </div>
    <h2 class="mt-4 text-2xl font-bold tracking-tight text-zinc-900">Welcome back</h2>
    <p class="mt-2 text-sm text-zinc-500">Sign in to your account to continue</p>
  </div>

  {#if errorMsg}
    <div class="mb-6 rounded-lg bg-red-50 p-4 text-sm text-red-600 ring-1 ring-red-500/20">
      {errorMsg}
    </div>
  {/if}

  <form on:submit={handleLogin} class="space-y-5">
    <div>
      <label for="email" class="block text-sm font-medium text-zinc-700">Email address</label>
      <div class="relative mt-2">
        <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-zinc-400">
          <Mail size={18} />
        </div>
        <input
          id="email"
          type="email"
          bind:value={email}
          required
          placeholder="you@example.com"
          class="block w-full rounded-xl border-0 py-2.5 pl-10 text-zinc-900 shadow-sm ring-1 ring-inset ring-zinc-300 placeholder:text-zinc-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6 transition-all duration-200"
        />
      </div>
    </div>

    <div>
      <div class="flex items-center justify-between">
        <label for="password" class="block text-sm font-medium text-zinc-700">Password</label>
        <Link to="/forgot-password" class="text-sm font-medium text-indigo-600 hover:text-indigo-500">
          Forgot password?
        </Link>
      </div>
      <div class="relative mt-2">
        <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-zinc-400">
          <Lock size={18} />
        </div>
        <input
          id="password"
          type="password"
          bind:value={password}
          required
          placeholder="••••••••"
          class="block w-full rounded-xl border-0 py-2.5 pl-10 text-zinc-900 shadow-sm ring-1 ring-inset ring-zinc-300 placeholder:text-zinc-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6 transition-all duration-200"
        />
      </div>
    </div>

    <button
      type="submit"
      disabled={isLoading}
      class="group flex w-full justify-center rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:opacity-70 transition-all duration-200"
    >
      {#if isLoading}
        <Loader2 class="mr-2 animate-spin" size={20} />
        Signing in...
      {:else}
        Sign in
        <ArrowRight class="ml-2 opacity-0 -translate-x-2 group-hover:opacity-100 group-hover:translate-x-0 transition-all duration-200" size={20} />
      {/if}
    </button>
  </form>

  <div class="mt-8">
    <div class="relative">
      <div class="absolute inset-0 flex items-center" aria-hidden="true">
        <div class="w-full border-t border-zinc-200"></div>
      </div>
      <div class="relative flex justify-center text-sm font-medium leading-6">
        <span class="bg-white px-6 text-zinc-500">Or continue with</span>
      </div>
    </div>

    <div class="mt-6 grid grid-cols-2 gap-4">
      <button
        on:click={() => handleSocialLogin('google')}
        class="flex w-full items-center justify-center gap-3 rounded-xl bg-white px-3 py-2 text-sm font-semibold text-zinc-900 shadow-sm ring-1 ring-inset ring-zinc-300 hover:bg-zinc-50 transition-colors duration-200"
      >
        <svg class="h-5 w-5" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12.0003 4.75C13.7703 4.75 15.3553 5.36002 16.6053 6.54998L20.0303 3.125C17.9502 1.19 15.2353 0 12.0003 0C7.31028 0 3.25527 2.69 1.28027 6.60998L5.27028 9.70498C6.21525 6.86002 8.87028 4.75 12.0003 4.75Z" fill="#EA4335" />
          <path d="M23.49 12.275C23.49 11.49 23.415 10.73 23.3 10H12V14.51H18.47C18.18 15.99 17.34 17.25 16.08 18.1L19.945 21.1C22.2 19.01 23.49 15.92 23.49 12.275Z" fill="#4285F4" />
          <path d="M5.26498 14.2949C5.02498 13.5699 4.88501 12.7999 4.88501 11.9999C4.88501 11.1999 5.01998 10.4299 5.26498 9.7049L1.275 6.60986C0.46 8.22986 0 10.0599 0 11.9999C0 13.9399 0.46 15.7699 1.28 17.3899L5.26498 14.2949Z" fill="#FBBC05" />
          <path d="M12.0004 24.0001C15.2404 24.0001 17.9654 22.935 19.9454 21.095L16.0804 18.095C15.0054 18.82 13.6204 19.245 12.0004 19.245C8.8704 19.245 6.21537 17.135 5.2654 14.29L1.27539 17.385C3.25539 21.31 7.3104 24.0001 12.0004 24.0001Z" fill="#34A853" />
        </svg>
        <span class="text-sm font-semibold leading-6">Google</span>
      </button>

      <button
        on:click={() => handleSocialLogin('twitter')}
        class="flex w-full items-center justify-center gap-3 rounded-xl bg-white px-3 py-2 text-sm font-semibold text-zinc-900 shadow-sm ring-1 ring-inset ring-zinc-300 hover:bg-zinc-50 transition-colors duration-200"
      >
        <svg class="h-5 w-5 fill-current" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
        </svg>
        <span class="text-sm font-semibold leading-6">X</span>
      </button>
    </div>
  </div>

  <p class="mt-8 text-center text-sm text-zinc-500">
    Not a member?
    <Link to="/signup" class="font-semibold leading-6 text-indigo-600 hover:text-indigo-500">
      Sign up now
    </Link>
  </p>
</div>
