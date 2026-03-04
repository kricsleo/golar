# Golar

Golar is an embedded language framework based on [typescript-go](https://github.com/microsoft/typescript-go). It enables type checking for TypeScript-based languages like `.astro`, `.svelte`, and `.vue` via typescript-go. Its architecture is inspired by [@johnsoncodehk](https://github.com/johnsoncodehk)'s [Volar.js](https://github.com/volarjs/volar.js).

Currently, there [are no movements towards official support of extension languages in typescript-go](https://github.com/microsoft/typescript-go/issues/648).

## Language support

Currently, Golar supports Astro, Ember, Svelte, and Vue by integrating their official language tooling:

- Astro: [withastro/compiler](https://github.com/withastro/compiler)
- Ember: [@glint/ember-tsc](https://github.com/typed-ember/glint/tree/main/packages/core)
- Svelte: [svelte2tsx](https://github.com/sveltejs/language-tools/tree/master/packages/svelte2tsx)
- Vue: [@vue/language-core](https://github.com/vuejs/language-tools/tree/master/packages/language-core)

## Usage

```bash
# for Astro
npm add -D golar @golar/astro
npx golar --noEmit

# for Ember
npm add -D golar @golar/ember
npx golar --noEmit
# emit .d.ts files
npx golar --declaration --emitDeclarationOnly

# for Svelte
npm add -D golar @golar/svelte
npx golar --noEmit

# for Vue
npm add -D golar @golar/vue
npx golar --noEmit
# for Nuxt projects whose root tsconfig.json contains a "references" field
npx golar --build --noEmit
# emit .vue.d.ts files
npx golar --declaration --emitDeclarationOnly
```

## Plugin architecture

> [!NOTE]
> The plugin interface is in a very early stage. Expect breaking changes.

Golar can be extended with plugins. Communication with plugins is performed via STDIO, so plugins can be written in any language.

- Go SDK: supported
- TypeScript SDK: supported
- Rust SDK: coming soon

Currently, plugins support only virtual code generation, but in the future they will be able to enhance the LSP experience and perform linting.

## Why use the official JS-based tools instead of rewriting them in Go

The main reason is simple: code generation is not the bottleneck; typechecking is.

Rewriting all codegen in Go would:

- reduce compatibility with official tools
- create long-term maintenance burden
- provide little practical performance improvement

However, if Svelte and Vue one day provide official compiler/codegen infra written in Go/Rust, I'd be more than happy to use it in a Golar plugin for an even greater speedup.

## Other approaches

There are a few related projects exploring this space:

- [astralhpi/svelte-fast-check](https://github.com/astralhpi/svelte-fast-check/)
- [pheuter/svelte-check-rs](https://github.com/pheuter/svelte-check-rs)
- [KazariEX/vue-tsgo](https://github.com/KazariEX/vue-tsgo)

A common pattern in these projects is to copy files into `/tmp`, rewrite custom extensions and their imports to `.ts`, spawn `tsgo` as a subprocess to collect diagnostics, and then map locations back to the original files. This works, but it is hacky.

Another project, [ubugeeei/vize](https://github.com/ubugeeei/vize), takes an interesting approach by using `tsgo`'s LSP, which avoids the `/tmp` file strategy.

Golar goes further by patching `tsgo` so extension-based languages are treated as if they are supported natively. That means no rewriting import extensions or related paths. A `.vue` file is handled like a `.ts` file.

## Current scope

Right now, Golar supports CLI workflows only:

- typechecking (`--noEmit`)
- type emitting (`--declaration --emitDeclarationOnly`)

## Volar.js compatibility layer

Golar includes a [Volar.js](https://github.com/volarjs/volar.js) compatibility layer.
This means language plugins built on Volar.js can be adapted to work with Golar.

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
