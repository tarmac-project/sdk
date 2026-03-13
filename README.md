# Tarmac SDK (Go) 🛠️

**A tiny, testable Go SDK for Tarmac WebAssembly functions.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/tarmac-project/sdk)](https://github.com/tarmac-project/sdk)
[![Go Reference](https://pkg.go.dev/badge/github.com/tarmac-project/sdk.svg)](https://pkg.go.dev/github.com/tarmac-project/sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/tarmac-project/sdk)](https://goreportcard.com/report/github.com/tarmac-project/sdk)
[![Tests](https://github.com/tarmac-project/sdk/actions/workflows/tests.yml/badge.svg)](https://github.com/tarmac-project/sdk/actions/workflows/tests.yml)
[![Lint](https://github.com/tarmac-project/sdk/actions/workflows/lint.yml/badge.svg)](https://github.com/tarmac-project/sdk/actions/workflows/lint.yml)
[![Codecov](https://codecov.io/gh/tarmac-project/sdk/branch/main/graph/badge.svg)](https://codecov.io/gh/tarmac-project/sdk)

---

## 🧠 What is Tarmac SDK (Go)?

A minimal Go library for building Tarmac guest functions.

A key feature of this SDK is its testability.
You can use the included mock clients or create your own.
You can also use the low-level hostmock to simulate hostcalls to Tarmac if you want to get really deep into testing.

---

## 🚀 Getting Started

### Prerequisites

- Go 1.23 or newer
- Protobuf support ships with the module. Generated clients currently use
  `protoc-gen-go-lite` and expose helpers such as `MarshalVT`.

Register your handler with the SDK:

```go
package main

import (
  "github.com/tarmac-project/sdk"
)

func main() {
  _, err := sdk.New(sdk.Config{
    Namespace: "tarmac", // optional; defaults to "tarmac"
    Handler: func(b []byte) ([]byte, error) {
      // Your function logic here
      return b, nil
    },
  })
  if err != nil {
    panic(err)
  }
}
```

---

## 🧱 Structure

The project is organized into focused modules so you can depend only on what you need.

| Module/Path   | Description                                        | Docs                                                      |
| ------------- | -------------------------------------------------- | --------------------------------------------------------- |
| `sdk`         | Core runtime config and handler registration  | <https://pkg.go.dev/github.com/tarmac-project/sdk>          |
| `sdk/httpclient`    | HTTP client      | <https://pkg.go.dev/github.com/tarmac-project/sdk/httpclient>     |
| `sdk/function`      | Function-to-function callback client | <https://pkg.go.dev/github.com/tarmac-project/sdk/function> |
| `sdk/kv`      | Key-value client | <https://pkg.go.dev/github.com/tarmac-project/sdk/kv>       |
| `sdk/metrics` | Metrics client | <https://pkg.go.dev/github.com/tarmac-project/sdk/metrics> |
| `sdk/sql`      | SQL client | <https://pkg.go.dev/github.com/tarmac-project/sdk/sql>       |
| `sdk/hostmock` | Low-level host-call simulator for assertions | <https://pkg.go.dev/github.com/tarmac-project/sdk/hostmock> |
| `sdk/logging` | Logging client | <https://pkg.go.dev/github.com/tarmac-project/sdk/logging> |

---

## 🤝 Contributing

PRs welcome! Please open an issue to discuss changes.

---

## 📄 License

Apache-2.0 — see [LICENSE](LICENSE).

---

## 🌴 Stay Tiny, Ship Fast!

Questions, ideas, or rough edges? Open an issue or PR — we’d love to hear from you.
