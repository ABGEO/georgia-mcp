package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const toolErrorType = "tool_error"

// failureKey holds the slot of a tool call in its context. The SDK turns the error of a
// handler into a result, so the wrapper of AddTool hands the error to the middleware
// through the slot.
type failureKey struct{}

type failure struct {
	err error
}

func recordFailure(ctx context.Context, err error) {
	if slot, ok := ctx.Value(failureKey{}).(*failure); ok {
		slot.err = err
	}
}

func logToolCalls(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			call, ok := req.(*mcp.CallToolRequest)
			if !ok {
				return next(ctx, method, req)
			}

			slot := &failure{}
			ctx = context.WithValue(ctx, failureKey{}, slot)

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

			level := slog.LevelInfo
			if errors.Is(slot.err, kernel.ErrSourceUnavailable) {
				level = slog.LevelWarn

				attrs = append(attrs, slog.Any("error", causeOf(slot.err)))
			}

			logger.LogAttrs(ctx, level, "tool call", attrs...)

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

// causeOf returns the cause of a ToolError, which the model does not read, else err.
func causeOf(err error) error {
	var toolErr *ToolError
	if errors.As(err, &toolErr) && toolErr.Cause != nil {
		return toolErr.Cause
	}

	return err
}
