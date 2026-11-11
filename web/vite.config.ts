import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		// `pnpm dev` forwards API calls to the Go server (`make dev-server`).
		proxy: { '/api': 'http://localhost:9092' }
	}
});
