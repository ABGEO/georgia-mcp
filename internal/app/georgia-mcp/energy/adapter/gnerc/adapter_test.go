package gnerc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	gnercadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/gnerc"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	gnercclient "github.com/abgeo/georgia-mcp/pkg/gnerc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata     = "../../../../../../pkg/gnerc/testdata/"
	lastModified = "Wed, 21 Oct 2026 09:45:18 GMT"
	homeFile     = "home.html"
	balanceFile  = "balance.xlsx"

	balanceName     = "2015 - 2026 ფაქტიური ბალანსი.xlsx"
	balancingName   = "საბალანსო ელექტროენერგიის ფასები 2015 - 2026 წლები.xlsx"
	capacityFeeName = "გარანტირებული სიმძლავრის საფასური  2015 - 2026 წლები.xlsx"
	deregulatedName = "დერეგულირებული სადგურების ფასები 2015 - 2026 წლები.xlsx"
)

type answer struct {
	status int
	file   string
	body   string
}

func ok(file string) answer {
	return answer{status: http.StatusOK, file: file}
}

func clock() time.Time {
	return time.Date(2026, time.November, 17, 3, 0, 0, 0, kernel.TbilisiZone())
}

// recorded answers the home page and each workbook of the recorded answers.
func recorded() map[string]answer {
	return map[string]answer{
		"/":             ok(homeFile),
		balanceName:     ok(balanceFile),
		balancingName:   ok("balancing.xlsx"),
		capacityFeeName: ok("capacity-fee.xlsx"),
		deregulatedName: ok("deregulated.xlsx"),
	}
}

// serve answers "/" and each name of a workbook in files, and any other path with 404.
func serve(t *testing.T, files map[string]answer) *gnercadapter.Adapter {
	t.Helper()

	paths := map[string]answer{}

	for name, reply := range files {
		if name == "/" {
			paths[name] = reply
		} else {
			paths["/excel/"+url.PathEscape(name)] = reply
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply, found := paths[r.URL.EscapedPath()]
		if !found {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		body := []byte(reply.body)

		if reply.file != "" {
			read, err := os.ReadFile(testdata + reply.file)
			if err != nil {
				t.Error(err)

				return
			}

			body = read
		}

		w.Header().Set("Last-Modified", lastModified)
		w.WriteHeader(reply.status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return gnercadapter.NewAdapter(gnercclient.New(vendorhttp.WithBaseURL(server.URL)), clock)
}

func month(year int, m time.Month) domain.Month {
	return domain.Month{Year: year, Month: m}
}

func monthFigures(t *testing.T, book app.BalanceBook, target domain.Month) domain.Figures {
	t.Helper()

	for _, year := range book.Years {
		for _, m := range year.Months {
			if m.Month == target {
				return m.Figures
			}
		}
	}

	t.Fatalf("the book has no month %s", target)

	return nil
}

// checkFigures compares each item of want, written with 3 decimals, and each item of
// absent, which must have no key.
func checkFigures(
	t *testing.T,
	period string,
	got domain.Figures,
	want map[domain.Item]string,
	absent ...domain.Item,
) {
	t.Helper()

	for item, value := range want {
		number, found := got[item]
		if !found || number.StringFixed(domain.Precision) != value {
			t.Errorf(
				"%s %s = %s (%v), want %s",
				period,
				item,
				number.StringFixed(domain.Precision),
				found,
				value,
			)
		}
	}

	for _, item := range absent {
		if _, found := got[item]; found {
			t.Errorf("%s holds %s, want no key", period, item)
		}
	}
}

// GNERCMON-SC-009: The adapter maps the years from 2022 with their published months, each
// item by its label, and the time of the last change in Tbilisi.
func TestAdapterMapsTheBalance(t *testing.T) {
	t.Parallel()

	book, err := serve(t, recorded()).Balance(t.Context())
	if err != nil {
		t.Fatalf("Balance() error = %v", err)
	}

	counts := map[int]int{2022: 12, 2023: 12, 2024: 12, 2025: 12, 2026: 9}
	if len(book.Years) != len(counts) {
		t.Fatalf("Balance() = %d years, want 2022 to 2026", len(book.Years))
	}

	for _, year := range book.Years {
		if len(year.Months) != counts[year.Year] {
			t.Errorf("%d = %d months, want %d", year.Year, len(year.Months), counts[year.Year])
		}
	}

	checkRecordedBalance(t, book)
	checkTimes(t, book.Modified, book.ReadAt)
}

// checkRecordedBalance compares the values of the recorded balance in GNERCMON-SC-009.
func checkRecordedBalance(t *testing.T, book app.BalanceBook) {
	t.Helper()

	march := monthFigures(t, book, month(2026, time.March))
	checkFigures(t, "2026-03", march, map[domain.Item]string{
		domain.GenerationTotal: "1102.668", domain.GenerationSolar: "1.314",
		domain.GenerationHydro: "759.389", domain.ImportTotal: "228.916",
		domain.ImportRussia: "155.484", domain.Supply: "1306.646",
		domain.ConsumptionTotal: "1306.646", domain.ExportTurkey: "0.000",
		domain.DomesticConsumption: "1326.633", domain.ConsumptionFreeMarket: "1.727",
	})

	if len(march) != len(domain.Items()) {
		t.Errorf("2026-03 holds %d items, want %d", len(march), len(domain.Items()))
	}

	checkFigures(t, "2026", book.Years[4].Total, map[domain.Item]string{
		domain.GenerationTotal: "10308.382",
		domain.ImportTotal:     "1186.498",
		domain.ExportTurkey:    "169.008",
	})
	checkFigures(
		t,
		"2025-03",
		monthFigures(t, book, month(2025, time.March)),
		map[domain.Item]string{
			domain.GenerationSolar: "0.000", domain.ConsumptionFreeMarket: "1.576",
		},
	)
	checkFigures(
		t,
		"2024-03",
		monthFigures(t, book, month(2024, time.March)),
		map[domain.Item]string{
			domain.GenerationTotal: "1130.057", domain.ConsumptionFreeMarket: "0.609",
		},
		domain.GenerationSolar,
	)
	checkFigures(
		t,
		"2022-03",
		monthFigures(t, book, month(2022, time.March)),
		map[domain.Item]string{
			domain.GenerationTotal: "1076.963", domain.GenerationHydro: "570.784",
		},
		domain.GenerationSolar,
		domain.ConsumptionFreeMarket,
	)
}

func checkTimes(t *testing.T, modified, readAt time.Time) {
	t.Helper()

	want := time.Date(2026, time.October, 21, 13, 45, 18, 0, kernel.TbilisiZone())
	if !modified.Equal(want) || modified.Location().String() != kernel.TbilisiZone().String() {
		t.Errorf("Modified = %v, want %v", modified, want)
	}

	if !readAt.Equal(clock()) {
		t.Errorf("ReadAt = %v, want %v", readAt, clock())
	}
}

// GNERCMON-SC-010: A workbook that the adapter cannot read, a home page with no link of
// the balance, and a failure of GNERC are an outage of GNERC.
func TestAdapterRefusesABrokenBalance(t *testing.T) {
	t.Parallel()

	tests := map[string]answer{
		"no import": ok("balance-no-import.xlsx"),
		"bad value": ok("balance-bad-value.xlsx"),
		"not a zip": ok("not-zip.xlsx"),
		"500":       {status: http.StatusInternalServerError, body: "down"},
	}

	for name, reply := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			files := recorded()
			files[balanceName] = reply

			_, err := serve(t, files).Balance(t.Context())
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("Balance() error = %v, want kernel.ErrSourceUnavailable", err)
			}
		})
	}

	files := recorded()
	files["/"] = answer{status: http.StatusOK, body: `<a href="/excel/other.xlsx">other</a>`}

	if _, err := serve(
		t,
		files,
	).Balance(t.Context()); !errors.Is(
		err,
		kernel.ErrSourceUnavailable,
	) {
		t.Errorf("Balance() with no link error = %v, want kernel.ErrSourceUnavailable", err)
	}
}

func checkPrices(t *testing.T, got domain.MonthPrices, balancing, capacityFee, deregulated string) {
	t.Helper()

	text := func(price *decimal.Decimal) string {
		if price == nil {
			return "none"
		}

		return price.StringFixed(domain.Precision)
	}

	if text(got.Balancing) != balancing || text(got.CapacityFee) != capacityFee ||
		text(got.Deregulated) != deregulated {
		t.Errorf("%s = %s, %s, %s; want %s, %s, %s", got.Month,
			text(got.Balancing), text(got.CapacityFee), text(got.Deregulated),
			balancing, capacityFee, deregulated)
	}
}

func priceOf(t *testing.T, book app.PriceBook, target domain.Month) domain.MonthPrices {
	t.Helper()

	for _, prices := range book.Months {
		if prices.Month == target {
			return prices
		}
	}

	t.Fatalf("the book has no month %s", target)

	return domain.MonthPrices{}
}

// GNERCMON-SC-011: The adapter joins the three price workbooks into the published months,
// in order. An empty cell gives no price and keeps the other prices.
func TestAdapterMapsThePrices(t *testing.T) {
	t.Parallel()

	book, err := serve(t, recorded()).Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	if len(book.Months) != 141 || book.Months[0].Month != month(2015, time.January) ||
		book.Months[140].Month != month(2026, time.September) {
		t.Fatalf("Prices() = %d months, want 141 from 2015-01 to 2026-09", len(book.Months))
	}

	for i := 1; i < len(book.Months); i++ {
		if !book.Months[i-1].Month.Before(book.Months[i].Month) {
			t.Fatalf("Prices() is not in order at %s", book.Months[i].Month)
		}
	}

	checkPrices(t, priceOf(t, book, month(2015, time.January)), "13.365", "0.592", "7.397")
	checkPrices(t, priceOf(t, book, month(2026, time.September)), "16.068", "1.607", "11.655")

	if got := priceOf(t, book, month(2021, time.February)).Deregulated; got == nil ||
		got.StringFixed(domain.Precision) != "12.304" {
		t.Errorf("2021-02 deregulated = %v, want 12.304", got)
	}

	checkTimes(t, book.Modified, book.ReadAt)

	files := recorded()
	files[balancingName] = ok("balancing-empty-cell.xlsx")

	book, err = serve(t, files).Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() with an empty cell error = %v", err)
	}

	checkPrices(t, book.Months[len(book.Months)-1], "none", "1.607", "11.655")
}
