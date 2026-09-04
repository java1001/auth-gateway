<script>
  import { Link, navigate } from "svelte-routing";
  import api from "../lib/api";
  import { Mail, Lock, Loader2, ArrowRight, UserPlus } from "@lucide/svelte";

  let email = "";
  let password = "";
  let isLoading = false;
  let errorMsg = "";

  const handleSignup = async (e) => {
    e.preventDefault();
    isLoading = true;
    errorMsg = "";

    try {
      await api.post("/signup", { email, password });
      // Navigate to OTP verification passing the email in state
      navigate("/verify-email", { state: { email } });
    } catch (err) {
      errorMsg = err.response?.data?.error || "Registration failed. Please try again.";
    } finally {
      isLoading = false;
    }
  };
</script>

<div class="rounded-3xl bg-white p-8 shadow-xl shadow-zinc-200/50 ring-1 ring-zinc-100">
  <div class="mb-8 text-center">
    <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-50 text-indigo-600">
      <UserPlus size={24} />
    </div>
    <h2 class="mt-4 text-2xl font-bold tracking-tight text-zinc-900">Create an account</h2>
    <p class="mt-2 text-sm text-zinc-500">Join us and start your journey</p>
  </div>

  {#if errorMsg}
    <div class="mb-6 rounded-lg bg-red-50 p-4 text-sm text-red-600 ring-1 ring-red-500/20">
      {errorMsg}
    </div>
  {/if}

  <form on:submit={handleSignup} class="space-y-5">
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
      <label for="password" class="block text-sm font-medium text-zinc-700">Password</label>
      <div class="relative mt-2">
        <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-zinc-400">
          <Lock size={18} />
        </div>
        <input
          id="password"
          type="password"
          bind:value={password}
          required
          minlength="8"
          maxlength="72"
          placeholder="Create a strong password"
          class="block w-full rounded-xl border-0 py-2.5 pl-10 text-zinc-900 shadow-sm ring-1 ring-inset ring-zinc-300 placeholder:text-zinc-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6 transition-all duration-200"
        />
      </div>
      <p class="mt-2 text-xs text-zinc-500">Must be at least 8 characters long.</p>
    </div>

    <button
      type="submit"
      disabled={isLoading}
      class="group mt-8 flex w-full justify-center rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:opacity-70 transition-all duration-200"
    >
      {#if isLoading}
        <Loader2 class="mr-2 animate-spin" size={20} />
        Creating account...
      {:else}
        Sign up
        <ArrowRight class="ml-2 opacity-0 -translate-x-2 group-hover:opacity-100 group-hover:translate-x-0 transition-all duration-200" size={20} />
      {/if}
    </button>
  </form>

  <p class="mt-8 text-center text-sm text-zinc-500">
    Already have an account?
    <Link to="/login" class="font-semibold leading-6 text-indigo-600 hover:text-indigo-500">
      Sign in instead
    </Link>
  </p>
</div>
