<script lang="ts">
	import type { EventType } from '$lib/api';
	import { fmt, pad } from '$lib/dates';

	let { year, month, days, types }: { year: number; month: number; days: Record<string, number[]>; types: EventType[] } =
		$props();

	function tally(prefix: string) {
		const c = new Map<number, number>();
		for (const [k, ids] of Object.entries(days)) if (k.startsWith(prefix)) for (const id of ids) c.set(id, (c.get(id) ?? 0) + 1);
		return c;
	}
	const groups = $derived([
		{
			label: `${fmt(new Date(year, month, 1), { month: 'long' })} ${year}`,
			types: types.filter((t) => t.count === 'month'),
			counts: tally(`${year}-${pad(month + 1)}-`)
		},
		{ label: `Year ${year}`, types: types.filter((t) => t.count === 'year'), counts: tally(`${year}-`) }
	].filter((g) => g.types.length));
</script>

<section class="card flex flex-col gap-4 px-5 pt-[18px] pb-5" aria-label="Totals">
	<h3 class="m-0 font-display text-[1.3rem] font-bold tracking-tight">Totals</h3>
	{#if !groups.length}
		<p class="m-0 text-[0.9rem] text-muted">
			No events are counted yet. Open Events and choose Monthly or Yearly for the ones you want totals for.
		</p>
	{/if}
	{#each groups as g}
		<div class="flex flex-col gap-2">
			<h4 class="label m-0">{g.label}</h4>
			<div class="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-2.5">
				{#each g.types as t (t.id)}
					{@const n = g.counts.get(t.id) ?? 0}
					<div class="flex min-w-0 flex-col gap-0.5 rounded-xl bg-surface-2 px-3.5 py-3">
						<div class="flex min-w-0 items-center gap-2 text-[0.86rem] text-muted">
							<span class="text-[1.15rem]">{t.emoji}</span><span class="truncate">{t.name}</span>
						</div>
						<b class="font-display text-[2rem] leading-[1.05] tabular-nums">{n}</b>
						<small class="text-[0.74rem] text-faint">{n === 1 ? 'day' : 'days'}</small>
					</div>
				{/each}
			</div>
		</div>
	{/each}
</section>
