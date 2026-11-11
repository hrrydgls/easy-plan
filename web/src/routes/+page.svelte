<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { lastCalendar } from '$lib/state.svelte';

	// Open the last calendar used, else the first one, else the calendar list.
	$effect(() => {
		api.calendars()
			.then((cals) => {
				const last = lastCalendar();
				const pick = cals.find((c) => c.id === last) ?? cals[0];
				goto(pick ? `/calendar/${pick.id}` : '/calendars', { replaceState: true });
			})
			.catch(() => goto('/calendars', { replaceState: true }));
	});
</script>
