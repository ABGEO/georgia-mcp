# neogas

A Go client of the gas price of Neogas, a network of gas stations of Georgia that sells
compressed natural gas.

Neogas publishes no API and no price list. The client reads the price of the calculator
of the home page [neogas.ge](https://neogas.ge/ka/), a web page for people. The page can
change with no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/neogas
```

```go
client := neogas.New()

prices, err := client.GasPrices(ctx)
if err != nil {
    return err
}

fmt.Println(prices) // [1.60 1.60]
```

## The page

| Method      | Request      | Answer                                          |
| ----------- | ------------ | ----------------------------------------------- |
| `GasPrices` | `GET /ka/`   | The text of each gas price of the calculator    |

The page needs no key. This client mirrors what the page showed on 2026-10-21:

- The calculator has two panes, a jeep and a car. Each shows "გაზის ფასი" with the same
  value in an element of the class `gas_price`: `1.60`.
- The client returns the text of each such element, in the order of the page. A page
  with none gives an empty list. The meaning of zero, one, or two values belongs to the
  caller.
- The page names no unit, no date, and no product. The stations sell CNG by the cubic
  metre.
- The page holds the text of PHP notices. They sit outside the price.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
