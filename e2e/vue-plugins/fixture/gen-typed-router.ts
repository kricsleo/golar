import { createRoutesContext, resolveOptions } from 'vue-router/unplugin'

const options = resolveOptions({
	logs: true,
})

const ctx = createRoutesContext(options)
await ctx.scanPages(false)
