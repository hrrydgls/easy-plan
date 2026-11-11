<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import AuthCard from '$lib/components/AuthCard.svelte';
	import { session } from '$lib/state.svelte';

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);
	let registration = $state(true);
	$effect(() => {
		api.config().then((c) => (registration = c.registration)).catch(() => {});
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		if (password.length < 8) {
			error = 'Use a password of at least 8 characters.';
			return;
		}
		busy = true;
		try {
			session.user = await api.register(name, email, password);
			goto('/');
		} catch (err) {
			error = (err as Error).message;
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Create account · Easy Plan</title></svelte:head>

<AuthCard title="Create your account" subtitle="You'll start with one calendar. Add more any time.">
	{#if !registration}
		<p class="m-0 text-muted">Registration is turned off on this server. Ask the person who runs it for an account.</p>
	{:else}
		<form class="flex flex-col gap-4" onsubmit={submit} novalidate>
			<label class="flex flex-col gap-1.5"
				><span class="label">Name</span>
				<input id="name" class="input" autocomplete="name" maxlength="80" required bind:value={name} /></label
			>
			<label class="flex flex-col gap-1.5"
				><span class="label">Email</span>
				<input id="email" class="input" type="email" autocomplete="email" required bind:value={email} /></label
			>
			<label class="flex flex-col gap-1.5"
				><span class="label">Password</span>
				<input id="password" class="input" type="password" autocomplete="new-password" minlength="8" required bind:value={password} />
				<span class="text-[0.8rem] text-faint">At least 8 characters.</span></label
			>
			{#if error}<p class="m-0 text-[0.88rem] text-danger" role="alert">{error}</p>{/if}
			<button class="btn btn-primary py-2.5" disabled={busy}>{busy ? 'Creating account…' : 'Create account'}</button>
		</form>
	{/if}
	<p class="m-0 text-[0.9rem] text-muted">Already have an account? <a href="/login" class="font-semibold text-accent">Log in</a></p>
</AuthCard>
