package ctx

import (
	"context"

	"github.com/jmoiron/sqlx" // https://jmoiron.github.io/sqlx/
)

// The key type for Context value
type key int

// requestIDKey is the context key for the Request ID.  Its value of zero is
// arbitrary.  If this package defined other context keys, they would have
// different integer values.
const requestIDKey key = 0
const txKey key = 1
const sqlKey key = 2

// NewContextRequestID returns a new Context carrying RequestID.
func NewContextRequestID(ctx context.Context, requestID uint64) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// FromContextRequestID extracts the RequestID from ctx, if present.
func FromContextRequestID(ctx context.Context) uint64 {
	// ctx.Value returns nil if ctx has no value for the key;
	// the uint64 type assertion returns ok=false for nil.
	requestID, ok := ctx.Value(requestIDKey).(uint64)
	if !ok {
		return 0
	}
	return requestID
}

// FromContextTx extracts the Tx from ctx, if present.
func FromContextTx(ctx context.Context) *sqlx.Tx {
	tx, ok := ctx.Value(txKey).(*sqlx.Tx)
	if !ok {
		return nil
	}
	return tx
}

// NewContextTx returns a new Context carrying Tx.
func NewContextTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// FromContextSQLId extracts the SQL Id from ctx, if present
func FromContextSQLId(ctx context.Context) uint64 {
	sqlID, ok := ctx.Value(sqlKey).(uint64)
	if !ok {
		return 0
	}
	return sqlID
}

// NewContextSQLId returns a new Context carrying SQL Id
func NewContextSQLId(ctx context.Context, sqlID uint64) context.Context {
	return context.WithValue(ctx, sqlKey, sqlID)
}
