# Golar

Golar is an embedded language framework built on [typescript-go](https://github.com/microsoft/typescript-go). Currently there [is no official way to do this](https://github.com/microsoft/typescript-go/issues/648) with typescript-go.

## Language support

Currently, Golar supports Astro, Svelte, and Vue by integrating their official language tooling:

- Astro: [withastro/compiler](https://github.com/withastro/compiler)
- Svelte: [sveltejs/language-tools](https://github.com/sveltejs/language-tools)
- Vue: [vuejs/language-tools](https://github.com/vuejs/language-tools)

## Usage

```bash
# for Astro
npm add golar @golar/astro
npx golar --noEmit

# for Svelte
npm add golar @golar/svelte
npx golar --noEmit

# for Vue
npm add golar @golar/vue
npx golar --noEmit
# emit .vue.d.ts files
npx golar --declaration --emitDeclarationOnly
```

## Plugin architecture

> [!NOTE]
> The plugin interface is in a very early stage. Expect breaking changes.

Golar uses plugins for language integration.

- Go SDK: supported
- TypeScript SDK: supported
- Rust SDK: coming soon

This design makes language support incremental and keeps the core focused on the typechecking pipeline.

## Why this approach

The project intentionally reuses official language tooling instead of reimplementing everything in Go.

The main reason is simple: code generation is not the bottleneck; typechecking is.
Rewriting all codegen in Go would:

- reduce compatibility with official tools
- create long-term maintenance burden
- provide little practical performance improvement

By staying close to official tooling, Golar keeps behavior aligned with ecosystem expectations while still providing a fast and clean integration path.

## Other approaches

There are a few related projects exploring this space:

- [astralhpi/svelte-fast-check](https://github.com/astralhpi/svelte-fast-check/)
- [pheuter/svelte-check-rs](https://github.com/pheuter/svelte-check-rs)
- [KazariEX/vue-tsgo](https://github.com/KazariEX/vue-tsgo)

A common pattern in these projects is to copy files into `/tmp`, rewrite custom extensions and imports to `.ts`, spawn `tsgo` as a subprocess to collect diagnostics, and then map locations back to the original files. This works, but it is hacky.

Another project, [ubugeeei/vize](https://github.com/ubugeeei/vize), takes an interesting approach by using `tsgo`'s LSP, which avoids the `/tmp` file strategy.

Golar goes further by patching `tsgo` so extension-based languages are treated as if they are supported natively. That means no rewriting import extensions or related paths. A `.vue` file is handled like a `.ts` file.

## Current scope

Right now, Golar supports CLI workflows only:

- typechecking (`--noEmit`)
- type emitting (`--declaration --emitDeclarationOnly`)

## Volar.js compatibility layer

Golar includes a [Volar.js](https://github.com/volarjs/volar.js) compatibility layer.
This means languages built on Volar.js can be adapted to work with Golar.

This layer is currently unstable and not documented yet.
Documentation will be added in future releases.

## Future plans

- Linting (so please don't use Golar in your linters, at least for now; I want to explore this route myself!)
- LSP support: TypeScript-only features
- LSP support: Embedded languages (HTML, CSS, etc.)
- LSP support: Custom completions, hovers, etc.
- Explore support for Angular, MDX, and other languages

## License

[MIT](./LICENSE)
