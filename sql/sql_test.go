package sql

import (
	"errors"
	"testing"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/sql"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
)

func TestNew_DefaultNamespace(t *testing.T) {
	t.Parallel()

	cfg := hostmock.Config{
		ExpectedNamespace:  sdk.DefaultNamespace,
		ExpectedCapability: capabilityName,
		ExpectedFunction:   fnExec,
		PayloadValidator: func(payload []byte) error {
			var req proto.SQLExec
			return req.UnmarshalVT(payload)
		},
		Response: func() []byte {
			resp := &proto.SQLExecResponse{
				Status:       &sdkproto.Status{Status: "OK", Code: 200},
				LastInsertId: 1,
				RowsAffected: 1,
			}
			b, _ := resp.MarshalVT()
			return b
		},
	}

	mock, err := hostmock.New(cfg)
	if err != nil {
		t.Fatalf("hostmock: %v", err)
	}

	client, err := New(Config{HostCall: mock.HostCall})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.Exec("SELECT 1")
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
}

func TestExec_HappyPath(t *testing.T) {
	t.Parallel()

	want := ExecResult{LastInsertID: 42, RowsAffected: 3}
	query := "INSERT INTO table_name (col) VALUES (1)"

	cfg := hostmock.Config{
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
			resp := &proto.SQLExecResponse{
				Status:       &sdkproto.Status{Status: "OK", Code: 200},
				LastInsertId: want.LastInsertID,
				RowsAffected: want.RowsAffected,
			}
			b, _ := resp.MarshalVT()
			return b
		},
	}

	mock, err := hostmock.New(cfg)
	if err != nil {
		t.Fatalf("hostmock: %v", err)
	}

	client, err := New(Config{
		SDKConfig: sdk.RuntimeConfig{Namespace: "tarmac"},
		HostCall:  mock.HostCall,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got, err := client.Exec(query)
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if got != want {
		t.Fatalf("Exec result mismatch: want %+v got %+v", want, got)
	}
}

func TestQuery_HappyPath(t *testing.T) {
	t.Parallel()

	want := QueryResult{
		Columns: []string{"id", "name"},
		Data:    []byte(`[{"id":1,"name":"alpha"}]`),
	}
	query := "SELECT id, name FROM table_name"

	cfg := hostmock.Config{
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
			resp := &proto.SQLQueryResponse{
				Status:  &sdkproto.Status{Status: "OK", Code: 200},
				Columns: want.Columns,
				Data:    want.Data,
			}
			b, _ := resp.MarshalVT()
			return b
		},
	}

	mock, err := hostmock.New(cfg)
	if err != nil {
		t.Fatalf("hostmock: %v", err)
	}

	client, err := New(Config{
		SDKConfig: sdk.RuntimeConfig{Namespace: "tarmac"},
		HostCall:  mock.HostCall,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got, err := client.Query(query)
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(got.Columns) != len(want.Columns) || string(got.Data) != string(want.Data) {
		t.Fatalf("Query result mismatch: want %+v got %+v", want, got)
	}
}
