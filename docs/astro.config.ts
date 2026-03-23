import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
// import starlightBlog from 'starlight-blog'

export default defineConfig({
	integrations: [
		starlight({
			title: 'Golar',
      plugins: [
        // starlightBlog(),
      ],
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/auvred/golar' }],
			sidebar: [
				{
					label: 'About',
					items: [
						{ label: 'Getting started', slug: 'about/getting-started' },
					],
				},
			],
      customCss: ['./src/styles/custom.css']
		}),
	],
});
