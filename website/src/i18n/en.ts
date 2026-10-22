type Widen<T> = T extends string
  ? string
  : T extends (...args: infer A) => infer R
    ? (...args: A) => R
    : { [K in keyof T]: Widen<T[K]> }

export const en = {
  meta: {
    title: "Georgia MCP: Georgian public data in your chat",
    description:
      "An MCP server for Georgian public data. Ask Claude, ChatGPT, or your favorite AI assistant about exchange rates, fuel prices, Tbilisi transit, and card offers.",
    tool: (title: string) => `${title} · Georgia MCP`,
    locale: "en_US",
  },
  nav: {
    skip: "Skip to the content",
    home: "Georgia MCP, home",
    tools: "Tools",
    install: "Install",
    languageLabel: "Read this page in Georgian",
    theme: "Theme",
    themes: { system: "System", light: "Light", dark: "Dark" },
  },
  intro: {
    eyebrow: "An MCP server for Georgia",
    title: "Georgia's public data,",
    titleAccent: "inside your chat.",
    lead: "Ask Claude, ChatGPT, or your favorite AI assistant about the lari, fuel prices, Tbilisi buses, or card offers.",
    install: "Install",
    tools: "See the tools",
    stamp: "Read-only",
  },
  contexts: {
    heading: "What you can ask",
    lead: "Four subjects, each read from the organizations that publish the data.",
    tools: (count: number) => (count === 1 ? "1 tool" : `${count} tools`),
    sources: "Sources",
    more: (count: number) => `+${count} more`,
  },
  examples: {
    heading: "Real answers",
    lead: "Ask in your own words. The model calls a tool, reads the source, and answers. These are examples, not live data.",
  },
  install: {
    heading: "Connect your client",
    lead: "Add the server once. Then ask in your own words.",
    remote: "Hosted server",
    local: "Prefer to run it yourself?",
    address: "Server address",
    copy: "Copy",
    copied: "Copied",
    fallback: "Press Ctrl+C to copy",
    guides: {
      claude: {
        name: "claude.ai",
        steps: [
          "On claude.ai, open Customize, then Connectors.",
          "Select Add custom connector.",
          "Name it Georgia MCP, paste the address below, and choose No sign-in. The Free plan allows one custom connector.",
        ],
      },
      claudeApps: {
        name: "Claude Desktop and mobile",
        steps: [
          "Add the connector once on claude.ai, as the claude.ai tab shows.",
          "It then appears in Claude Desktop and the Claude mobile apps under the same account, with nothing to install.",
          "To run it on your computer instead, see the “Prefer to run it yourself?” tabs.",
        ],
      },
      claudeCode: { name: "Claude Code", steps: ["Run this command in a terminal."] },
      chatgpt: {
        name: "ChatGPT",
        steps: [
          "On chatgpt.com, open Plugins and select the + button.",
          "Choose Add custom MCP server.",
          "Name it Georgia MCP, paste the address below, set the authentication to None, and accept the warning. It needs a paid plan.",
        ],
      },
      cursor: { name: "Cursor", steps: ["Add this to ~/.cursor/mcp.json."] },
      vscode: { name: "VS Code", steps: ["Add this to .vscode/mcp.json in your project."] },
      binary: {
        name: "Binary",
        steps: [
          "Download georgia-mcp for your system from the latest release, or install it with Go.",
          "Connect Claude Code to it over stdio.",
        ],
      },
      docker: {
        name: "Docker",
        steps: ["Start the container.", "Connect your client to the local address."],
      },
    },
    keys: {
      ttc: "The transit tools need a TTC API key in GEOMCP_TRANSIT_TTC_API_KEY. To start without them, set GEOMCP_TRANSIT_TTC_ENABLED=false.",
      tbc: "The commercial rates of TBC Bank need a TBC API key in GEOMCP_RATES_TBC_API_KEY. To start without TBC Bank, set GEOMCP_RATES_TBC_ENABLED=false.",
    },
    download: "Latest release",
  },
  how: {
    heading: "How it works",
    steps: {
      ask: { title: "You ask", text: "In your own words, in English or in Georgian." },
      call: {
        title: "The model calls a tool",
        text: "Georgia MCP reads the data from the organization that publishes it.",
      },
      answer: {
        title: "You get the answer",
        text: "With the name of the source and the time of the data, so you can check it.",
      },
    },
    counts: { tools: "Tools", contexts: "Subjects", vendors: "Sources" },
  },
  tools: {
    heading: "Every tool",
    lead: "Each tool reads public data and changes nothing. Open one to see its input and a real output.",
  },
  tool: {
    back: "All tools",
    definition: "Definition",
    inputs: "Input",
    noInputs: "This tool takes no input.",
    columns: { name: "Name", type: "Type", required: "Required", default: "Default", description: "Description" },
    yes: "Yes",
    no: "No",
    examples: "Examples",
    stamp: "Read-only",
  },
  chat: {
    you: "You",
    model: "Model",
    calling: "Calling",
  },
  ticket: {
    asked: "You asked",
    input: "Input",
    output: "Output",
  },
  notFound: {
    title: "This page does not exist",
    text: "The address may be old or mistyped.",
    home: "Go to the home page",
  },
  footer: {
    repo: "Source on GitHub",
  },
}

/** The shape of the copy of one language. */
export type Dictionary = Widen<typeof en>
