package wallet

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

type faucetLoggerKey struct{}

// WithFaucetLogger correlates HTTP and service events without logging request secrets.
func WithFaucetLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, faucetLoggerKey{}, logger)
}

func faucetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(faucetLoggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default().With("component", "solana_faucet")
}

// Deliberately omit error text, RPC data, URLs and bodies: providers may echo credentials.
func faucetErrorFields(err error) []any {
	fields := []any{"error_type", fmt.Sprintf("%T", err)}
	var httpErr *jsonrpc.HTTPError
	var rpcErr *jsonrpc.RPCError
	var netErr net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		fields = append(fields, "error_class", "deadline_exceeded")
	case errors.Is(err, context.Canceled):
		fields = append(fields, "error_class", "canceled")
	case errors.As(err, &httpErr):
		fields = append(fields, "error_class", "http", "upstream_http_status", httpErr.Code)
	case errors.As(err, &rpcErr):
		fields = append(fields, "error_class", "rpc", "upstream_rpc_code", rpcErr.Code)
	case errors.As(err, &netErr) && netErr.Timeout():
		fields = append(fields, "error_class", "timeout")
	default:
		fields = append(fields, "error_class", "other")
	}
	return fields
}
