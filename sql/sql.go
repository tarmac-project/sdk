package sql

import (
	"errors"

	sdk "github.com/tarmac-project/sdk"
)

var (
	// ErrNotImplemented signals that the SQL client is not implemented yet.
	ErrNotImplemented = errors.New("sql client not implemented")
)

// HostCall defines the waPC host function signature used by SQL operations.
type HostCall func(string, string, string, []byte) ([]byte, error)

// Client defines the SQL capability interface.
type Client interface {
	// Exec executes a SQL statement that does not return rows.
	Exec(query string) (ExecResult, error)

	// Query executes a SQL statement that returns rows.
	Query(query string) (QueryResult, error)

	// Close releases resources held by the client.
	Close() error
}

// Config controls how a Client instance interacts with the host runtime.
type Config struct {
	// SDKConfig provides the runtime namespace used for host calls.
	SDKConfig sdk.RuntimeConfig

	// HostCall overrides the waPC host function used for SQL operations.
	HostCall HostCall
}

// ExecResult mirrors the SQLExecResponse payload fields.
type ExecResult struct {
	// LastInsertID is the ID of the last inserted row, when available.
	LastInsertID int64
	// RowsAffected is the number of rows affected by the statement.
	RowsAffected int64
}

// QueryResult mirrors the SQLQueryResponse payload fields.
type QueryResult struct {
	// Columns are the column names returned by the query.
	Columns []string
	// Data is a JSON-encoded byte slice of the query result data.
	Data []byte
}

// client is a placeholder implementation for the SQL capability client.
type client struct {
	cfg Config
}

// New creates a SQL client. Implementation will follow in future iterations.
func New(config Config) (Client, error) {
	_ = Client(&client{})
	return &client{cfg: config}, nil
}

// Exec executes a SQL statement that does not return rows.
func (c *client) Exec(query string) (ExecResult, error) {
	_ = c
	_ = query
	return ExecResult{}, ErrNotImplemented
}

// Query executes a SQL statement that returns rows.
func (c *client) Query(query string) (QueryResult, error) {
	_ = c
	_ = query
	return QueryResult{}, ErrNotImplemented
}

// Close releases resources held by the client.
func (c *client) Close() error {
	_ = c
	return ErrNotImplemented
}
