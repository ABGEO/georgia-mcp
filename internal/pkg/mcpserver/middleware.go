package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const toolErrorType = "tool_error"

func logToolCalls(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			call, ok := req.(*mcp.CallToolRequest)
			if !ok {
				return next(ctx, method, req)
			}

			start := time.Now()
			result, err := next(ctx, method, req)

			attrs := []slog.Attr{
				slog.String("mcp.method.name", method),
				slog.String("gen_ai.tool.name", call.Params.Name),
				slog.Duration("event.duration", time.Since(start)),
			}

			if logger.Enabled(ctx, slog.LevelDebug) {
				arguments := string(call.Params.Arguments)
				attrs = append(attrs, slog.String("mcp.tool.arguments", arguments))
			}

			if errorType := errorTypeOf(result, err); errorType != "" {
				attrs = append(attrs, slog.String("error.type", errorType))
			}

			logger.LogAttrs(ctx, slog.LevelInfo, "tool call", attrs...)

			return result, err
		}
	}
}

func errorTypeOf(result mcp.Result, err error) string {
	if err != nil {
		return fmt.Sprintf("%T", err)
	}

	if toolResult, ok := result.(*mcp.CallToolResult); ok && toolResult.IsError {
		return toolErrorType
	}

	return ""
}
