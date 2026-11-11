<script lang="ts">
	import type { Snippet } from 'svelte';
	import { setTheme, theme } from '$lib/state.svelte';

	let { title, subtitle, children }: { title: string; subtitle: string; children: Snippet } = $props();
	const days = Array.from({ length: 35 }, (_, i) => i - 2);
	const marks: Record<number, string> = { 3: '💼', 4: '💼', 5: '💼', 10: '🏖️', 11: '🏖️', 12: '🏖️', 18: '🎉', 24: '💼', 25: '💼' };
</script>

<main class="page min-h-dvh justify-center">
	<div class="grid items-center gap-8 md:grid-cols-[1fr_400px]">
		<section class="hidden flex-col gap-5 md:flex" aria-hidden="true">
			<p class="m-0 font-display text-[3.4rem] leading-[0.95] font-bold tracking-tight">
				Every day,<br />tagged and <span class="text-accent">counted.</span>
			</p>
			<p class="m-0 max-w-[30rem] text-muted">
				Mark work days, holidays and anything else you track. Easy Plan adds them up per month and per year.
			</p>
			<div class="card grid max-w-[360px] grid-cols-7 gap-1 p-3">
				{#each days as d}
					<div
						class="grid aspect-square place-items-center rounded-md text-[0.72rem] font-mono {d < 1 || d > 30
							? 'opacity-0'
							: d === 18
								? 'bg-today text-today-ink font-bold'
								: 'bg-surface-2 text-muted'}"
					>
						{#if d >= 1 && d <= 30}{marks[d] ?? d}{/if}
					</div>
				{/each}
			</div>
		</section>

		<section class="card flex flex-col gap-5 p-6">
			<div class="flex items-center justify-between gap-3">
				<span class="font-display text-[1.4rem] font-bold tracking-tight">Easy<span class="text-accent">.</span>Plan</span>
				<button
					type="button"
					class="text-[0.85rem] text-muted"
					onclick={() => setTheme(theme.value === 'dark' ? 'light' : 'dark')}
					>{theme.value === 'dark' ? '☀ Light' : '☾ Dark'}</button
				>
			</div>
			<div>
				<h1 class="m-0 font-display text-[1.7rem] font-bold tracking-tight">{title}</h1>
				<p class="m-0 mt-1 text-muted">{subtitle}</p>
			</div>
			{@render children()}
		</section>
	</div>
</main>
