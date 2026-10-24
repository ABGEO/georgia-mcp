type Widen<T> = T extends string
  ? string
  : T extends (...args: infer A) => infer R
    ? (...args: A) => R
    : { [K in keyof T]: Widen<T[K]> }

export const en = {
  meta: {
    title: "Georgia MCP: Georgian public data in your chat",
    description:
      "An MCP server for Georgian public data. Ask Claude, ChatGPT, or your favorite AI assistant about exchange rates, fuel prices, Tbilisi transit, card offers, and postal codes.",
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
    lead: "Ask Claude, ChatGPT, or your favorite AI assistant about the lari, fuel prices, Tbilisi buses, card offers, or postal codes.",
    install: "Install",
    tools: "See the tools",
    stamp: "Read-only",
  },
  contexts: {
    heading: "What you can ask",
    lead: "Five subjects, each read from the organizations that publish the data.",
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
          "Open Georgia MCP in the Claude directory.",
          "Select Connect.",
          "To add it as a custom connector instead, use the address below.",
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
      claudeCode: {
        name: "Claude Code",
        steps: [
          "If you added Georgia MCP on claude.ai, it is already in Claude Code under the same account. Run `/mcp` to check: it is listed as claude.ai Georgia MCP.",
          "To add it yourself, for example when you sign in with an API key, run this command in a terminal.",
        ],
      },
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
    directory: "the Claude directory",
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
  legal: {
    effective: "Effective date",
    privacy: {
      title: "Privacy policy",
      description: "What Georgia MCP collects, what it logs, and how long it keeps it.",
      effective: "2026-10-24",
      sections: [
        {
          heading: "Scope",
          paragraphs: [
            "This policy covers Georgia MCP: the server at https://geo-mcp.abgeo.cloud/mcp and the website at https://geo-mcp.abgeo.cloud.",
            "It does not cover a copy of the server that you run on your own computer. That copy sends nothing to us.",
          ],
        },
        {
          heading: "Who we are",
          paragraphs: [
            "Temuri Takalandze, operating as ABGEO, runs Georgia MCP from Tbilisi, Georgia. Write to me@abgeo.dev about anything in this policy.",
          ],
        },
        {
          heading: "No account",
          paragraphs: [
            "Georgia MCP has no sign-in. It never asks for your name, your email, a payment, or any other personal data. The website has no form.",
          ],
        },
        {
          heading: "What the server receives",
          paragraphs: [
            "When you use Georgia MCP in a chat app, such as ChatGPT or Claude, the app sends the server the input of each tool call: a currency, a fuel grade, a stop, or a place name, for example.",
            "The server never sees your conversation. The chat app handles it under its own privacy policy.",
          ],
        },
        {
          heading: "What we log",
          paragraphs: [
            "The server and the website keep three records, and nothing else.",
            "The access log of the server: for each request, the time, the method, the path, the status, the duration, the IP address, and the user agent.",
            "The record of a tool call: the name of the tool, the duration, and the type of an error, if one happens.",
            "The access log of the website: for each request, the time, the method, the path, the status, the size of the response, the IP address, the user agent, and the referrer.",
            "The server does not keep the input of a tool call. It does not keep the content of a request or a response.",
          ],
        },
        {
          heading: "Why we log",
          paragraphs: [
            "We use the logs to run the service, find and fix errors, and protect it from abuse. We don't use them for advertising or profiles. We don't sell them or share them for marketing.",
          ],
        },
        {
          heading: "How long we keep the logs",
          paragraphs: [
            "We keep the logs only as long as we need them to run and protect the service. Then we delete them.",
          ],
        },
        {
          heading: "Cookies",
          paragraphs: [
            "The website sets no cookie and runs no analytics. When you choose a theme, your browser keeps the choice in its local storage. It never leaves your browser.",
          ],
        },
        {
          heading: "Public sources",
          paragraphs: [
            "To answer a tool call, the server reads public data from the organizations that publish it: the National Bank of Georgia, Georgian banks and exchange bureaus, fuel companies, and the Tbilisi Transport Company.",
            "The server sends a source only the input that the call needs. It never sends your IP address. A text that you search for, such as the query of an offer search, reaches the source as you wrote it. Don't put personal data in it.",
          ],
        },
        {
          heading: "Security",
          paragraphs: [
            "Every request to the server and the website travels over HTTPS.",
          ],
        },
        {
          heading: "Your rights",
          paragraphs: [
            "Depending on where you live, a law such as the GDPR, the Georgian Law on Personal Data Protection, or the CCPA may give you rights over your personal data: to access it, correct it, delete it, or object to its use.",
            "To ask about the log records of your IP address, or to delete them, write to me@abgeo.dev. We answer within the period that the law sets.",
          ],
        },
        {
          heading: "Children",
          paragraphs: [
            "Georgia MCP is not made for children under 16, and it collects no data that identifies a child.",
          ],
        },
        {
          heading: "Changes",
          paragraphs: [
            "When this policy changes, we update this page and its effective date.",
          ],
        },
        {
          heading: "Contact",
          paragraphs: ["Temuri Takalandze, Tbilisi, Georgia. Email: me@abgeo.dev."],
        },
      ],
    },
    terms: {
      title: "Terms of service",
      description: "The terms of use of the Georgia MCP server and website.",
      effective: "2026-10-24",
      sections: [
        {
          heading: "Scope",
          paragraphs: [
            "These terms cover Georgia MCP: the server at https://geo-mcp.abgeo.cloud/mcp and the website at https://geo-mcp.abgeo.cloud. Temuri Takalandze, operating as ABGEO, runs them.",
            "When you use the server or the website, you accept these terms. If you don't accept them, don't use Georgia MCP.",
          ],
        },
        {
          heading: "The service",
          paragraphs: [
            "Georgia MCP is free. It answers with public data and changes nothing: it never makes a payment, an exchange, a booking, or any other transaction.",
            "The source code is open at https://github.com/abgeo/georgia-mcp. Its license governs the code. These terms govern the use of the hosted server and the website.",
          ],
        },
        {
          heading: "The data",
          paragraphs: [
            "The data comes from the organizations that publish it, not from us. It can be late, incomplete, or wrong. Each answer names its source and the time of the data, so you can check it.",
            "A rate or a price is not an offer of a bank or a company. An answer is not financial advice. Check with the source before you decide.",
          ],
        },
        {
          heading: "Chat apps",
          paragraphs: [
            "You use Georgia MCP through a chat app, such as ChatGPT or Claude. The terms of that app also apply. The model of the app writes the final answer, and we don't control what it writes.",
          ],
        },
        {
          heading: "Acceptable use",
          paragraphs: [
            "Use Georgia MCP only for lawful purposes. Don't overload the server, try to break it, or get around its limits. Don't use it to copy the data of a source in bulk.",
            "We may limit or block access that harms the service or a source.",
          ],
        },
        {
          heading: "No warranty",
          paragraphs: [
            "Georgia MCP is provided “as is” and “as available”, with no warranty of any kind, express or implied. We don't promise that it will be available, free of errors, or correct.",
          ],
        },
        {
          heading: "Limitation of liability",
          paragraphs: [
            "As far as the law allows, we are not liable for any indirect, incidental, special, or consequential damage, or for any loss of data, revenue, or profit, from your use of Georgia MCP or from a decision based on its data. Georgia MCP is free, so our total liability is zero. Nothing in these terms limits a liability that the law does not let us limit.",
          ],
        },
        {
          heading: "Changes to the service",
          paragraphs: [
            "We may change, limit, or stop Georgia MCP, or any tool, at any time, with no notice.",
          ],
        },
        {
          heading: "Governing law",
          paragraphs: [
            "The laws of Georgia govern these terms. This does not take away a consumer protection that the law of your country gives you.",
          ],
        },
        {
          heading: "Changes to these terms",
          paragraphs: [
            "When these terms change, we update this page and its effective date. If you use Georgia MCP after a change, you accept the new terms.",
          ],
        },
        {
          heading: "Contact",
          paragraphs: ["Temuri Takalandze, Tbilisi, Georgia. Email: me@abgeo.dev."],
        },
      ],
    },
  },
  footer: {
    repo: "Source on GitHub",
    label: "Legal and source",
    privacy: "Privacy",
    terms: "Terms",
  },
}

/** The shape of the copy of one language. */
export type Dictionary = Widen<typeof en>
