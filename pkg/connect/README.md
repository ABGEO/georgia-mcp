# connect

A Go client of the fuel price API of Connect, a fuel company of Georgia. It covers the
current prices.

The API is unofficial. A plugin of the site serves it, and the page
[connect.com.ge/საწვავი](https://connect.com.ge/%E1%83%A1%E1%83%90%E1%83%AC%E1%83%95%E1%83%90%E1%83%95%E1%83%98/)
shows the same prices. Connect publishes no documentation of it. The API can change with
no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/connect
```

```go
client := connect.New()

snapshot, err := client.Current(ctx)
if err != nil {
    return err
}

for key, price := range snapshot.Prices {
    fmt.Println(key, price.Label, price.Standard, price.SelfService)
}
```

## The API

| Method    | Request        | Answer                                         |
| --------- | -------------- | ---------------------------------------------- |
| `Current` | `GET /current` | Every product with its standard and self-service price |

The API needs no key. This client mirrors what the API answered on 2026-10-21:

- **One answer holds every product**, in `prices`, under a key: `super_ron98`.
- Each product has a Georgian label, a standard price, and a self-service price, in GEL
  for one litre. The site has no English name.
- **A price is the full digits of a binary float**:
  `4.3499999999999996447286321199499070644378662109375` for 4.35. `json.Number` keeps
  them. The shortest text that parses to the same `float64` is the price that Connect
  entered.
- `date` is the day of the snapshot, not the day of the last change. Connect writes a
  snapshot each day at 04:00, and a snapshot with `is_manual` true when it enters
  prices by hand.
- `/history` and `/date/{date}` give the past days. A day with no snapshot answers 404
  with a JSON message. The client does not cover them.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
