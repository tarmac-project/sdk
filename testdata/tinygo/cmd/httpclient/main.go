package main

import (
	"io"

	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/httpclient"
)

func main() {
	client, err := httpclient.New(httpclient.Config{})
	if err != nil {
		panic(err)
	}

	_, err = sdk.New(sdk.Config{
		Handler: func(input []byte) ([]byte, error) {
			resp, callErr := client.Get(string(input))
			if callErr != nil || resp.Body == nil {
				return nil, callErr
			}
			defer resp.Body.Close()
			return io.ReadAll(resp.Body)
		},
	})
	if err != nil {
		panic(err)
	}
}
