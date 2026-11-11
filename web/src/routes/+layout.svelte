<script lang="ts">
	import '../app.css';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { loadSession, session, toast } from '$lib/state.svelte';

	let { children } = $props();
	const PUBLIC = ['/login', '/register'];
	let failed = $state('');

	// Route guard: send logged-out people to /login and logged-in people away from it.
	$effect(() => {
		const path = page.url.pathname;
		loadSession()
			.then((user) => {
				failed = '';
				if (!user && !PUBLIC.includes(path)) goto('/login', { replaceState: true });
				else if (user && PUBLIC.includes(path)) goto('/', { replaceState: true });
			})
			.catch((e) => (failed = e.message));
	});
	const ready = $derived(session.checked && (session.user ? !PUBLIC.includes(page.url.pathname) : PUBLIC.includes(page.url.pathname)));
</script>

{#if failed}
	<div class="page"><p class="card p-5 text-muted">{failed}</p></div>
{:else if ready}
	{@render children()}
{/if}

{#if toast.text}
	{#key toast.id}
		<div
			class="fixed left-1/2 z-[60] -translate-x-1/2 rounded-[10px] bg-fg px-4 py-2 text-[0.88rem] text-bg"
			style="bottom: calc(20px + env(safe-area-inset-bottom, 0px))"
			role="status"
		>
			{toast.text}
		</div>
	{/key}
{/if}
