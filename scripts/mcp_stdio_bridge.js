#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { Client } = require("../../CC ibwhale/node_modules/@modelcontextprotocol/sdk/dist/cjs/client/index.js");
const { StdioClientTransport } = require("../../CC ibwhale/node_modules/@modelcontextprotocol/sdk/dist/cjs/client/stdio.js");

async function readJSON() {
  const chunks = [];
  for await (const chunk of process.stdin) {
    chunks.push(Buffer.from(chunk));
  }
  const raw = Buffer.concat(chunks).toString("utf8").trim();
  if (!raw) {
    throw new Error("empty request");
  }
  return JSON.parse(raw);
}

function normalizeContent(result) {
  if (!result || !Array.isArray(result.content)) {
    return "";
  }
  return result.content
    .map((item) => {
      if (item && typeof item.text === "string") {
        return item.text.trim();
      }
      return "";
    })
    .filter(Boolean)
    .join(" ");
}

async function main() {
  const payload = await readJSON();
  const command = Array.isArray(payload.command) ? payload.command : [];
  if (command.length === 0) {
    throw new Error("missing external MCP command");
  }

  const client = new Client({
    name: "gen-code-mcp-bridge",
    version: "1.0.0",
  });

  const transport = new StdioClientTransport({
    command: command[0],
    args: command.slice(1),
    cwd: payload.cwd || path.dirname(command[0]),
    env: payload.env || {},
    stderr: "pipe",
  });

  let stderr = "";
  if (transport.stderr) {
    transport.stderr.on("data", (chunk) => {
      stderr += Buffer.from(chunk).toString("utf8");
    });
  }

  await client.connect(transport);
  try {
    await client.listTools();
    const result = await client.callTool({
      name: String(payload.toolName || "").trim(),
      arguments: payload.arguments || {},
    });

    const text = normalizeContent(result);
    const summary =
      String(payload.summary || "").trim() ||
      text ||
      `mcp tool ${String(payload.serverId || "").trim()}/${String(payload.toolName || "").trim()} executed`;

    const structured = result.structuredContent && typeof result.structuredContent === "object"
      ? result.structuredContent
      : {};

    process.stdout.write(
      JSON.stringify({
        ok: !result.isError,
        error: result.isError ? text || stderr.trim() || "tool returned error" : "",
        summary,
        result: structured,
      }),
    );
  } finally {
    await client.close();
    await transport.close();
  }
}

main().catch((error) => {
  process.stdout.write(
    JSON.stringify({
      ok: false,
      error: error instanceof Error ? error.message : String(error),
    }),
  );
  process.exit(0);
});
