"use strict";

// ETS language extension.
//
// Registers the .ets language and performs the "first wake" handshake
// described in the typescript-go content mapper proposal: when an .ets
// file is opened before TypeScript's server knows about the project's
// contentMappers, this extension activates the TypeScript 7
// (native-preview) extension and asks it to discover content mappers
// for open .ets documents.
//
// Also provides hover for the `run` keyword: the content mapper protocol
// gives the server no way to attach quickinfo to keywords, so hovering
// `run` would show nothing (same as `yield` in plain TS). Instead we
// detect `run` here and ask the TypeScript 7 API (over the API session
// pipe, see tsapi.js) for the type of the operand expression, which the
// server computes from the generated code and maps back for us. If the
// API is unavailable we fall back to re-presenting the operand's
// regular hover.

const vscode = require("vscode");
const { TsApiClient } = require("./tsapi");

const ETS_EXTENSIONS = [".ets"];

// Mirrors the operand-start exclusions of the `run` keyword in the ETS
// grammar: `run` is only a keyword when followed on the same line by an
// expression that does not start with one of these.
const NON_RUN_OPERAND_START = new Set(["(", "[", "`", "+", "-", "/"]);

/** @param {vscode.ExtensionContext} context */
async function activate(context) {
  const nativePreview = vscode.extensions.getExtension("TypeScriptTeam.native-preview");
  if (!nativePreview) {
    void vscode.window.showWarningMessage(
      "ETS: the TypeScript 7 (TypeScriptTeam.native-preview) extension is required but was not found."
    );
    return;
  }
  await nativePreview.activate();

  const apiClient = new TsApiClient();

  const discover = () => {
    const uris = vscode.workspace.textDocuments
      .filter((document) => document.languageId === "ets")
      .map((document) => document.uri);
    if (uris.length === 0) {
      return;
    }
    void vscode.commands.executeCommand("typescript.native-preview.discoverContentMappers", {
      uris,
      extensions: ETS_EXTENSIONS,
    });
  };

  context.subscriptions.push(
    vscode.workspace.onDidOpenTextDocument((document) => {
      if (document.languageId === "ets") {
        discover();
      }
    }),
    vscode.languages.registerHoverProvider({ language: "ets" }, new EtsRunHoverProvider(apiClient)),
    { dispose: () => apiClient.dispose() }
  );

  discover();
}

class EtsRunHoverProvider {
  /** @param {TsApiClient} apiClient */
  constructor(apiClient) {
    this.apiClient = apiClient;
  }

  /** @param {vscode.TextDocument} document @param {vscode.Position} position @param {vscode.CancellationToken} token */
  async provideHover(document, position, token) {
    const wordRange = document.getWordRangeAtPosition(position);
    if (!wordRange || document.getText(wordRange) !== "run") {
      return undefined;
    }
    const line = document.lineAt(position.line).text;
    // Property accesses such as `foo.run` or `foo?.run` are plain identifiers.
    let before = wordRange.start.character - 1;
    while (before >= 0 && (line[before] === " " || line[before] === "\t")) {
      before--;
    }
    if (before >= 0 && line[before] === ".") {
      return undefined;
    }
    // The operand must start on the same line.
    let operand = wordRange.end.character;
    while (operand < line.length && (line[operand] === " " || line[operand] === "\t")) {
      operand++;
    }
    if (operand >= line.length || NON_RUN_OPERAND_START.has(line[operand])) {
      return undefined;
    }
    if (token.isCancellationRequested) {
      return undefined;
    }
    const operandPosition = new vscode.Position(position.line, operand);
    const typeString = await this.expressionTypeString(document, operandPosition);
    if (typeString) {
      const markdown = new vscode.MarkdownString();
      markdown.appendCodeblock(typeString, "typescript");
      return new vscode.Hover(markdown, wordRange);
    }
    return this.fallbackHover(document, operandPosition);
  }

  /** @param {vscode.TextDocument} document @param {vscode.Position} operandPosition @returns {Promise<string | undefined>} */
  async expressionTypeString(document, operandPosition) {
    try {
      return await this.apiClient.getExpressionTypeString(
        document.uri.toString(),
        document.offsetAt(operandPosition)
      );
    } catch {
      return undefined;
    }
  }

  /** @param {vscode.TextDocument} document @param {vscode.Position} operandPosition @returns {Promise<vscode.Hover | undefined>} */
  async fallbackHover(document, operandPosition) {
    // This re-enters our provider at the operand position; that is fine
    // because the operand is not `run` (or is a nested `run`, which then
    // resolves its own operand one position further to the right).
    const hovers = await vscode.commands.executeCommand(
      "vscode.executeHoverProvider",
      document.uri,
      operandPosition
    );
    if (!Array.isArray(hovers)) {
      return undefined;
    }
    for (const hover of hovers) {
      if (hover.contents.length > 0) {
        return new vscode.Hover(hover.contents);
      }
    }
    return undefined;
  }
}

function deactivate() {}

module.exports = { activate, deactivate };
