<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import AuthCard from '$lib/components/AuthCard.svelte';
	import { session } from '$lib/state.svelte';

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
		busy = true;
		try {
			session.user = await api.login(email, password);
			goto('/');
		} catch (err) {
			error = (err as Error).message;
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Log in · Easy Plan</title></svelte:head>

<AuthCard title="Welcome back" subtitle="Log in to see your calendars.">
	<form class="flex flex-col gap-4" onsubmit={submit} novalidate>
		<label class="flex flex-col gap-1.5"
			><span class="label">Email</span>
			<input id="email" class="input" type="email" autocomplete="email" required bind:value={email} /></label
		>
		<label class="flex flex-col gap-1.5"
			><span class="label">Password</span>
			<input id="password" class="input" type="password" autocomplete="current-password" required bind:value={password} /></label
		>
		{#if error}<p class="m-0 text-[0.88rem] text-danger" role="alert">{error}</p>{/if}
		<button class="btn btn-primary py-2.5" disabled={busy}>{busy ? 'Logging in…' : 'Log in'}</button>
	</form>
	{#if registration}
		<p class="m-0 text-[0.9rem] text-muted">New here? <a href="/register" class="font-semibold text-accent">Create an account</a></p>
	{/if}
</AuthCard>
