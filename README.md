<div align="center">

<a href="https://geo-mcp.abgeo.cloud">
  <img src="website/public/favicon.svg" alt="Georgia MCP" width="96" height="96">
</a>

<h1>Georgia MCP</h1>

<p><strong>Georgia's public data, inside your chat.</strong></p>

<p>
An MCP server for exchange rates, fuel prices, card offers, and Tbilisi public transport.<br>
Ask Claude, ChatGPT, or any MCP client in your own words.
</p>

[![Images](https://github.com/abgeo/georgia-mcp/actions/workflows/images.yml/badge.svg)](https://github.com/abgeo/georgia-mcp/actions/workflows/images.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/abgeo/georgia-mcp)](go.mod)
[![License: GPL v3](https://img.shields.io/github/license/abgeo/georgia-mcp)](LICENSE)
[![Listed on mcpservers.org](https://mcpservers.org/badge.svg)](https://mcpservers.org/servers/abgeo/georgia-mcp)

[Website](https://geo-mcp.abgeo.cloud) ·
[Claude directory](https://claude.ai/directory/georgia-mcp) ·
[Install](#install) ·
[Tools](#tools) ·
[Report a bug](https://github.com/abgeo/georgia-mcp/issues)

</div>

## Why

Georgian public data is spread over many sites: the National Bank, each commercial
bank, each fuel company, the Tbilisi Transport Company. Georgia MCP reads them for your
model, so you can ask one question and get one answer.

- **Read-only.** Every tool reads public data and changes nothing.
- **Sourced.** Every answer names its source and the time of the data, so you can check it.
- **Exact.** A rate or a price is a decimal string, never a rounded float.
- **No account.** The hosted server needs no sign-in and no key.

## What you can ask

> What is the official USD to GEL rate today?
>
> Which bank buys euros at the best rate right now?
>
> Where is diesel cheapest in Georgia?
>
> When does the next bus reach my stop?
>
> Which TBC Bank card offers give a discount at restaurants?

| Subject                | Sources                                                                                  |
| ---------------------- | ---------------------------------------------------------------------------------------- |
| Exchange rates         | National Bank of Georgia, Bank of Georgia, TBC Bank, Liberty Bank, Terabank, Rico Credit, Kursi.ge |
| Fuel prices            | SOCAR, Wissol, Gulf, Rompetrol, Lukoil, Portal, Connect, Neogas                          |
| Card offers            | Bank of Georgia, TBC Bank, Liberty Studio                                                |
| Tbilisi public transport | Tbilisi Transport Company                                                              |
| Postal codes           | Georgian Post                                                                            |

## Install

The hosted server runs at:

```text
https://geo-mcp.abgeo.cloud/mcp
```

### Claude

Open [Georgia MCP in the Claude directory](https://claude.ai/directory/georgia-mcp) and
select **Connect**. It then works on claude.ai, Claude Desktop, and the Claude mobile apps.

### Claude Code

```sh
claude mcp add --transport http georgia-mcp https://geo-mcp.abgeo.cloud/mcp
```

### ChatGPT

Open **Plugins**, select **+**, then **Add custom MCP server**. Paste the address above
and set the authentication to **None**. A custom server needs a paid plan.

### Cursor

Add this to `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "georgia-mcp": { "url": "https://geo-mcp.abgeo.cloud/mcp" }
  }
}
```

### VS Code

Add this to `.vscode/mcp.json` in your project:

```json
{
  "servers": {
    "georgia-mcp": { "type": "http", "url": "https://geo-mcp.abgeo.cloud/mcp" }
  }
}
```

### Run it yourself

With Docker:

```sh
docker run -p 8080:8080 \
  -e GEOMCP_TRANSIT_TTC_API_KEY=... \
  -e GEOMCP_RATES_TBC_API_KEY=... \
  ghcr.io/abgeo/georgia-mcp
```

Then connect your client to `http://localhost:8080/mcp`.

With Go, over stdio:

```sh
go install github.com/abgeo/georgia-mcp/cmd/georgia-mcp@latest
claude mcp add georgia-mcp -- georgia-mcp stdio
```

Two sources need a key:

| Setting                      | Source                    | Without the key                          |
| ---------------------------- | ------------------------- | ---------------------------------------- |
| `GEOMCP_TRANSIT_TTC_API_KEY` | Tbilisi Transport Company | Set `GEOMCP_TRANSIT_TTC_ENABLED=false`.  |
| `GEOMCP_RATES_TBC_API_KEY`   | TBC Bank developer API    | Set `GEOMCP_RATES_TBC_ENABLED=false`.    |

[`configs/.env.example`](configs/.env.example) lists every setting.

## Tools

| Tool                          | Answers                                                                 |
| ----------------------------- | ----------------------------------------------------------------------- |
| `rates_get_official_rates`    | The official rate of the lari for today or a past date.                 |
| `rates_list_official_history` | The official rate of one currency for each day of a period.             |
| `rates_get_commercial_rates`  | The buy and sell rates of the commercial banks right now.               |
| `fuel_get_prices`             | Today's fuel prices of each company, grouped by grade.                  |
| `offers_search_offers`        | The current card offers of the banks.                                   |
| `offers_get_offer`            | The full terms of one card offer.                                       |
| `transit_search_stops`        | Stops of Tbilisi by name or code.                                       |
| `transit_list_nearby_stops`   | Stops near a point, nearest first.                                      |
| `transit_list_arrivals`       | The next vehicles at a stop, live.                                      |
| `transit_search_routes`       | Routes of Tbilisi by number or name.                                    |
| `transit_get_route`           | One route with each direction and every stop.                           |
| `transit_get_timetable`       | The planned departures of a route from one stop.                        |
| `transit_search_places`       | A place in Tbilisi by name or address.                                  |
| `transit_plan_trip`           | A trip by public transport between two points of Tbilisi.               |
| `postal_search_codes`         | The postal code of an address, a street, a town, or a village.          |

The [website](https://geo-mcp.abgeo.cloud) shows the input and a real output of each tool.

## Development

Georgia MCP follows spec driven development. The code comes from the documents in
[`docs/specs/`](docs/specs/README.md). Read that guide before you open a pull request.

You need Go (the version in [`go.mod`](go.mod)) and [Task](https://taskfile.dev).

```sh
cp configs/.env.example .env  # then fill in the keys
task run                      # serve over HTTP on :8080
task lint test build          # check the server
```

Each vendor client is a package under `pkg/<vendor>` that another project can import. `task --list` shows every task.

## Contributing

Bug reports and ideas are welcome in [issues](https://github.com/abgeo/georgia-mcp/issues).
To add a source or a tool, open an issue first: each one starts as a requirements document.
Read the [contributing guide](.github/CONTRIBUTING.md) before you open a pull request.

## Privacy and security

The hosted server keeps an access log and a record of each tool call, never the input of a
call. Read the [privacy policy](https://geo-mcp.abgeo.cloud/privacy/) and the
[terms of service](https://geo-mcp.abgeo.cloud/terms/).

To report a vulnerability, follow the [security policy](.github/SECURITY.md). Please don't open a
public issue.

## Disclaimer

Georgia MCP is not an official service. It is not affiliated with the National Bank of
Georgia or any bank, fuel company, transport operator, or postal operator it reads. The data belongs to its
publishers and can be late or wrong. Check with the source before you decide.

## Author

Made with ❤️ in Georgia by [Temuri Takalandze (ABGEO)](https://www.abgeo.dev).

## License

Georgia MCP is free software under the [GNU General Public License v3.0](LICENSE).
