"use strict";

// ETS language extension.
//
// Registers the .ets language and performs the "first wake" handshake
// described in the typescript-go content mapper proposal: when an .ets
// file is opened before TypeScript's server knows about the project's
// contentMappers, this extension activates the TypeScript 7
// (native-preview) extension and asks it to discover content mappers
// for open .ets documents.

const vscode = require("vscode");

const ETS_EXTENSIONS = [".ets"];

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
    })
  );

  discover();
}

function deactivate() {}

module.exports = { activate, deactivate };
