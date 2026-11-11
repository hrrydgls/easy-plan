<script lang="ts">
	import type { CountMode, EventType } from '$lib/api';
	import Sheet from './Sheet.svelte';

	let {
		open = $bindable(false),
		types,
		onsave
	}: { open: boolean; types: EventType[]; onsave: (types: EventType[]) => Promise<void> } = $props();

	type Draft = EventType & { key: string };
	let draft = $state<Draft[]>([]);
	let error = $state('');
	let busy = $state(false);
	let seq = 0;

	$effect(() => {
		if (open) {
			draft = types.map((t) => ({ ...t, key: `k${t.id}` }));
			error = '';
		}
	});

	function add() {
		const key = `new${++seq}`;
		draft.push({ id: 0, key, name: '', emoji: '⭐', count: 'month' });
		queueMicrotask(() => document.getElementById(`name-${key}`)?.focus());
	}

	async function save() {
		const clean = draft.map(({ key, ...t }) => ({ ...t, name: t.name.trim(), emoji: t.emoji.trim() || '•' }));
		if (clean.some((t) => !t.name)) {
			error = 'Every event needs a name.';
			return;
		}
		busy = true;
		error = '';
		try {
			await onsave(clean);
			open = false;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	const modes: [CountMode, string][] = [
		['none', "Don't count"],
		['month', 'Monthly'],
		['year', 'Yearly']
	];
</script>

<Sheet bind:open title="Events" subtitle="Name each kind of day, give it an emoji, and choose whether its days are counted per month, per year, or not at all.">
	<div class="grid grid-cols-[52px_minmax(0,1fr)_auto_32px] gap-2 max-[440px]:hidden">
		<span class="label">Emoji</span><span class="label">Name</span><span class="label">Count</span><span></span>
	</div>
	<div class="flex flex-col gap-2.5">
		{#each draft as t, i (t.key)}
			<div class="grid grid-cols-[52px_minmax(0,1fr)_auto_32px] items-center gap-2 max-[440px]:grid-cols-[48px_minmax(0,1fr)_32px]">
				<input id="emoji-{t.key}" class="input !px-1 !py-1.5 text-center text-[1.15rem]" maxlength="8" aria-label="Emoji" bind:value={t.emoji} />
				<input id="name-{t.key}" class="input" maxlength="40" placeholder="Event name" aria-label="Name" bind:value={t.name} />
				<select
					id="count-{t.key}"
					class="input !w-auto !py-2 text-[0.86rem] max-[440px]:col-start-2 max-[440px]:row-start-2"
					aria-label="Count"
					bind:value={t.count}
				>
					{#each modes as [v, label]}<option value={v}>{label}</option>{/each}
				</select>
				<button
					type="button"
					class="grid size-8 place-items-center rounded-lg text-[1.2rem] text-faint hover:bg-surface-2 hover:text-danger max-[440px]:col-start-3 max-[440px]:row-start-1"
					aria-label="Delete {t.name || 'event'}"
					onclick={() => draft.splice(i, 1)}>×</button
				>
			</div>
		{/each}
		{#if !draft.length}<p class="m-0 text-[0.9rem] text-muted">No events yet. Add the first one below.</p>{/if}
	</div>
	<div><button type="button" class="btn" onclick={add}>+ Add event</button></div>
	<p class="m-0 text-[0.8rem] text-faint">Deleting an event also removes it from every day it was tagged on.</p>
	{#if error}<p class="m-0 text-[0.88rem] text-danger" role="alert">{error}</p>{/if}
	{#snippet footer()}
		<button type="button" class="btn" onclick={() => (open = false)}>Cancel</button>
		<button type="button" class="btn btn-primary" disabled={busy} onclick={save}>{busy ? 'Saving…' : 'Save'}</button>
	{/snippet}
</Sheet>
