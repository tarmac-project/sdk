package main

import (
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/kv"
)

func main() {
	client, err := kv.New(kv.Config{})
	if err != nil {
		panic(err)
	}

	_, err = sdk.New(sdk.Config{
		Handler: func(input []byte) ([]byte, error) {
			if err := client.Set("fixture", input); err != nil {
				return nil, err
			}
			return client.Get("fixture")
		},
	})
	if err != nil {
		panic(err)
	}
}
