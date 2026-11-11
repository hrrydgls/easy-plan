<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, type Calendar } from '$lib/api';
	import Header from '$lib/components/Header.svelte';
	import Sheet from '$lib/components/Sheet.svelte';
	import { fmt } from '$lib/dates';
	import { showToast } from '$lib/state.svelte';

	let cals = $state<Calendar[] | null>(null);
	let loadError = $state('');
	$effect(() => {
		api.calendars().then((c) => (cals = c)).catch((e) => (loadError = e.message));
	});

	// Create
	let createOpen = $state(false);
	let newName = $state('');
	let starter = $state(true);
	let createErr = $state('');
	let busy = $state(false);
	async function create() {
		createErr = '';
		if (!newName.trim()) {
			createErr = 'Give the calendar a name.';
			return;
		}
		busy = true;
		try {
			const c = await api.createCalendar(newName.trim(), starter);
			goto(`/calendar/${c.id}`);
		} catch (e) {
			createErr = (e as Error).message;
		} finally {
			busy = false;
		}
	}

	// Rename / delete
	let editing = $state<Calendar | null>(null);
	let editOpen = $state(false);
	let editName = $state('');
	let editErr = $state('');
	let confirmDelete = $state(false);
	function openEdit(c: Calendar) {
		editing = c;
		editName = c.name;
		editErr = '';
		confirmDelete = false;
		editOpen = true;
	}
	async function rename() {
		if (!editing) return;
		try {
			const c = await api.renameCalendar(editing.id, editName);
			cals = cals!.map((x) => (x.id === c.id ? c : x));
			editOpen = false;
			showToast('Calendar renamed');
		} catch (e) {
			editErr = (e as Error).message;
		}
	}
	async function remove() {
		if (!editing) return;
		try {
			await api.deleteCalendar(editing.id);
			cals = cals!.filter((x) => x.id !== editing!.id);
			editOpen = false;
			showToast('Calendar deleted');
		} catch (e) {
			editErr = (e as Error).message;
		}
	}
</script>

<svelte:head><title>Calendars · Easy Plan</title></svelte:head>

<main class="page">
	<Header />
	<section class="card flex flex-col gap-4 p-5">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="m-0 font-display text-[2rem] font-bold tracking-tight">Your calendars</h1>
			<button type="button" class="btn btn-primary" onclick={() => ((newName = ''), (createErr = ''), (starter = true), (createOpen = true))}
				>+ New calendar</button
			>
		</div>
		{#if loadError}
			<p class="m-0 text-danger">{loadError}</p>
		{:else if !cals}
			<p class="m-0 text-muted">Loading…</p>
		{:else if !cals.length}
			<p class="m-0 text-muted">No calendars yet. Create one to start tagging days.</p>
		{:else}
			<ul class="m-0 flex list-none flex-col gap-2 p-0">
				{#each cals as c (c.id)}
					<li class="flex items-center gap-2 rounded-xl bg-surface-2 p-1.5 pl-4">
						<a href="/calendar/{c.id}" class="flex min-w-0 flex-1 flex-col py-1.5 text-fg no-underline">
							<strong class="truncate font-semibold">{c.name}</strong>
							<span class="text-[0.8rem] text-muted">Created {fmt(new Date(c.createdAt * 1000), { day: 'numeric', month: 'short', year: 'numeric' })}</span>
						</a>
						<button type="button" class="btn !py-1.5 text-[0.85rem]" onclick={() => openEdit(c)}>Edit</button>
						<a href="/calendar/{c.id}" class="btn !py-1.5 text-[0.85rem] text-fg no-underline">Open</a>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</main>

<Sheet bind:open={createOpen} title="New calendar" subtitle="Each calendar has its own events, tagged days and totals.">
	<label class="flex flex-col gap-1.5"
		><span class="label">Name</span>
		<input id="new-cal-name" class="input" maxlength="60" placeholder="2027, Side project, Family…" bind:value={newName}
			onkeydown={(e) => e.key === 'Enter' && create()} /></label
	>
	<label class="flex items-start gap-2.5 text-[0.92rem]">
		<input id="new-cal-starter" type="checkbox" class="mt-1" bind:checked={starter} />
		<span>Start with example events<br /><span class="text-[0.82rem] text-muted">💼 Work, 🏖️ Holiday, 🎌 National holiday, 🛋️ Weekend, 🎉 New job. Leave unticked for a blank calendar.</span></span>
	</label>
	{#if createErr}<p class="m-0 text-[0.88rem] text-danger">{createErr}</p>{/if}
	{#snippet footer()}
		<button type="button" class="btn" onclick={() => (createOpen = false)}>Cancel</button>
		<button type="button" class="btn btn-primary" disabled={busy} onclick={create}>Create calendar</button>
	{/snippet}
</Sheet>

<Sheet bind:open={editOpen} title="Edit calendar">
	<label class="flex flex-col gap-1.5"
		><span class="label">Name</span>
		<input id="edit-cal-name" class="input" maxlength="60" bind:value={editName} onkeydown={(e) => e.key === 'Enter' && rename()} /></label
	>
	{#if confirmDelete}
		<div class="flex flex-wrap items-center gap-2 rounded-[10px] bg-[color-mix(in_srgb,var(--danger)_10%,transparent)] p-3 text-[0.9rem]">
			<span class="mr-auto">Delete “{editing?.name}” and all its tagged days?</span>
			<button type="button" class="btn" onclick={() => (confirmDelete = false)}>Keep it</button>
			<button type="button" class="btn btn-danger-solid" onclick={remove}>Delete</button>
		</div>
	{/if}
	{#if editErr}<p class="m-0 text-[0.88rem] text-danger">{editErr}</p>{/if}
	{#snippet footer()}
		<button type="button" class="btn btn-danger mr-auto" onclick={() => (confirmDelete = true)}>Delete</button>
		<button type="button" class="btn" onclick={() => (editOpen = false)}>Cancel</button>
		<button type="button" class="btn btn-primary" onclick={rename}>Save</button>
	{/snippet}
</Sheet>
