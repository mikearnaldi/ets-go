"use strict";

// Minimal client for the TypeScript 7 API session protocol. The
// native-preview extension hands out a per-session pipe via the
// typescript.native-preview.initializeAPIConnection command; over that
// pipe the server speaks JSON-RPC (LSP-style Content-Length framing)
// and serves the compiler API (snapshots, checker queries, ...).
//
// We hand-roll the transport instead of depending on
// @typescript/native-preview because the published client keeps its
// raw-request channel private and does not know our custom
// getExpressionTypeAtPosition method.

const net = require("node:net");
const {
  createMessageConnection,
  SocketMessageReader,
  SocketMessageWriter,
} = require("vscode-jsonrpc/node");

class TsApiClient {
  constructor() {
    /** @type {import("vscode-jsonrpc").MessageConnection | undefined} */
    this.connection = undefined;
    /** @type {Promise<import("vscode-jsonrpc").MessageConnection> | undefined} */
    this.connecting = undefined;
  }

  /** @returns {Promise<import("vscode-jsonrpc").MessageConnection>} */
  connect() {
    if (this.connection) {
      return Promise.resolve(this.connection);
    }
    if (!this.connecting) {
      this.connecting = this.open()
        .then((connection) => {
          this.connection = connection;
          return connection;
        })
        .finally(() => {
          this.connecting = undefined;
        });
    }
    return this.connecting;
  }

  /** @returns {Promise<import("vscode-jsonrpc").MessageConnection>} */
  async open() {
    const vscode = require("vscode");
    const pipe = await vscode.commands.executeCommand(
      "typescript.native-preview.initializeAPIConnection"
    );
    if (typeof pipe !== "string" || pipe.length === 0) {
      throw new Error("native-preview did not return an API pipe path");
    }
    const socket = net.createConnection(pipe);
    await new Promise((resolve, reject) => {
      socket.once("connect", resolve);
      socket.once("error", reject);
    });
    const connection = createMessageConnection(
      new SocketMessageReader(socket),
      new SocketMessageWriter(socket)
    );
    connection.listen();
    const reset = () => {
      if (this.connection === connection) {
        this.connection = undefined;
      }
      connection.dispose();
    };
    socket.once("close", reset);
    socket.once("error", reset);
    await connection.sendRequest("initialize", null);
    return connection;
  }

  /**
   * Returns the display string of the type of the outermost expression
   * starting at `offset` (a UTF-16 offset in the original document), or
   * undefined when the position has no expression type.
   * @param {string} uri @param {number} offset @returns {Promise<string | undefined>}
   */
  async getExpressionTypeString(uri, offset) {
    const connection = await this.connect();
    const snapshotResponse = await connection.sendRequest("updateSnapshot", {});
    const snapshot = snapshotResponse.snapshot;
    try {
      const project = await connection.sendRequest("getDefaultProjectForFile", {
        snapshot,
        file: { uri },
      });
      if (!project) {
        return undefined;
      }
      const type = await connection.sendRequest("getExpressionTypeAtPosition", {
        snapshot,
        project: project.id,
        file: { uri },
        position: offset,
      });
      if (!type) {
        return undefined;
      }
      return await connection.sendRequest("typeToString", {
        snapshot,
        project: project.id,
        type: type.id,
      });
    } finally {
      connection.sendRequest("release", { snapshot }).then(undefined, () => {});
    }
  }

  dispose() {
    if (this.connection) {
      this.connection.dispose();
      this.connection = undefined;
    }
  }
}

module.exports = { TsApiClient };
