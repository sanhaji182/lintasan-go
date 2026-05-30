import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		runes: ({ filename }) => (filename.split(/[/\\\\]/).includes('node_modules') ? undefined : true)
	},
	kit: {
		// Static SPA build embedded into the Go binary. fallback=index.html lets
		// the client-side router handle every /dashboard/* path (200.html is the
		// SPA-friendly fallback name some hosts expect; index.html works with our
		// Go static handler which serves index.html for unknown non-asset paths).
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: 'index.html',
			precompress: false,
			strict: false
		}),
		csrf: {
			checkOrigin: false
		}
	}
};

export default config;
