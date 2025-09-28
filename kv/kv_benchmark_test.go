package kv

import (
	"testing"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/kvstore"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
	pb "google.golang.org/protobuf/proto"
)

func BenchmarkKVClient(b *testing.B) {
	const namespace = "benchmark"
	const capability = "kvstore"

	// Pre-marshal a happy-path GET response
	getResp := func() []byte {
		resp := &proto.KVStoreGetResponse{
			Status: &sdkproto.Status{Status: "OK", Code: 0},
			Data:   []byte("value"),
		}
		bz, _ := pb.Marshal(resp)
		return bz
	}
	mockGet, _ := hostmock.New(hostmock.Config{
		ExpectedNamespace:  namespace,
		ExpectedCapability: capability,
		ExpectedFunction:   "get",
		Response:           getResp,
	})
	clientGet, _ := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: namespace}, HostCall: mockGet.HostCall})

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			if _, err := clientGet.Get("benchmark-key"); err != nil {
				b.Fatalf("Get failed: %v", err)
			}
		}
	})

	// Pre-marshal a happy-path SET response
	setResp := func() []byte {
		resp := &proto.KVStoreSetResponse{Status: &sdkproto.Status{Status: "OK", Code: 0}}
		bz, _ := pb.Marshal(resp)
		return bz
	}
	mockSet, _ := hostmock.New(hostmock.Config{
		ExpectedNamespace:  namespace,
		ExpectedCapability: capability,
		ExpectedFunction:   "set",
		Response:           setResp,
	})
	clientSet, _ := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: namespace}, HostCall: mockSet.HostCall})

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			if err := clientSet.Set("benchmark-key", []byte("value")); err != nil {
				b.Fatalf("Set failed: %v", err)
			}
		}
	})

	// Pre-marshal a happy-path DELETE response
	delResp := func() []byte {
		resp := &proto.KVStoreDeleteResponse{Status: &sdkproto.Status{Status: "OK", Code: 0}}
		bz, _ := pb.Marshal(resp)
		return bz
	}
	mockDel, _ := hostmock.New(hostmock.Config{
		ExpectedNamespace:  namespace,
		ExpectedCapability: capability,
		ExpectedFunction:   "delete",
		Response:           delResp,
	})
	clientDel, _ := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: namespace}, HostCall: mockDel.HostCall})

	b.Run("Delete", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			if err := clientDel.Delete("benchmark-key"); err != nil {
				b.Fatalf("Delete failed: %v", err)
			}
		}
	})

	// Pre-marshal a happy-path KEYS response
	keysResp := func() []byte {
		resp := &proto.KVStoreKeysResponse{
			Status: &sdkproto.Status{Status: "OK", Code: 0},
			Keys:   []string{"a", "b", "c"},
		}
		bz, _ := pb.Marshal(resp)
		return bz
	}
	mockKeys, _ := hostmock.New(hostmock.Config{
		ExpectedNamespace:  namespace,
		ExpectedCapability: capability,
		ExpectedFunction:   "keys",
		Response:           keysResp,
	})
	clientKeys, _ := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: namespace}, HostCall: mockKeys.HostCall})

	b.Run("Keys", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			if _, err := clientKeys.Keys(); err != nil {
				b.Fatalf("Keys failed: %v", err)
			}
		}
	})
}
