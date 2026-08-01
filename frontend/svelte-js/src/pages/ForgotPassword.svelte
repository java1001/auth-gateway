<script>
  import { Link } from "svelte-routing";
  import { Mail, ArrowLeft, KeySquare } from "@lucide/svelte";

  let email = "";
  let isSubmitted = false;

  const handleSubmit = (e) => {
    e.preventDefault();
    // In a real implementation, we would call api.post('/forgot-password', { email }) here.
    // For now, just simulate success to show the UI state.
    isSubmitted = true;
  };
</script>

<div class="rounded-3xl bg-white p-8 shadow-xl shadow-zinc-200/50 ring-1 ring-zinc-100">
  <div class="mb-8 text-center">
    <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-50 text-indigo-600">
      <KeySquare size={24} />
    </div>
    <h2 class="mt-4 text-2xl font-bold tracking-tight text-zinc-900">Reset your password</h2>
    <p class="mt-2 text-sm text-zinc-500">
      {#if !isSubmitted}
        Enter your email and we'll send you instructions to reset your password.
      {:else}
        Check your email for reset instructions.
      {/if}
    </p>
  </div>

  {#if !isSubmitted}
    <form on:submit={handleSubmit} class="space-y-5">
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

      <button
        type="submit"
        class="flex w-full justify-center rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 transition-all duration-200"
      >
        Send reset link
      </button>
    </form>
  {:else}
    <div class="rounded-lg bg-indigo-50 p-6 text-center text-sm text-indigo-800 ring-1 ring-indigo-500/20">
      We've sent a password reset link to <strong>{email}</strong>. Please check your inbox and spam folder.
    </div>
  {/if}

  <p class="mt-8 text-center text-sm text-zinc-500">
    <Link to="/login" class="inline-flex items-center font-semibold leading-6 text-indigo-600 hover:text-indigo-500">
      <ArrowLeft size={16} class="mr-2" />
      Back to login
    </Link>
  </p>
</div>
