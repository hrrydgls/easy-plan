import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
export default {
	preprocess: vitePreprocess(),
	compilerOptions: { runes: true },
	kit: {
		// Static SPA build: the Go server embeds this output and falls back
		// to index.html for client-side routes like /calendar/3.
		adapter: adapter({ pages: 'build', assets: 'build', fallback: 'index.html', strict: false })
	}
};
