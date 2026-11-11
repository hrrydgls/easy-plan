<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, ApiError, type Calendar, type CalendarDetail, type EventType } from '$lib/api';
	import EventsEditor from '$lib/components/EventsEditor.svelte';
	import Header from '$lib/components/Header.svelte';
	import MonthGrid from '$lib/components/MonthGrid.svelte';
	import Sheet from '$lib/components/Sheet.svelte';
	import Totals from '$lib/components/Totals.svelte';
	import { fmt } from '$lib/dates';
	import { rememberCalendar, showToast } from '$lib/state.svelte';

	const id = $derived(Number(page.params.id));

	let cal = $state<CalendarDetail | null>(null);
	let loadError = $state('');

	$effect(() => {
		const calId = id;
		cal = null;
		loadError = '';
		api.calendar(calId)
			.then((c) => {
				cal = c;
				rememberCalendar(c.id);
			})
			.catch((e) => {
				if (e instanceof ApiError && e.status === 404) goto('/calendars', { replaceState: true });
				else loadError = e.message;
			});
	});

	// ---------- Month navigation ----------
	const now = new Date();
	let year = $state(now.getFullYear());
	let month = $state(now.getMonth());
	let direction = $state(0);
	function go(step: number) {
		const d = new Date(year, month + step, 1);
		direction = step;
		year = d.getFullYear();
		month = d.getMonth();
	}
	function goToday() {
		const t = new Date();
		direction = 0;
		year = t.getFullYear();
		month = t.getMonth();
	}

	// ---------- Day tagging (optimistic, last change wins) ----------
	let openDay = $state<string | null>(null);
	let dayOpen = $state(false);
	const seqs = new Map<string, number>();

	function pick(day: string) {
		openDay = day;
		dayOpen = true;
	}
	async function setDay(day: string, ids: number[]) {
		if (!cal) return;
		const before = cal.days[day] ?? [];
		if (ids.length) cal.days[day] = ids;
		else delete cal.days[day];
		const seq = (seqs.get(day) ?? 0) + 1;
		seqs.set(day, seq);
		try {
			const res = await api.setDay(cal.id, day, ids);
			if (seqs.get(day) !== seq || !cal) return; // a newer change is in flight
			if (res.eventTypeIds.length) cal.days[day] = res.eventTypeIds;
			else delete cal.days[day];
		} catch (e) {
			if (seqs.get(day) !== seq || !cal) return;
			if (before.length) cal.days[day] = before;
			else delete cal.days[day];
			showToast(`Couldn't save that day. ${(e as Error).message}`);
		}
	}
	function toggle(t: EventType) {
		if (!openDay || !cal) return;
		const cur = cal.days[openDay] ?? [];
		setDay(openDay, cur.includes(t.id) ? cur.filter((x) => x !== t.id) : [...cur, t.id]);
	}
	const dayLabel = $derived.by(() => {
		if (!openDay) return '';
		const [y, m, d] = openDay.split('-').map(Number);
		return fmt(new Date(y, m - 1, d), { weekday: 'long', day: 'numeric', month: 'long' });
	});
	const dayIds = $derived(openDay && cal ? (cal.days[openDay] ?? []) : []);

	// ---------- Events editor ----------
	let eventsOpen = $state(false);
	async function saveTypes(types: EventType[]) {
		if (!cal) return;
		const removed = cal.eventTypes.filter((t) => !types.some((x) => x.id === t.id)).length;
		cal = await api.saveEventTypes(cal.id, types);
		showToast(removed ? `Events saved. Removed ${removed} and cleared ${removed === 1 ? 'it' : 'them'} from your days.` : 'Events saved');
	}

	function keydown(e: KeyboardEvent) {
		if (dayOpen || eventsOpen || (e.target as HTMLElement).matches('input, select, textarea')) return;
		if (e.key === 'ArrowLeft') go(-1);
		if (e.key === 'ArrowRight') go(1);
	}
</script>

<svelte:head><title>{cal ? `${cal.name} · Easy Plan` : 'Easy Plan'}</title></svelte:head>
<svelte:window onkeydown={keydown} />

<main class="page">
	<Header>
		<button type="button" class="btn" onclick={() => (eventsOpen = true)} disabled={!cal}>Events</button>
		<div class="flex gap-2">
			<button type="button" class="btn w-[38px] !px-0" aria-label="Previous month" onclick={() => go(-1)}>‹</button>
			<button type="button" class="btn" onclick={goToday}>Today</button>
			<button type="button" class="btn w-[38px] !px-0" aria-label="Next month" onclick={() => go(1)}>›</button>
		</div>
	</Header>

	{#if loadError}
		<p class="card m-0 p-5 text-danger">{loadError}</p>
	{:else if !cal}
		<div class="card h-[520px] animate-pulse bg-surface-2" aria-label="Loading calendar"></div>
	{:else}
		<MonthGrid {year} {month} {direction} days={cal.days} types={cal.eventTypes} onpick={pick} onswipe={go} />
		<Totals {year} {month} days={cal.days} types={cal.eventTypes} />
	{/if}
</main>

<Sheet bind:open={dayOpen} title={dayLabel} subtitle="Tap to add or remove. A day can have more than one." onclose={() => (openDay = null)}>
	<div class="flex flex-wrap gap-2">
		{#each cal?.eventTypes ?? [] as t (t.id)}
			<button type="button" class="pill" aria-pressed={dayIds.includes(t.id)} onclick={() => toggle(t)}>
				<span class="text-base leading-none">{t.emoji}</span>{t.name}
			</button>
		{:else}
			<p class="m-0 text-muted">This calendar has no events yet. Add some under Events.</p>
		{/each}
	</div>
	{#snippet footer()}
		{#if dayIds.length}
			<button type="button" class="btn btn-danger mr-auto" onclick={() => openDay && setDay(openDay, [])}>Clear day</button>
		{:else if !cal?.eventTypes.length}
			<button type="button" class="btn mr-auto" onclick={() => ((dayOpen = false), (eventsOpen = true))}>Open Events</button>
		{/if}
		<button type="button" class="btn btn-primary" onclick={() => ((dayOpen = false), (openDay = null))}>Done</button>
	{/snippet}
</Sheet>

{#if cal}
	<EventsEditor bind:open={eventsOpen} types={cal.eventTypes} onsave={saveTypes} />
{/if}
