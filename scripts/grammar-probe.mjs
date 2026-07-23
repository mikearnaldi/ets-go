// Grammar probe: tokenizes ETS sample code with the vscode-ets TextMate
// grammar and prints scopes per token. Usage:
//   node scripts/grammar-probe.mjs            (built-in samples)
//   node scripts/grammar-probe.mjs file.ets   (tokenize a file)

import fs from "node:fs";
import path from "node:path";
import url from "node:url";
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const oniguruma = require("vscode-oniguruma");
const vsctm = require("vscode-textmate");
const root = path.dirname(path.dirname(url.fileURLToPath(import.meta.url)));

const VSCODE_TS_GRAMMAR =
  "/Applications/Visual Studio Code.app/Contents/Resources/app/extensions/typescript-basics/syntaxes/TypeScript.tmLanguage.json";
const ETS_GRAMMAR = path.join(root, "packages", "vscode-ets", "syntaxes", "ets.tmLanguage.json");
const ETS_INJECTION = path.join(root, "packages", "vscode-ets", "syntaxes", "ets-trailing-block-call.injection.tmLanguage.json");

const wasmBin = fs.readFileSync(require.resolve("vscode-oniguruma/release/onig.wasm")).buffer;
await oniguruma.loadWASM(wasmBin);

const registry = new vsctm.Registry({
  onigLib: Promise.resolve({
    createOnigScanner: (sources) => new oniguruma.OnigScanner(sources),
    createOnigString: (s) => new oniguruma.OnigString(s),
  }),
  getInjections: (scopeName) =>
    scopeName === "source.ets" ? ["ets.trailing-block-call.injection"] : undefined,
  loadGrammar: async (scopeName) => {
    if (scopeName === "source.ts") {
      return vsctm.parseRawGrammar(fs.readFileSync(VSCODE_TS_GRAMMAR, "utf8"), VSCODE_TS_GRAMMAR);
    }
    if (scopeName === "source.ets") {
      return vsctm.parseRawGrammar(fs.readFileSync(ETS_GRAMMAR, "utf8"), ETS_GRAMMAR);
    }
    if (scopeName === "ets.trailing-block-call.injection") {
      return vsctm.parseRawGrammar(fs.readFileSync(ETS_INJECTION, "utf8"), ETS_INJECTION);
    }
    return null;
  },
});

const grammar = await registry.loadGrammar("source.ets");

const samples = process.argv[2]
  ? [fs.readFileSync(process.argv[2], "utf8")]
  : [
      "export const program = Effect.gen {\n  const user = yield* getUser(1);\n  return user.name;\n}\n",
      "class A extends B.C {\n  foo() { return 1 }\n}\n",
      "const obj = { a: 1, b: { c: 2 } };\nfunction f() {\n  const x = 1;\n  return x;\n}\n",
    ];

for (const sample of samples) {
  let ruleStack = vsctm.INITIAL;
  for (const line of sample.split("\n")) {
    if (line === "") continue;
    const { tokens, ruleStack: next } = grammar.tokenizeLine(line, ruleStack);
    ruleStack = next;
    const parts = tokens.map((t) => {
      const text = line.slice(t.startIndex, t.endIndex);
      const scope = t.scopes[t.scopes.length - 1];
      return `${JSON.stringify(text)}:${scope}`;
    });
    console.log(parts.join("  "));
  }
  console.log("---");
}
