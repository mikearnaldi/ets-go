#!/usr/bin/env node
"use strict";

// Content mapper server for the ETS language.
//
// Speaks JSON-RPC 2.0 over stdio with LSP base-protocol framing
// (Content-Length headers), implementing the typescript-go content
// mapper protocol (microsoft/typescript-go PR #4712). The host sends
// all requests; this process only responds. Methods:
//
//   initialize -> { protocolVersion, positionEncoding, diagnosticSource }
//   transform  -> { text, scriptKind, mappings, diagnostics }
//
// stdout carries framed protocol messages only; logs go to stderr.

const { transform } = require("./transform");

const PROTOCOL_VERSION = 1;
const DIAGNOSTIC_SOURCE = "ets";

function log(...args) {
  process.stderr.write(`[ets-content-mapper] ${args.join(" ")}\n`);
}

let buffer = Buffer.alloc(0);

process.stdin.on("data", (chunk) => {
  buffer = buffer.length === 0 ? chunk : Buffer.concat([buffer, chunk]);
  try {
    drain();
  } catch (err) {
    log("fatal:", err && err.stack || err);
    process.exit(1);
  }
});

process.stdin.on("end", () => process.exit(0));

function drain() {
  for (;;) {
    const headerEnd = buffer.indexOf("\r\n\r\n");
    if (headerEnd === -1) return;
    const header = buffer.toString("ascii", 0, headerEnd);
    const match = /(?:^|\r\n)Content-Length: *(\d+)/i.exec(header);
    if (!match) throw new Error(`malformed header: ${JSON.stringify(header)}`);
    const length = Number(match[1]);
    const start = headerEnd + 4;
    if (buffer.length < start + length) return;
    const payload = buffer.toString("utf8", start, start + length);
    buffer = buffer.subarray(start + length);
    let message;
    try {
      message = JSON.parse(payload);
    } catch {
      send({ jsonrpc: "2.0", id: null, error: { code: -32700, message: "Parse error" } });
      continue;
    }
    handleMessage(message);
  }
}

function send(message) {
  const data = Buffer.from(JSON.stringify(message), "utf8");
  process.stdout.write(`Content-Length: ${data.length}\r\n\r\n`);
  process.stdout.write(data);
}

function handleMessage(message) {
  if (message.id === undefined || message.id === null) {
    return;
  }
  try {
    const result = dispatch(message.method, message.params ?? {});
    send({ jsonrpc: "2.0", id: message.id, result });
  } catch (err) {
    send({
      jsonrpc: "2.0",
      id: message.id,
      error: { code: typeof err.code === "number" ? err.code : -32603, message: String(err && err.message || err) },
    });
  }
}

function dispatch(method, params) {
  switch (method) {
    case "initialize":
      return initialize(params);
    case "transform":
      return transform(params);
    default: {
      const err = new Error(`Method not found: ${method}`);
      err.code = -32601;
      throw err;
    }
  }
}

function initialize(params) {
  if (params.protocolVersion !== PROTOCOL_VERSION) {
    throw new Error(`unsupported protocol version: ${params.protocolVersion}`);
  }
  const encodings = Array.isArray(params.positionEncodings) ? params.positionEncodings : [];
  if (!encodings.includes("utf-16")) {
    throw new Error("host does not support utf-16 position encoding");
  }
  return {
    protocolVersion: PROTOCOL_VERSION,
    positionEncoding: "utf-16",
    diagnosticSource: DIAGNOSTIC_SOURCE,
  };
}
