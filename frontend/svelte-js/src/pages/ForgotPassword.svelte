<script>
  import { Link } from "svelte-routing";
  import api from "../lib/api";
  import { Mail, ArrowLeft, KeySquare, Loader2 } from "@lucide/svelte";

  let email = "";
  let code = "";
  let password = "";
  let step = 1;
  let isLoading = false;
  let errorMsg = "";
  let successMsg = "";

  const submitEmail = async (event) => {
    event.preventDefault(); isLoading = true; errorMsg = "";
    try { await api.post("/forgot-password", { email: email.trim() }); step = 2; }
    catch (error) { errorMsg = error.response?.data?.error || "Could not start password reset."; }
    finally { isLoading = false; }
  };

  const resetPassword = async (event) => {
    event.preventDefault(); isLoading = true; errorMsg = "";
    try {
      const response = await api.post("/reset-password", { email: email.trim(), code: code.trim(), password });
      successMsg = response.data.message || "Password reset successfully.";
    } catch (error) { errorMsg = error.response?.data?.error || "Could not reset password."; }
    finally { isLoading = false; }
  };
</script>

<div class="rounded-3xl bg-white p-8 shadow-xl shadow-zinc-200/50 ring-1 ring-zinc-100">
  <div class="mb-8 text-center">
    <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-50 text-indigo-600"><KeySquare size={24} /></div>
    <h2 class="mt-4 text-2xl font-bold tracking-tight text-zinc-900">Reset your password</h2>
    <p class="mt-2 text-sm text-zinc-500">{step === 1 ? "Enter your email and we'll send a reset code." : "Enter the code from your email and choose a new password."}</p>
  </div>
  {#if errorMsg}<div class="mb-6 rounded-lg bg-red-50 p-4 text-sm text-red-600">{errorMsg}</div>{/if}
  {#if successMsg}<div class="mb-6 rounded-lg bg-green-50 p-4 text-sm text-green-700">{successMsg} <Link to="/login" class="font-semibold">Return to login</Link></div>{/if}
  {#if !successMsg}
    {#if step === 1}
      <form on:submit={submitEmail} class="space-y-5">
        <label for="email" class="block text-sm font-medium text-zinc-700">Email address</label>
        <div class="relative"><Mail class="pointer-events-none absolute left-3 top-3 text-zinc-400" size={18} /><input id="email" type="email" bind:value={email} required class="block w-full rounded-xl border-0 py-2.5 pl-10 text-zinc-900 ring-1 ring-inset ring-zinc-300" /></div>
        <button type="submit" disabled={isLoading} class="flex w-full justify-center rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-70">{#if isLoading}<Loader2 class="mr-2 animate-spin" size={20} />Sending...{:else}Send reset code{/if}</button>
      </form>
    {:else}
      <form on:submit={resetPassword} class="space-y-5">
        <input type="text" bind:value={code} required inputmode="numeric" autocomplete="one-time-code" placeholder="Reset code" class="block w-full rounded-xl border-0 py-2.5 px-3 text-zinc-900 ring-1 ring-inset ring-zinc-300" />
        <input type="password" bind:value={password} required minlength="8" maxlength="72" placeholder="New password" class="block w-full rounded-xl border-0 py-2.5 px-3 text-zinc-900 ring-1 ring-inset ring-zinc-300" />
        <button type="submit" disabled={isLoading} class="flex w-full justify-center rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-70">{#if isLoading}<Loader2 class="mr-2 animate-spin" size={20} />Resetting...{:else}Reset password{/if}</button>
      </form>
    {/if}
  {/if}
  <p class="mt-8 text-center text-sm text-zinc-500"><Link to="/login" class="inline-flex items-center font-semibold text-indigo-600"><ArrowLeft size={16} class="mr-2" />Back to login</Link></p>
</div>
