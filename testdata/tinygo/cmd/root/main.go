package main

import sdk "github.com/tarmac-project/sdk"

func main() {
	_, err := sdk.New(sdk.Config{
		Handler: func(input []byte) ([]byte, error) {
			return input, nil
		},
	})
	if err != nil {
		panic(err)
	}
}
