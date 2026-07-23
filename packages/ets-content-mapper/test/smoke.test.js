"use strict";

const { test } = require("node:test");
const assert = require("node:assert/strict");
const { spawn } = require("node:child_process");
const path = require("node:path");

const SERVER = path.join(__dirname, "..", "src", "server.js");

function frame(message) {
  const data = Buffer.from(JSON.stringify(message), "utf8");
  return Buffer.concat([Buffer.from(`Content-Length: ${data.length}\r\n\r\n`, "ascii"), data]);
}

function startServer() {
  const child = spawn(process.execPath, [SERVER], { stdio: ["pipe", "pipe", "inherit"] });
  let buffer = Buffer.alloc(0);
  const pending = new Map();
  const queue = [];
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
      queue.push(message);
      const resolver = pending.get(message.id);
      if (resolver) {
        pending.delete(message.id);
        resolver(message);
      }
    }
  });
  let nextId = 1;
  return {
    child,
    call(method, params) {
      const id = nextId++;
      return new Promise((resolve, reject) => {
        pending.set(id, (message) => {
          if (message.error) reject(new Error(message.error.message));
          else resolve(message.result);
        });
        child.stdin.write(frame({ jsonrpc: "2.0", id, method, params }));
      });
    },
  };
}

test("initialize handshake", async (t) => {
  const server = startServer();
  t.after(() => server.child.kill());
  const result = await server.call("initialize", {
    protocolVersion: 1,
    positionEncodings: ["utf-8", "utf-16"],
    locale: "en",
  });
  assert.equal(result.protocolVersion, 1);
  assert.equal(result.positionEncoding, "utf-16");
  assert.equal(result.diagnosticSource, "ets");
});

test("transform pass-through maps the whole file verbatim", async (t) => {
  const server = startServer();
  t.after(() => server.child.kill());
  const content = "export const answer: number = 42;\n";
  const result = await server.call("transform", {
    fileName: "/project/src/answer.ets",
    content,
    compilerOptions: {},
  });
  assert.equal(result.text, content);
  assert.equal(result.scriptKind, 3);
  assert.deepEqual(result.mappings, [[0, content.length, 0, content.length, 0]]);
});
