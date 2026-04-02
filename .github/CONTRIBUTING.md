# Development

Golar is at a very early stage of development, and the overall architecture is not yet settled.
That's why I want to implement major architectural changes myself.

However, I'd be more than happy to accept enthusiastic third-party contributions with a smaller scope.
For simple changes and trivial bugs, you can just open a PR.
If you want to contribute an enhancement or fix a non-trivial bug, it's better to open an issue first.
This way, we can discuss things and decide what the right approach to the problem is.

Here is a brief guide on how to build Golar locally.

> [!NOTE]
> I haven't tried developing Golar on macOS or Windows, so some steps may be missing in this guide.
> If you encounter any inconsistencies or have suggestions, don't hesitate to open a [new issue](https://github.com/auvred/golar/issues/new).

## Required tools

- Node.js (recommended 24+)
- Go
- C compiler
- rustc & Cargo

If you're using Nix, you can enter the development shell with all required packages:

```bash
nix develop .
```

## Setting up the project

First, you need to clone the project and its third-party submodules:

```bash
git clone https://github.com/auvred/golar
cd golar

git submodule update --init --depth 1 thirdparty/astro-compiler thirdparty/typescript
git submodule update --init thirdparty/typescript-go
pnpm i
```

After that, you need to patch the third-party submodules:

```bash
node ./tools/patch-astro.ts
node ./tools/patch-tsgo.ts
```

If you're on Windows, you also need to download `node.lib`:

```bash
node ./tools/download-node-lib.ts
```

Then you need to build a lighter version of `typescript`, which is required for `@golar/vue`:

```bash
pnpm build:typescript-lite
```

## Building the project

Golar has Go, Rust, and TypeScript parts.

To build the Go part (it's required to run Golar locally and run tests), run:

```bash
pnpm build:golar
```

To build the Rust part (it's required to run e2e tests), run this from the repo root:

```bash
cargo build
```

And finally, to build the TypeScript part (the packages published on npm), run:

```bash
pnpm build:packages
```

Building the TypeScript part isn't required to run tests.

## Running tests

Before running tests, ensure that you've built the Go and Rust parts (`pnpm build:golar && cargo build`).
After every change to the Go or C code, you should run `pnpm build:golar` to avoid accidentally running tests against stale source code.

Golar uses Vitest. To run tests, execute:

```bash
pnpm vitest
```

To test the Go part, run:

```bash
go test ./internal/...
```

## Validating changes

To format code, you should run:

```bash
pnpm format
```

To check that the TypeScript code doesn't have any type errors (using Golar), run:

```bash
pnpm check
```
