package main

import (
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/function"
)

func main() {
	client, err := function.New(function.Config{})
	if err != nil {
		panic(err)
	}

	_, err = sdk.New(sdk.Config{
		Handler: func(input []byte) ([]byte, error) {
			return client.Call("fixture", input)
		},
	})
	if err != nil {
		panic(err)
	}
}
