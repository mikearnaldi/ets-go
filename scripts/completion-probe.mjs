// Completion probe: drives tsgo --lsp --stdio against the playground and
// compares textDocument/completion for an in-memory .ts document vs an
// in-memory .ets document with identical content. Usage:
//   node scripts/completion-probe.mjs ["imp"] [line] [character]

import { spawn } from "node:child_process";
import path from "node:path";
import url from "node:url";

const root = path.dirname(path.dirname(url.fileURLToPath(import.meta.url)));
const tsgo = path.join(root, "typescript-go", "built", "local", "tsgo");
const projectDir = path.join(root, "playground");

const content = process.argv[2] ?? "imp\n";
const line = Number(process.argv[3] ?? 0);
const character = Number(process.argv[4] ?? content.indexOf("\n") === -1 ? content.length : content.indexOf("\n"));

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
    if (message.method === "client/registerCapability") {
      console.error(
        "registerCapability:",
        JSON.stringify(message.params?.registrations?.map((r) => ({ id: r.id, method: r.method }))),
      );
    }
    send({ jsonrpc: "2.0", id: message.id, result: null });
    return;
  }
}

async function main() {
  await call("initialize", {
    processId: process.pid,
    rootUri: url.pathToFileURL(projectDir).href,
    capabilities: {
      textDocument: {
        synchronization: { dynamicRegistration: true },
        completion: {
          dynamicRegistration: true,
          completionItem: { snippetSupport: true, labelDetailsSupport: true },
        },
      },
      workspace: {},
    },
    initializationOptions: { loadExternalPlugins: true },
    workspaceFolders: [{ uri: url.pathToFileURL(projectDir).href, name: "playground" }],
  });
  notify("initialized", {});

  const probe = async (fileName, languageId, { openText, changes = [], waitAfter = 3000 } = {}) => {
    const uri = url.pathToFileURL(path.join(projectDir, "src", fileName)).href;
    let version = 1;
    notify("textDocument/didOpen", {
      textDocument: { uri, languageId, version: version++, text: openText },
    });
    for (const change of changes) {
      notify("textDocument/didChange", {
        textDocument: { uri, version: version++ },
        contentChanges: [change],
      });
      await new Promise((r) => setTimeout(r, 300));
    }
    await new Promise((r) => setTimeout(r, waitAfter));
    const results = {};
    const positions = process.env.SWEEP
      ? [0, 1, 2, 3].map((c) => ({ line: 0, character: c }))
      : [{ line, character }, { line: 0, character: 0 }];
    for (const pos of positions) {
      try {
        const result = await call("textDocument/completion", {
          textDocument: { uri },
          position: pos,
          context: { triggerKind: 1 },
        });
        const items = Array.isArray(result) ? result : (result?.items ?? []);
        results[`completion@${pos.line}:${pos.character}`] = result === null ? "null" : `${items.length} items`;
        if (pos.line === line && pos.character === character && items.length > 0) {
          for (const item of items.slice(0, 5)) {
            console.log(`  ${item.label}  kind=${item.kind} detail=${item.detail ?? ""}`);
          }
        }
      } catch (err) {
        results[`completion@${pos.line}:${pos.character}`] = `error: ${err.message}`;
      }
      if (process.env.SWEEP) {
        try {
          const hover = await call("textDocument/hover", { textDocument: { uri }, position: pos });
          results[`hover@${pos.line}:${pos.character}`] = hover === null ? "null" : JSON.stringify(hover?.contents ?? hover).slice(0, 80);
        } catch (err) {
          results[`hover@${pos.line}:${pos.character}`] = `error: ${err.message}`;
        }
      }
    }
    try {
      const diag = await call("textDocument/diagnostic", { textDocument: { uri } });
      results.diagnostic = JSON.stringify(diag?.items?.map((d) => d.message) ?? diag);
    } catch (err) {
      results.diagnostic = `error: ${err.message}`;
    }
    console.error(`${fileName}:`, JSON.stringify(results, null, 1));
    notify("textDocument/didClose", { textDocument: { uri } });
  };

  const text = content.split("\n")[0];
  const incremental = [...text].map((ch, i) => ({
    range: { start: { line: 0, character: i }, end: { line: 0, character: i } },
    text: ch,
  }));

  const mode = process.argv[5] ?? "memory";
  if (mode === "memory") {
    await probe("probe-completion.ts", "typescript", { openText: content });
    await probe("probe-completion.ets", "ets", { openText: content });
  } else if (mode === "typing") {
    await probe("a.ts", "typescript", { openText: "", changes: incremental });
    await probe("a.ets", "ets", { openText: "", changes: incremental });
  } else if (mode === "diskfull") {
    await probe("a.ets", "ets", { openText: content });
  } else if (mode === "fullchange") {
    await probe("a.ets", "ets", { openText: "", changes: [{ text: content }] });
  }

  child.kill();
  process.exit(0);
}

main().catch((err) => {
  console.error(err);
  child.kill();
  process.exit(1);
});
