// LSP probe: drives tsgo --lsp --stdio against the playground and dumps the
// semanticTokens/full response for a given file. Usage:
//   node scripts/lsp-probe.mjs [file]
// Defaults to playground/src/hello.ets.

import { spawn } from "node:child_process";
import path from "node:path";
import fs from "node:fs";
import url from "node:url";

const root = path.dirname(path.dirname(url.fileURLToPath(import.meta.url)));
const tsgo = path.join(root, "typescript-go", "built", "local", "tsgo");
const projectDir = path.join(root, "playground");
const targetFile = process.argv[2]
  ? path.resolve(process.argv[2])
  : path.join(projectDir, "src", "hello.ets");

const child = spawn(tsgo, ["--lsp", "--stdio"], { stdio: ["pipe", "pipe", "inherit"] });

let buffer = Buffer.alloc(0);
const pending = new Map();
let nextId = 1;

child.stdout.on("data", (chunk) => {
  buffer = buffer.length === 0 ? chunk : Buffer.concat([buffer, chunk]);
  for (;;) {
    const headerEnd = buffer.indexOf("\r\n\r\n");
    if (headerEnd === -1) break;
    const match = /Content-Length: *(\d+)/i.exec(buffer.toString("ascii", 0, headerEnd));
    const length = Number(match[1]);
    const start = headerEnd + 4;
    if (buffer.length < start + length) break;
    const message = JSON.parse(buffer.toString("utf8", start, start + length));
    buffer = buffer.subarray(start + length);
    handleMessage(message);
  }
});

function send(message) {
  const data = Buffer.from(JSON.stringify(message), "utf8");
  child.stdin.write(`Content-Length: ${data.length}\r\n\r\n`);
  child.stdin.write(data);
}

function call(method, params) {
  const id = nextId++;
  return new Promise((resolve, reject) => {
    pending.set(id, { resolve, reject });
    send({ jsonrpc: "2.0", id, method, params });
  });
}

function notify(method, params) {
  send({ jsonrpc: "2.0", method, params });
}

const serverRequests = [];
function handleMessage(message) {
  if (message.id !== undefined && (message.result !== undefined || message.error !== undefined)) {
    const entry = pending.get(message.id);
    if (entry) {
      pending.delete(message.id);
      if (message.error) entry.reject(new Error(JSON.stringify(message.error)));
      else entry.resolve(message.result);
    }
    return;
  }
  if (message.id !== undefined && message.method) {
    // Server-initiated request: log and reply null.
    if (message.method === "client/registerCapability") {
      console.error("registerCapability:", JSON.stringify(message.params?.registrations?.map((r) => ({ id: r.id, method: r.method }))));
    } else {
      serverRequests.push(message.method);
    }
    send({ jsonrpc: "2.0", id: message.id, result: null });
    return;
  }
  // Notification; ignore.
}

const TOKEN_TYPES = [
  "namespace", "type", "class", "enum", "interface", "struct", "typeParameter",
  "parameter", "variable", "property", "enumMember", "event", "function",
  "method", "macro", "keyword", "modifier", "comment", "string", "number",
  "regexp", "operator", "decorator",
];
const TOKEN_MODIFIERS = [
  "declaration", "definition", "readonly", "static", "deprecated", "abstract",
  "async", "modification", "documentation", "defaultLibrary",
];

async function main() {
  const init = await call("initialize", {
    processId: process.pid,
    rootUri: url.pathToFileURL(projectDir).href,
    capabilities: {
      textDocument: {
        synchronization: { dynamicRegistration: true },
        hover: { dynamicRegistration: true },
        definition: { dynamicRegistration: true },
        completion: { dynamicRegistration: true },
        diagnostic: { dynamicRegistration: true },
        semanticTokens: {
          dynamicRegistration: true,
          requests: { full: true, range: false },
          tokenTypes: TOKEN_TYPES,
          tokenModifiers: TOKEN_MODIFIERS,
          formats: ["relative"],
          overlappingTokenSupport: false,
          multilineTokenSupport: false,
        },
      },
      workspace: {},
    },
    initializationOptions: { loadExternalPlugins: true },
    workspaceFolders: [{ uri: url.pathToFileURL(projectDir).href, name: "playground" }],
  });
  notify("initialized", {});

  const legend = init.capabilities.semanticTokensProvider?.legend;
  console.error("server:", JSON.stringify(init.serverInfo ?? {}));
  console.error("legend:", JSON.stringify(legend));

  const content = fs.readFileSync(targetFile, "utf8");
  const uri = url.pathToFileURL(targetFile).href;
  notify("textDocument/didOpen", {
    textDocument: { uri, languageId: "ets", version: 1, text: content },
  });

  // Give the project time to load (config discovery, mapper spawn, transform).
  await new Promise((r) => setTimeout(r, 3000));

  const decode = (data, label) => {
    console.error(`${label}: ${data.length / 5} tokens`);
    const lines = content.split("\n");
    let line = 0, char = 0;
    for (let i = 0; i + 4 < data.length; i += 5) {
      const dl = data[i], dc = data[i + 1], len = data[i + 2], type = data[i + 3], mods = data[i + 4];
      line += dl;
      char = dl === 0 ? char + dc : dc;
      const text = (lines[line] ?? "").slice(char, char + len);
      const modNames = TOKEN_MODIFIERS.filter((_, b) => mods & (1 << b)).join(".");
      console.log(
        `${String(line + 1).padStart(3)}:${String(char).padStart(3)} ${TOKEN_TYPES[type] ?? type}${modNames ? "." + modNames : ""} ${JSON.stringify(text)}`,
      );
    }
  };

  const rangeResult = await call("textDocument/semanticTokens/range", {
    textDocument: { uri },
    range: { start: { line: 0, character: 0 }, end: { line: 14, character: 0 } },
  });
  decode(rangeResult?.data ?? [], "range[0:14]");

  const result = await call("textDocument/semanticTokens/full", {
    textDocument: { uri },
  });
  decode(result?.data ?? result?.semanticTokens?.data ?? [], "full");
  child.kill();
  process.exit(0);
}

main().catch((err) => {
  console.error(err);
  child.kill();
  process.exit(1);
});
