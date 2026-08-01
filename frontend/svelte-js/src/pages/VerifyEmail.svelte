<script>
  import { onMount } from "svelte";
  import { navigate, Link } from "svelte-routing";
  import api from "../lib/api";
  import { MailCheck, Loader2, KeyRound } from "@lucide/svelte";

  let email = "";
  let code = "";
  let isLoading = false;
  let errorMsg = "";
  let successMsg = "";

  onMount(() => {
    // Svelte-routing doesn't natively expose `state` in the component directly 
    // unless passed as a prop, but we can grab it from history if it was passed via navigate.
    if (window.history.state && window.history.state.email) {
      email = window.history.state.email;
    }
  });

  const handleVerify = async (e) => {
    e.preventDefault();
    isLoading = true;
    errorMsg = "";
    successMsg = "";

    try {
      const response = await api.post("/verify-email", { email, code });
      successMsg = response.data.message || "Email verified successfully!";
      // Delay slightly before redirecting to login so they see the success message
      setTimeout(() => {
        navigate("/login");
      }, 2000);
    } catch (err) {
      errorMsg = err.response?.data?.error || "Verification failed. Please check the code and try again.";
    } finally {
      isLoading = false;
    }
  };
</script>

<div class="rounded-3xl bg-white p-8 shadow-xl shadow-zinc-200/50 ring-1 ring-zinc-100">
  <div class="mb-8 text-center">
    <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-50 text-indigo-600">
      <MailCheck size={24} />
    </div>
    <h2 class="mt-4 text-2xl font-bold tracking-tight text-zinc-900">Check your email</h2>
    <p class="mt-2 text-sm text-zinc-500">We sent a verification code to {email || "your email"}</p>
  </div>

  {#if errorMsg}
    <div class="mb-6 rounded-lg bg-red-50 p-4 text-sm text-red-600 ring-1 ring-red-500/20">
      {errorMsg}
    </div>
  {/if}

  {#if successMsg}
    <div class="mb-6 rounded-lg bg-green-50 p-4 text-sm text-green-700 ring-1 ring-green-600/20">
      {successMsg}
      <p class="mt-1 text-xs">Redirecting to login...</p>
    </div>
  {/if}

  <form on:submit={handleVerify} class="space-y-5">
    <div>
      <label for="email" class="block text-sm font-medium text-zinc-700">Email address</label>
      <div class="mt-2">
        <input
          id="email"
          type="email"
          bind:value={email}
          required
          class="block w-full rounded-xl border-0 py-2.5 px-3 text-zinc-900 shadow-sm ring-1 ring-inset ring-zinc-300 placeholder:text-zinc-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6"
        />
      </div>
    </div>

    <div>
      <label for="code" class="block text-sm font-medium text-zinc-700">8-Digit Code</label>
      <div class="relative mt-2">
        <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-zinc-400">
          <KeyRound size={18} />
        </div>
        <input
          id="code"
          type="text"
          bind:value={code}
          required
          maxlength="10"
          placeholder="e.g. 12345678"
          class="block w-full rounded-xl border-0 py-2.5 pl-10 text-zinc-900 shadow-sm ring-1 ring-inset ring-zinc-300 placeholder:text-zinc-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6 font-mono tracking-widest"
        />
      </div>
    </div>

    <button
      type="submit"
      disabled={isLoading || successMsg !== ""}
      class="mt-8 flex w-full justify-center rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:opacity-70 transition-all duration-200"
    >
      {#if isLoading}
        <Loader2 class="mr-2 animate-spin" size={20} />
        Verifying...
      {:else}
        Verify Email
      {/if}
    </button>
  </form>

  <p class="mt-8 text-center text-sm text-zinc-500">
    Back to <Link to="/login" class="font-semibold leading-6 text-indigo-600 hover:text-indigo-500">Login</Link>
  </p>
</div>
