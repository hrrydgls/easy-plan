<script lang="ts">
	import type { EventType } from '$lib/api';
	import { fmt, isoDay, isoWeek, weekdayNames } from '$lib/dates';

	let {
		year,
		month,
		days,
		types,
		direction = 0,
		onpick,
		onswipe
	}: {
		year: number;
		month: number;
		days: Record<string, number[]>;
		types: EventType[];
		direction?: number;
		onpick: (day: string) => void;
		onswipe: (step: number) => void;
	} = $props();

	const names = weekdayNames();
	const byId = $derived(new Map(types.map((t) => [t.id, t])));
	const today = new Date();

	type Cell = { n: number; key: string; weekend: boolean; today: boolean } | null;
	const weeks = $derived.by(() => {
		const first = new Date(year, month, 1);
		const lead = (first.getDay() + 6) % 7;
		const count = new Date(year, month + 1, 0).getDate();
		const rows: { week: number; cells: Cell[] }[] = [];
		for (let w = 0; w < Math.ceil((lead + count) / 7); w++) {
			const cells: Cell[] = [];
			for (let i = 0; i < 7; i++) {
				const n = w * 7 + i - lead + 1;
				cells.push(
					n < 1 || n > count
						? null
						: {
								n,
								key: isoDay(year, month, n),
								weekend: i > 4,
								today: today.getFullYear() === year && today.getMonth() === month && today.getDate() === n
							}
				);
			}
			rows.push({ week: isoWeek(new Date(year, month, 1 - lead + w * 7)), cells });
		}
		return rows;
	});

	// Swipe left/right (touch or mouse drag) to change month.
	let sx: number | null = null;
	let sy = 0;
	let swiped = false;
	function down(e: PointerEvent) {
		sx = e.clientX;
		sy = e.clientY;
		swiped = false;
	}
	function up(e: PointerEvent) {
		if (sx === null) return;
		const dx = e.clientX - sx;
		const dy = e.clientY - sy;
		sx = null;
		if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy) * 1.5) {
			swiped = true;
			setTimeout(() => (swiped = false), 50);
			onswipe(dx < 0 ? 1 : -1);
		}
	}
	const tagsOf = (key: string) => (days[key] ?? []).map((id) => byId.get(id)).filter((t): t is EventType => !!t);
</script>

<section
	class="card overflow-hidden select-none [touch-action:pan-y]"
	aria-label="Month view"
	onpointerdown={down}
	onpointerup={up}
	onpointercancel={() => (sx = null)}
>
	{#key `${year}-${month}`}
		<div class={direction > 0 ? 'slide-next' : direction < 0 ? 'slide-prev' : ''}>
			<div class="px-5 pt-[18px] pb-3">
				<h2 class="m-0 font-display text-[clamp(1.8rem,5vw,2.8rem)] leading-none font-bold tracking-[-0.03em]">
					{fmt(new Date(year, month, 1), { month: 'long' })}<small class="ml-2 font-medium text-faint">{year}</small>
				</h2>
			</div>
			<div class="grid grid-cols-7 sm:grid-cols-[34px_repeat(7,minmax(0,1fr))]">
				<div class="hidden border-t border-line py-1.5 text-center font-mono text-[0.7rem] text-faint uppercase sm:block">Wk</div>
				{#each names as name, i}
					<div
						class="border-t border-line py-1.5 text-center text-[0.7rem] font-semibold tracking-[0.08em] uppercase sm:px-2 sm:text-left {i > 4
							? 'text-[color-mix(in_srgb,var(--accent)_70%,var(--muted))]'
							: 'text-muted'}"
					>
						{name}
					</div>
				{/each}
				{#each weeks as row}
					<div class="hidden border-t border-line bg-surface-2 pt-3 text-center font-mono text-[0.68rem] text-faint sm:block">
						{row.week}
					</div>
					{#each row.cells as cell}
						{#if !cell}
							<div class="min-h-16 border-t border-l border-line bg-surface-2 sm:min-h-[92px]" aria-hidden="true"></div>
						{:else}
							{@const tags = tagsOf(cell.key)}
							<button
								type="button"
								class="flex min-h-16 min-w-0 flex-col items-center gap-1 border-t border-l border-line p-1.5 text-left hover:bg-surface-2 sm:min-h-[92px] sm:items-start sm:p-2 {cell.weekend
									? 'bg-[color-mix(in_srgb,var(--accent)_2.5%,var(--surface))]'
									: 'bg-surface'}"
								aria-label={fmt(new Date(year, month, cell.n), { weekday: 'long', day: 'numeric', month: 'long' }) +
									(tags.length ? ': ' + tags.map((t) => t.name).join(', ') : '')}
								onclick={() => !swiped && onpick(cell.key)}
							>
								<span
									class="grid size-[30px] place-items-center rounded-full font-mono text-[0.85rem] tabular-nums {cell.today
										? 'bg-today font-bold text-today-ink'
										: cell.weekend
											? 'text-[color-mix(in_srgb,var(--accent)_70%,var(--muted))]'
											: ''}">{cell.n}</span
								>
								{#if tags.length}
									<span class="flex w-full min-w-0 flex-wrap justify-center gap-px sm:flex-col sm:gap-0.5">
										{#each tags as t (t.id)}
											<span class="truncate text-[0.74rem] leading-tight text-muted">
												<span class="text-[0.95rem] sm:mr-1">{t.emoji}</span><span class="max-sm:hidden">{t.name}</span>
											</span>
										{/each}
									</span>
								{/if}
							</button>
						{/if}
					{/each}
				{/each}
			</div>
		</div>
	{/key}
</section>
