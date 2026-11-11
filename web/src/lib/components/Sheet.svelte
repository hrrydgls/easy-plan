<script lang="ts">
	import type { Snippet } from 'svelte';

	let { open = $bindable(false), title, subtitle = '', onclose, children, footer }: {
		open: boolean;
		title: string;
		subtitle?: string;
		onclose?: () => void;
		children: Snippet;
		footer?: Snippet;
	} = $props();

	let dialog = $state<HTMLDivElement>();
	const close = () => {
		open = false;
		onclose?.();
	};
	$effect(() => {
		if (open) queueMicrotask(() => dialog?.querySelector<HTMLElement>('input, button')?.focus());
	});
</script>

<svelte:window onkeydown={(e) => open && e.key === 'Escape' && close()} />

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 grid place-items-end justify-items-center bg-[rgba(10,14,22,.45)] sm:place-items-center"
		style="padding: 0 env(safe-area-inset-right, 0px) env(safe-area-inset-bottom, 0px) env(safe-area-inset-left, 0px)"
		onclick={(e) => e.target === e.currentTarget && close()}
	>
		<div
			bind:this={dialog}
			class="rise flex max-h-[90vh] w-full max-w-[520px] flex-col gap-3.5 overflow-auto rounded-t-[18px] bg-surface px-5 pt-5 pb-6 shadow-[0_18px_50px_-20px_rgba(0,0,0,.45)] sm:rounded-[18px]"
			role="dialog"
			aria-modal="true"
			aria-label={title}
		>
			<div class="flex flex-col gap-1">
				<h3 class="m-0 font-display text-[1.3rem] font-bold tracking-tight">{title}</h3>
				{#if subtitle}<p class="m-0 text-[0.88rem] text-muted">{subtitle}</p>{/if}
			</div>
			{@render children()}
			{#if footer}<div class="flex flex-wrap items-center justify-end gap-2">{@render footer()}</div>{/if}
		</div>
	</div>
{/if}
