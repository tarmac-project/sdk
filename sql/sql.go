package sql

import (
	"errors"
	"fmt"
	"strings"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/sql"
	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
)

const (
	capabilityName = "sql"
	fnExec         = "exec"
	fnQuery        = "query"

	hostStatusOK       = int32(200)
	hostStatusPartial  = int32(206)
	hostStatusBadInput = int32(400)
	hostStatusMissing  = int32(404)
	hostStatusError    = int32(500)
)

var (
	// ErrInvalidQuery indicates an empty or invalid SQL query.
	ErrInvalidQuery = errors.New("query is invalid")

	// ErrPartialResult indicates the host returned a partial result.
	ErrPartialResult = errors.New("operation completed with partial result")

	// ErrMarshalRequest wraps failures while encoding the request payload.
	ErrMarshalRequest = errors.New("failed to marshal request")

	// ErrUnmarshalResponse wraps failures while decoding the host response.
	ErrUnmarshalResponse = errors.New("failed to unmarshal response")
)

// PartialResultError indicates an operation completed with degraded metadata and
// includes the underlying cause reported by the host.
type PartialResultError struct {
	Operation string
	Cause     error
}

// Error returns a human-readable partial-result message.
func (e *PartialResultError) Error() string {
	if e == nil {
		return ErrPartialResult.Error()
	}

	op := e.Operation
	if op == "" {
		op = "sql operation"
	}

	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", op, ErrPartialResult, e.Cause)
	}

	return fmt.Sprintf("%s: %s", op, ErrPartialResult)
}

// Unwrap exposes both ErrPartialResult and the underlying cause to errors.Is/As.
func (e *PartialResultError) Unwrap() []error {
	if e == nil {
		return []error{ErrPartialResult}
	}
	if e.Cause != nil {
		return []error{ErrPartialResult, e.Cause}
	}
	return []error{ErrPartialResult}
}

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

// DBClient is the SQL capability client implementation.
type DBClient struct {
	runtime  sdk.RuntimeConfig
	hostCall HostCall
}

// New creates a SQL client. Implementation will follow in future iterations.
func New(config Config) (*DBClient, error) {
	runtime := config.SDKConfig
	if runtime.Namespace == "" {
		runtime.Namespace = sdk.DefaultNamespace
	}

	hostCall := config.HostCall
	if hostCall == nil {
		hostCall = wapc.HostCall
	}

	return &DBClient{runtime: runtime, hostCall: hostCall}, nil
}

// Exec executes a SQL statement that does not return rows.
func (c *DBClient) Exec(query string) (ExecResult, error) {
	if strings.TrimSpace(query) == "" {
		return ExecResult{}, ErrInvalidQuery
	}

	req := &proto.SQLExec{Query: []byte(query)}
	b, err := req.MarshalVT()
	if err != nil {
		return ExecResult{}, errors.Join(ErrMarshalRequest, err)
	}

	respBytes, callErr := c.hostCall(c.runtime.Namespace, capabilityName, fnExec, b)
	if callErr != nil && len(respBytes) == 0 {
		return ExecResult{}, errors.Join(sdk.ErrHostCall, callErr)
	}

	var resp proto.SQLExecResponse
	if unmarshalErr := resp.UnmarshalVT(respBytes); unmarshalErr != nil {
		if callErr != nil {
			return ExecResult{}, errors.Join(
				sdk.ErrHostCall,
				callErr,
				sdk.ErrHostResponseInvalid,
				ErrUnmarshalResponse,
				unmarshalErr,
			)
		}
		return ExecResult{}, errors.Join(sdk.ErrHostResponseInvalid, ErrUnmarshalResponse, unmarshalErr)
	}

	result := ExecResult{
		LastInsertID: resp.GetLastInsertId(),
		RowsAffected: resp.GetRowsAffected(),
	}

	if statusErr := validateStatus(resp.GetStatus(), callErr, fnExec); statusErr != nil {
		var partialErr *PartialResultError
		if errors.As(statusErr, &partialErr) {
			return result, statusErr
		}
		return ExecResult{}, statusErr
	}

	return result, nil
}

// Query executes a SQL statement that returns rows.
func (c *DBClient) Query(query string) (QueryResult, error) {
	if strings.TrimSpace(query) == "" {
		return QueryResult{}, ErrInvalidQuery
	}

	req := &proto.SQLQuery{Query: []byte(query)}
	b, err := req.MarshalVT()
	if err != nil {
		return QueryResult{}, errors.Join(ErrMarshalRequest, err)
	}

	respBytes, callErr := c.hostCall(c.runtime.Namespace, capabilityName, fnQuery, b)
	if callErr != nil && len(respBytes) == 0 {
		return QueryResult{}, errors.Join(sdk.ErrHostCall, callErr)
	}

	var resp proto.SQLQueryResponse
	if unmarshalErr := resp.UnmarshalVT(respBytes); unmarshalErr != nil {
		if callErr != nil {
			return QueryResult{}, errors.Join(
				sdk.ErrHostCall,
				callErr,
				sdk.ErrHostResponseInvalid,
				ErrUnmarshalResponse,
				unmarshalErr,
			)
		}
		return QueryResult{}, errors.Join(sdk.ErrHostResponseInvalid, ErrUnmarshalResponse, unmarshalErr)
	}

	result := QueryResult{
		Columns: resp.GetColumns(),
		Data:    resp.GetData(),
	}

	if statusErr := validateStatus(resp.GetStatus(), callErr, fnQuery); statusErr != nil {
		var partialErr *PartialResultError
		if errors.As(statusErr, &partialErr) {
			return result, statusErr
		}
		return QueryResult{}, statusErr
	}

	return result, nil
}

// Close releases resources held by the client.
func (c *DBClient) Close() error {
	_ = c
	return nil
}

func validateStatus(status *sdkproto.Status, callErr error, operation string) error {
	if status == nil {
		if callErr != nil {
			return errors.Join(sdk.ErrHostCall, callErr, sdk.ErrHostResponseInvalid)
		}
		return sdk.ErrHostResponseInvalid
	}

	code := status.GetCode()
	switch code {
	case hostStatusOK:
		return nil
	case hostStatusPartial:
		cause := callErr
		if cause == nil && status.GetStatus() != "" {
			cause = errors.New(status.GetStatus())
		}
		return &PartialResultError{
			Operation: operation,
			Cause:     cause,
		}
	case hostStatusBadInput, hostStatusMissing, hostStatusError:
		cause := error(nil)
		if msg := status.GetStatus(); msg != "" {
			cause = errors.New(msg)
		}
		if cause == nil {
			cause = errors.New("host returned an error status")
		}
		return &sdk.HostStatusError{
			Capability:  capabilityName,
			Operation:   operation,
			Cause:       cause,
			HostCallErr: callErr,
		}
	default:
		statusErr := fmt.Errorf("unexpected host status code %d", code)
		if callErr != nil {
			return errors.Join(sdk.ErrHostCall, callErr, sdk.ErrHostResponseInvalid, statusErr)
		}
		return errors.Join(sdk.ErrHostResponseInvalid, statusErr)
	}
}
