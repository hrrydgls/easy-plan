<script lang="ts">
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { session, setTheme, theme } from '$lib/state.svelte';

	let { children }: { children?: Snippet } = $props();
	let menu = $state(false);

	async function logout() {
		menu = false;
		try {
			await api.logout();
		} finally {
			session.user = null;
			goto('/login');
		}
	}
</script>

<svelte:window onclick={() => (menu = false)} />

<header class="flex flex-wrap items-center gap-x-4 gap-y-3">
	<a href="/calendars" class="mr-auto font-display text-[1.6rem] font-bold tracking-tight no-underline text-fg">
		Easy<span class="text-accent">.</span>Plan
	</a>
	{@render children?.()}
	{#if session.user}
		<div class="relative">
			<button
				type="button"
				class="btn !px-2.5"
				aria-haspopup="menu"
				aria-expanded={menu}
				onclick={(e) => {
					e.stopPropagation();
					menu = !menu;
				}}
			>
				<span
					class="grid size-6 place-items-center rounded-full bg-accent text-[0.72rem] font-bold text-accent-ink"
					aria-hidden="true">{session.user.name.slice(0, 1).toUpperCase()}</span
				>
				<span class="max-w-[9rem] truncate max-sm:hidden">{session.user.name}</span>
			</button>
			{#if menu}
				<div
					class="absolute right-0 z-40 mt-2 w-56 rounded-xl border border-line bg-surface p-1.5 shadow-[0_18px_50px_-20px_rgba(0,0,0,.4)]"
					role="menu"
				>
					<div class="truncate px-2.5 py-2 text-[0.82rem] text-muted">{session.user.email}</div>
					<a href="/calendars" class="block rounded-lg px-2.5 py-2 text-fg no-underline hover:bg-surface-2" role="menuitem"
						>All calendars</a
					>
					<div class="px-2.5 pb-1 pt-2 text-[0.72rem] font-semibold uppercase tracking-wide text-muted">Theme</div>
					<div class="mx-1 mb-1.5 flex rounded-[10px] border border-line bg-bg p-[3px]" role="group" aria-label="Theme">
						{#each [['light', '☀ Light'], ['dark', '☾ Dark']] as [t, label]}
							<button
								type="button"
								class="flex-1 rounded-[7px] px-2.5 py-1 text-[0.88rem] font-medium text-muted aria-pressed:bg-fg aria-pressed:text-bg"
								aria-pressed={theme.value === t}
								onclick={(e) => {
									e.stopPropagation();
									setTheme(t as 'light' | 'dark');
								}}>{label}</button
							>
						{/each}
					</div>
					<button type="button" class="block w-full rounded-lg px-2.5 py-2 text-left hover:bg-surface-2" role="menuitem" onclick={logout}
						>Log out</button
					>
				</div>
			{/if}
		</div>
	{/if}
</header>
