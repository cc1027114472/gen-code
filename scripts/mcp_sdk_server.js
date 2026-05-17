#!/usr/bin/env node

const { z } = require("../../CC ibwhale/node_modules/zod/v4");
const { McpServer } = require("../../CC ibwhale/node_modules/@modelcontextprotocol/sdk/dist/cjs/server/mcp.js");
const { StdioServerTransport } = require("../../CC ibwhale/node_modules/@modelcontextprotocol/sdk/dist/cjs/server/stdio.js");

const server = new McpServer({
  name: "sdk-external-fixture",
  version: "1.0.0",
});

server.registerTool(
  "echo",
  {
    description: "Echo a message through the official MCP SDK server baseline",
    inputSchema: {
      message: z.string(),
    },
  },
  async ({ message }) => ({
    content: [{ type: "text", text: `echo:${message}` }],
    structuredContent: { echo: message },
  }),
);

server.registerTool(
  "sum",
  {
    description: "Sum integer values through the official MCP SDK server baseline",
    inputSchema: {
      values: z.array(z.number()),
    },
  },
  async ({ values }) => {
    const total = values.reduce((acc, value) => acc + value, 0);
    return {
      content: [{ type: "text", text: `total:${total}` }],
      structuredContent: { total },
    };
  },
);

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
});
