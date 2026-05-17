#!/usr/bin/env node
import { z } from "../../../../../CC ibwhale/node_modules/zod/v4/index.js";
import { McpServer } from "../../../../../CC ibwhale/node_modules/@modelcontextprotocol/sdk/dist/esm/server/mcp.js";
import { StdioServerTransport } from "../../../../../CC ibwhale/node_modules/@modelcontextprotocol/sdk/dist/esm/server/stdio.js";

function isValidTimezone(timezone) {
  try {
    Intl.DateTimeFormat(undefined, { timeZone: timezone });
    return true;
  } catch {
    return false;
  }
}

const server = new McpServer({
  name: "third-party-time",
  version: "1.0.1",
});

server.registerTool(
  "get_current_time",
  {
    description: "Return the current time for a requested IANA timezone.",
    inputSchema: {
      timezone: z.string(),
    },
  },
  async ({ timezone }) => {
    if (!isValidTimezone(timezone)) {
      throw new Error(`Invalid timezone: ${timezone}`);
    }
    const now = new Date();
    const formatted = new Intl.DateTimeFormat("en-US", {
      timeZone: timezone,
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
      timeZoneName: "long",
    }).format(now);

    return {
      content: [{ type: "text", text: formatted }],
      structuredContent: {
        timezone,
        datetime: now.toISOString(),
        formatted_time: formatted,
      },
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
