// TS API probe: starts tsgo --lsp over the playground, obtains an API
// session pipe via custom/initializeAPISession, then asks
// getExpressionTypeAtPosition for the operand of `run` in hello.ets.
// This mirrors exactly what the vscode-ets extension's run-hover does.
// Usage: node scripts/tsapi-probe.mjs

import { spawn } from "node:child_process";
import net from "node:net";
import fs from "node:fs";
import path from "node:path";
import url from "node:url";
import { createMessageConnection, SocketMessageReader, SocketMessageWriter } from "vscode-jsonrpc/node.js";

const root = path.dirname(path.dirname(url.fileURLToPath(import.meta.url)));
const tsgo = path.join(root, "typescript-go", "built", "local", "tsgo");
const projectDir = path.join(root, "playground");

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
    if (message.id !== undefined && (message.result !== undefined || message.error !== undefined)) {
      const entry = pending.get(message.id);
      if (entry) {
        pending.delete(message.id);
        if (message.error) entry.reject(new Error(JSON.stringify(message.error)));
        else entry.resolve(message.result);
      }
    } else if (message.id !== undefined && message.method) {
      send({ jsonrpc: "2.0", id: message.id, result: null });
    }
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

async function main() {
  await call("initialize", {
    processId: process.pid,
    rootUri: url.pathToFileURL(projectDir).href,
    capabilities: {},
    initializationOptions: { loadExternalPlugins: true },
    workspaceFolders: [{ uri: url.pathToFileURL(projectDir).href, name: "playground" }],
  });
  notify("initialized", {});

  const filePath = path.join(projectDir, "src", "hello.ets");
  const text = fs.readFileSync(filePath, "utf8");
  const uri = url.pathToFileURL(filePath).href;
  notify("textDocument/didOpen", { textDocument: { uri, languageId: "ets", version: 1, text } });
  await new Promise((r) => setTimeout(r, 3000));

  // Handshake: LSP custom request -> pipe path (the extension uses the
  // typescript.native-preview.initializeAPIConnection command for this).
  const session = await call("custom/initializeAPISession", {});
  console.error("api session:", JSON.stringify(session));

  const socket = net.createConnection(session.pipe);
  await new Promise((resolve, reject) => {
    socket.once("connect", resolve);
    socket.once("error", reject);
  });
  const conn = createMessageConnection(new SocketMessageReader(socket), new SocketMessageWriter(socket));
  conn.listen();
  await conn.sendRequest("initialize", null);

  const snapshotResponse = await conn.sendRequest("updateSnapshot", {});
  const snapshot = snapshotResponse.snapshot;
  const project = await conn.sendRequest("getDefaultProjectForFile", { snapshot, file: { uri } });
  console.error("project:", project?.configFileName ?? "none");

  const check = async (label, marker, extra = 0) => {
    const offset = text.indexOf(marker) + extra;
    const type = await conn.sendRequest("getExpressionTypeAtPosition", {
      snapshot,
      project: project.id,
      file: { uri },
      position: offset,
    });
    if (!type) {
      console.log(`${label}: <no type>`);
      return;
    }
    const typeString = await conn.sendRequest("typeToString", {
      snapshot,
      project: project.id,
      type: type.id,
    });
    console.log(`${label}: ${typeString}`);
  };

  // operand of `run getUser(1)` (offset of `getUser`)
  await check("run operand       ", "run getUser(1)", 4);
  // operand of `yield* getUser(1)` in the plain-TS twin (offset of `getUser` after `yield* `)
  await check("yield* operand    ", "yield* getUser(1)", 7);
  // a plain call expression elsewhere: `getUser` definition site reference
  await check("Effect.succeed(..)", "Effect.succeed(new User", 0);
  // inside the mapped preamble shift: the `user.name` return in the gen block
  await check("return user.name  ", "return user.name;\n}", 7);

  await conn.sendRequest("release", { snapshot }).catch(() => {});
  conn.dispose();
  socket.destroy();
  child.kill();
  process.exit(0);
}

main().catch((err) => {
  console.error(err);
  child.kill();
  process.exit(1);
});
