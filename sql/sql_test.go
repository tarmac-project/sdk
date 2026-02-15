package sql

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/sql"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
)

func TestExec_Table(t *testing.T) {
	t.Parallel()

	query := "INSERT INTO table_name (col) VALUES (1)"
	want := ExecResult{LastInsertID: 42, RowsAffected: 3}

	tt := []struct {
		name               string
		namespace          string
		query              string
		hostCfg            *hostmock.Config
		hostCall           HostCall
		want               ExecResult
		wantErr            error
		wantErrMsg         string
		checkResultOnError bool
	}{
		{
			name:  "Default Namespace",
			query: "SELECT 1",
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  sdk.DefaultNamespace,
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				PayloadValidator: func(payload []byte) error {
					var req proto.SQLExec
					return req.UnmarshalVT(payload)
				},
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "OK", Code: 200}, 1, 1)
				},
			},
			want: ExecResult{LastInsertID: 1, RowsAffected: 1},
		},
		{
			name:      "Happy Path",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				PayloadValidator: func(payload []byte) error {
					var req proto.SQLExec
					if err := req.UnmarshalVT(payload); err != nil {
						return err
					}
					if string(req.GetQuery()) != query {
						return errors.New("query mismatch")
					}
					return nil
				},
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "OK", Code: 200}, want.LastInsertID, want.RowsAffected)
				},
			},
			want: want,
		},
		{
			name:    "Empty Query",
			query:   "",
			wantErr: ErrInvalidQuery,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:    "Whitespace Query",
			query:   " \n\t ",
			wantErr: ErrInvalidQuery,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:      "Host Call Failure",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Fail:               true,
				Error:              errors.New("host call failed"),
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Host Call Error With Empty Response",
			namespace: "tarmac",
			query:     query,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return []byte{}, errors.New("host call failed")
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Invalid Response Payload",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return []byte("not-proto")
				},
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
		{
			name:      "Missing Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					resp := &proto.SQLExecResponse{}
					b, _ := resp.MarshalVT()
					return b
				},
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
		{
			name:      "Host Status Error",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "boom", Code: 500}, 0, 0)
				},
			},
			wantErr: sdk.ErrHostError,
		},
		{
			name:      "Host Status Error With Message",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "boom", Code: 500}, 0, 0)
				},
			},
			wantErr:    sdk.ErrHostError,
			wantErrMsg: "boom",
		},
		{
			name:      "Host Status Bad Input",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "bad", Code: 400}, 0, 0)
				},
			},
			wantErr: sdk.ErrHostError,
		},
		{
			name:      "Host Status Bad Input With Message",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "bad input", Code: 400}, 0, 0)
				},
			},
			wantErr:    sdk.ErrHostError,
			wantErrMsg: "bad input",
		},
		{
			name:      "Host Status Missing",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "missing", Code: 404}, 0, 0)
				},
			},
			wantErr: sdk.ErrHostError,
		},
		{
			name:      "Host Status Missing With Message",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "missing key", Code: 404}, 0, 0)
				},
			},
			wantErr:    sdk.ErrHostError,
			wantErrMsg: "missing key",
		},
		{
			name:      "Host Status Partial",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return execResponse(
						&sdkproto.Status{Status: "partial", Code: 206},
						want.LastInsertID,
						want.RowsAffected,
					)
				},
			},
			want:               want,
			wantErr:            ErrPartialResult,
			wantErrMsg:         "partial",
			checkResultOnError: true,
		},
		{
			name:      "Custom Namespace",
			namespace: "custom",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "custom",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "OK", Code: 200}, want.LastInsertID, want.RowsAffected)
				},
			},
			want: want,
		},
		{
			name:      "Host Status Unknown",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "wat", Code: 777}, 0, 0)
				},
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
		{
			name:      "Host Call Error With OK Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "OK", Code: 200}, want.LastInsertID, want.RowsAffected)
				},
			},
			want: want,
		},
		{
			name:      "Host Call Error With Error Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					return execResponse(&sdkproto.Status{Status: "boom", Code: 500}, 0, 0)
				},
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Host Call Error With Partial Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					return execResponse(
						&sdkproto.Status{Status: "rows affected unavailable", Code: 206},
						want.LastInsertID,
						want.RowsAffected,
					)
				},
			},
			want:               want,
			wantErr:            ErrPartialResult,
			wantErrMsg:         "host call failed",
			checkResultOnError: true,
		},
		{
			name:      "Host Call Error With Invalid Payload",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					return []byte("not-proto")
				},
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Host Call Error With Missing Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnExec,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					resp := &proto.SQLExecResponse{}
					b, _ := resp.MarshalVT()
					return b
				},
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Nil Response Without Error",
			namespace: "tarmac",
			query:     query,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return nil, nil
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
		{
			name:      "Empty Response Without Error",
			namespace: "tarmac",
			query:     query,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return []byte{}, nil
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := newClient(t, tc.namespace, tc.hostCfg, tc.hostCall)
			got, err := client.Exec(tc.query)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if tc.wantErrMsg != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErrMsg)) {
					t.Fatalf("expected error to contain %q, got %v", tc.wantErrMsg, err)
				}
				if tc.checkResultOnError && got != tc.want {
					t.Fatalf("Exec result mismatch: want %+v got %+v", tc.want, got)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("Exec result mismatch: want %+v got %+v", tc.want, got)
			}
		})
	}
}

func TestQuery_Table(t *testing.T) {
	t.Parallel()

	query := "SELECT id, name FROM table_name"
	want := QueryResult{
		Columns: []string{"id", "name"},
		Data:    []byte(`[{"id":1,"name":"alpha"}]`),
	}

	tt := []struct {
		name               string
		namespace          string
		query              string
		hostCfg            *hostmock.Config
		hostCall           HostCall
		want               QueryResult
		wantErr            error
		wantErrMsg         string
		checkResultOnError bool
	}{
		{
			name:      "Happy Path",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				PayloadValidator: func(payload []byte) error {
					var req proto.SQLQuery
					if err := req.UnmarshalVT(payload); err != nil {
						return err
					}
					if string(req.GetQuery()) != query {
						return errors.New("query mismatch")
					}
					return nil
				},
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "OK", Code: 200}, want.Columns, want.Data)
				},
			},
			want: want,
		},
		{
			name:    "Empty Query",
			query:   "",
			wantErr: ErrInvalidQuery,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:    "Whitespace Query",
			query:   " \n\t ",
			wantErr: ErrInvalidQuery,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:      "Host Call Failure",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Fail:               true,
				Error:              errors.New("host call failed"),
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Host Call Error With Empty Response",
			namespace: "tarmac",
			query:     query,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return []byte{}, errors.New("host call failed")
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Invalid Response Payload",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return []byte("not-proto")
				},
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
		{
			name:      "Missing Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					resp := &proto.SQLQueryResponse{}
					b, _ := resp.MarshalVT()
					return b
				},
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
		{
			name:      "Host Status Error",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "boom", Code: 500}, nil, nil)
				},
			},
			wantErr: sdk.ErrHostError,
		},
		{
			name:      "Host Status Error With Message",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "boom", Code: 500}, nil, nil)
				},
			},
			wantErr:    sdk.ErrHostError,
			wantErrMsg: "boom",
		},
		{
			name:      "Host Status Bad Input",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "bad", Code: 400}, nil, nil)
				},
			},
			wantErr: sdk.ErrHostError,
		},
		{
			name:      "Host Status Bad Input With Message",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "bad input", Code: 400}, nil, nil)
				},
			},
			wantErr:    sdk.ErrHostError,
			wantErrMsg: "bad input",
		},
		{
			name:      "Host Status Missing",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "missing", Code: 404}, nil, nil)
				},
			},
			wantErr: sdk.ErrHostError,
		},
		{
			name:      "Host Status Missing With Message",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "missing key", Code: 404}, nil, nil)
				},
			},
			wantErr:    sdk.ErrHostError,
			wantErrMsg: "missing key",
		},
		{
			name:      "Host Status Partial",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "partial", Code: 206}, want.Columns, want.Data)
				},
			},
			want:               want,
			wantErr:            ErrPartialResult,
			wantErrMsg:         "partial",
			checkResultOnError: true,
		},
		{
			name:      "Custom Namespace",
			namespace: "custom",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "custom",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "OK", Code: 200}, want.Columns, want.Data)
				},
			},
			want: want,
		},
		{
			name:      "Host Status Unknown",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "wat", Code: 777}, nil, nil)
				},
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
		{
			name:      "Host Call Error With OK Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "OK", Code: 200}, want.Columns, want.Data)
				},
			},
			want: want,
		},
		{
			name:      "Host Call Error With Error Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					return queryResponse(&sdkproto.Status{Status: "boom", Code: 500}, nil, nil)
				},
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Host Call Error With Partial Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					return queryResponse(
						&sdkproto.Status{Status: "partial rows", Code: 206},
						want.Columns,
						want.Data,
					)
				},
			},
			want:               want,
			wantErr:            ErrPartialResult,
			wantErrMsg:         "host call failed",
			checkResultOnError: true,
		},
		{
			name:      "Host Call Error With Invalid Payload",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					return []byte("not-proto")
				},
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Host Call Error With Missing Status",
			namespace: "tarmac",
			query:     query,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnQuery,
				Fail:               true,
				Error:              errors.New("host call failed"),
				Response: func() []byte {
					resp := &proto.SQLQueryResponse{}
					b, _ := resp.MarshalVT()
					return b
				},
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "Nil Response Without Error",
			namespace: "tarmac",
			query:     query,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return nil, nil
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
		{
			name:      "Empty Response Without Error",
			namespace: "tarmac",
			query:     query,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return []byte{}, nil
			},
			wantErr: sdk.ErrHostResponseInvalid,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := newClient(t, tc.namespace, tc.hostCfg, tc.hostCall)
			got, err := client.Query(tc.query)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if tc.wantErrMsg != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErrMsg)) {
					t.Fatalf("expected error to contain %q, got %v", tc.wantErrMsg, err)
				}
				if tc.checkResultOnError && !equalQueryResult(got, tc.want) {
					t.Fatalf("Query result mismatch: want %+v got %+v", tc.want, got)
				}
				return
			}
			if !equalQueryResult(got, tc.want) {
				t.Fatalf("Query result mismatch: want %+v got %+v", tc.want, got)
			}
		})
	}
}

func execResponse(status *sdkproto.Status, lastInsertID, rowsAffected int64) []byte {
	resp := &proto.SQLExecResponse{
		Status:       status,
		LastInsertId: lastInsertID,
		RowsAffected: rowsAffected,
	}
	b, _ := resp.MarshalVT()
	return b
}

func queryResponse(status *sdkproto.Status, columns []string, data []byte) []byte {
	resp := &proto.SQLQueryResponse{
		Status:  status,
		Columns: columns,
		Data:    data,
	}
	b, _ := resp.MarshalVT()
	return b
}

func newClient(t *testing.T, namespace string, cfg *hostmock.Config, hostCall HostCall) Client {
	t.Helper()

	switch {
	case cfg != nil:
		mock, err := hostmock.New(*cfg)
		if err != nil {
			t.Fatalf("hostmock: %v", err)
		}
		hostCall = mock.HostCall
	case hostCall == nil:
		hostCall = func(string, string, string, []byte) ([]byte, error) {
			t.Fatalf("unexpected host call")
			return nil, nil
		}
	}

	client, err := New(Config{
		SDKConfig: sdk.RuntimeConfig{Namespace: namespace},
		HostCall:  hostCall,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Fatalf("Close returned error: %v", closeErr)
		}
	})
	return client
}

func equalQueryResult(got, want QueryResult) bool {
	if len(got.Columns) != len(want.Columns) || len(got.Data) != len(want.Data) {
		return false
	}
	for i, col := range want.Columns {
		if got.Columns[i] != col {
			return false
		}
	}
	return bytes.EqualFold(got.Data, want.Data)
}
