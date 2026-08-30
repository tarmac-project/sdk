package main

import (
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/metrics"
)

func main() {
	client, err := metrics.New(metrics.Config{})
	if err != nil {
		panic(err)
	}
	counter, err := client.NewCounter("fixture_calls")
	if err != nil {
		panic(err)
	}

	_, err = sdk.New(sdk.Config{
		Handler: func(input []byte) ([]byte, error) {
			counter.Inc()
			return input, nil
		},
	})
	if err != nil {
		panic(err)
	}
}
