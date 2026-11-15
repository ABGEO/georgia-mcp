package tool_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

func discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

// staleSource answers each method with a stale value, as the cache does while the source
// fails.
type staleSource struct {
	act domain.Act
}

func (s staleSource) Search(context.Context, domain.SearchQuery) (app.SearchPage, error) {
	return app.SearchPage{
		Acts: []domain.ActSummary{{ID: s.act.ID, Title: s.act.Title}}, Total: 1,
		ReadAt: clock(), Stale: true,
	}, nil
}

func (s staleSource) Titles(context.Context, string) (app.TitleList, error) {
	return app.TitleList{ReadAt: clock(), Stale: true}, nil
}

func (s staleSource) Act(
	context.Context,
	domain.ActID,
	*int,
	kernel.Language,
) (app.ActText, error) {
	return app.ActText{Act: s.act, ReadAt: clock(), Stale: true}, nil
}

func (s staleSource) Status(context.Context, string) (app.StatusResult, error) {
	return app.StatusResult{ReadAt: clock(), Stale: true}, nil
}

func connect(t *testing.T, source app.ActSource) *mcp.ClientSession {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, tool.UseCases{
		SearchActs: app.NewSearchActs(source, discard()),
		GetAct:     app.NewGetAct(source, discard()),
	}); err != nil {
		t.Fatal(err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = registry.RunStdio(ctx, serverTransport) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).
		Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

// MATSNE-SC-029: A stale answer of the cache reaches the model with stale true.
func TestAStaleAnswerSaysSo(t *testing.T) {
	t.Parallel()

	id, err := domain.ParseActID("1043717")
	if err != nil {
		t.Fatal(err)
	}

	session := connect(t, staleSource{act: domain.Act{
		ID: id, Title: "TAX CODE OF GEORGIA",
		Version: domain.Version{Number: 247}, Versions: []domain.Version{{Number: 247}},
	}})

	for name, args := range map[string]map[string]any{
		"law_search_acts": {"query": "Tax Code"},
		"law_get_act":     {"id": "1043717"},
	} {
		result, err := session.CallTool(
			t.Context(),
			&mcp.CallToolParams{Name: name, Arguments: args},
		)
		if err != nil || result.IsError {
			t.Fatalf("%s = %+v, %v, want a result", name, result, err)
		}

		raw, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}

		var out struct {
			Stale bool   `json:"stale"`
			AsOf  string `json:"as_of"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}

		if !out.Stale || out.AsOf != "2026-11-15T12:00:00+04:00" {
			t.Errorf(
				"%s = stale %v at %q, want stale at the read instant",
				name,
				out.Stale,
				out.AsOf,
			)
		}
	}
}
