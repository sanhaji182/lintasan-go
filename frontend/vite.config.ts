import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		port: 5173,
		proxy: {
			'/api': 'http://127.0.0.1:20180',
			'/v1': 'http://127.0.0.1:20180',
			'/health': 'http://127.0.0.1:20180'
		}
	}
});
