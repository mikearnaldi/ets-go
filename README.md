# ets-go

Testbed for **ETS**, an Effect-flavored dialect of TypeScript, built on the
[content mappers](https://github.com/microsoft/typescript-go/pull/4712)
proposal for typescript-go (PR #4712).

ETS is a superset of TypeScript: `.ets` files compile to plain TypeScript by
an external content mapper process, and `tsgo` typechecks, navigates, and
highlights the result with positions mapped back into the original source.

Currently implemented syntax:

```ts
// ETS
const program = Effect.gen {
  const user = run getUser(1);
  return user.name;
}

// compiles to
const program = Effect.gen(function* () {
  const user = yield* getUser(1);
  return user.name;
})
```

Two constructs, which compose:

- **Trailing block call**: `<MemberExpr> { statements }` in expression
  position (opening brace on the same line) becomes
  `<expr>(function* () { statements })`.
- **`run <expr>`**: inside a trailing block body, `run x` becomes `yield* x`
  anywhere `yield*` may appear, and maps back to `run` in diagnostics.
  `run` stays an ordinary identifier when followed by `(`, `[`, a template,
  or a binary/postfix operator (`run + 1`, `run(x)`, `run.x` are untouched),
  and outside trailing blocks it is never special.

More constructs (Schema, Service, ...) will follow the same machinery.

## Repository layout

| Path | What it is |
| --- | --- |
| `mapper/` | Self-contained Go module (`ets`) implementing the content mapper: vendored typescript-go scanner/parser (+ ETS grammar extensions), AST, transform, and protocol server. No dependency on the typescript-go checkout. |
| `packages/ets-content-mapper/` | npm manifest exposing the mapper to tsgo (`tsContentMapper.exec` → the Go binary). |
| `packages/vscode-ets/` | VS Code extension: `.ets` language registration, TextMate grammar (+ injection for ETS constructs), and the `discoverContentMappers` first-wake handshake. |
| `playground/` | Test project using the mapper, with `effect@4.0.0-beta` and sample `.ets` files. |
| `typescript-go/` | Gitignored clone of `microsoft/typescript-go`, checked out to PR #4712 plus local patches (`patches/`). This is the *system under test*, nothing here is imported by the mapper. |
| `patches/` | Local patches applied to the typescript-go clone by `scripts/setup-tsgo.sh` (currently: semantic tokens dynamic registration for content-mapped files). |
| `scripts/` | Setup, check, and debug tooling (see below). |

## Prerequisites

- **Nix + direnv** (recommended): `direnv allow` enters a shell with Go and
  Node.js provided by `flake.nix`.
- Or manually: Go ≥ 1.26 and Node.js ≥ 22 on `PATH`.

## Setup

```sh
npm install            # link workspaces, install effect + dev tools
npm run setup:tsgo     # clone typescript-go, checkout PR #4712, apply patches,
                       # build tsgo and the native-preview VS Code extension
```

## Everyday commands

```sh
npm test               # Go tests for the mapper (protocol, transform, parser)
npm run check          # build the mapper binary, typecheck playground/ with tsgo
npm run code           # open VS Code on playground/ with both extensions loaded
```

`npm run code` launches an Extension Development Host with:

1. the PR build of the **TypeScript 7** extension
   (`typescript-go/_extension`), serving the locally built `tsgo`, and
2. the **vscode-ets** extension.

Trust the workspace when prompted (`--loadExternalPlugins` requires trust).
Open `playground/src/hello.ets` to try completion, hover, go-to-definition,
diagnostics, and semantic highlighting inside ETS syntax.

## How it works

1. `playground/tsconfig.json` registers the mapper:

   ```json
   "contentMappers": [{ "package": "ets-content-mapper", "extensions": [".ets"] }]
   ```

2. When tsgo loads the project, it spawns `packages/ets-content-mapper/bin/ets-mapper`
   and talks JSON-RPC (LSP-style framing) over stdio: `initialize`, `transform`.
3. `transform` parses the `.ets` file with the vendored, ETS-extended
   typescript-go parser, rewrites ETS constructs, and returns the TypeScript
   text plus a **span map** between generated and original content
   (verbatim/atom/alias segments; scaffolding is synthesized).
4. tsgo typechecks the transformed text; diagnostics, hover, completions,
   rename, semantic tokens, etc. are mapped back into the original `.ets`
   source via the span map.

## Debugging tools

```sh
# Dump transformed text + span mappings for a file:
packages/ets-content-mapper/bin/ets-mapper debug playground/src/hello.ets

# Drive the real LSP server (tsgo --lsp --stdio) and print semantic tokens,
# dynamic registrations, etc.:
node scripts/lsp-probe.mjs [file]

# Print TextMate scopes produced by the vscode-ets grammar:
node scripts/grammar-probe.mjs [file]
```

## Updating typescript-go

```sh
npm run setup:tsgo     # re-fetches PR #4712, re-applies patches/, rebuilds
```

Local patches to the clone live in `patches/*.patch` and are applied with
`git am`; the clone itself is never edited by hand.

## Notes & limitations

- Content-mapped files are not emitted to JS (per the PR design); `tsc` usage
  here is typechecking + language services. Declaration emit is supported
  (`App.d.ets.ts`).
- The mapper package version participates in tsgo's incremental cache keys.
  When iterating on the mapper with `--incremental`/`--build`, bump
  `packages/ets-content-mapper/package.json` version or use `--force`/`--clean`.
