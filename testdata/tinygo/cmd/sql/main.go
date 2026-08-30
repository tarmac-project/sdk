package main

import (
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/sql"
)

func main() {
	client, err := sql.New(sql.Config{})
	if err != nil {
		panic(err)
	}

	_, err = sdk.New(sdk.Config{
		Handler: func(input []byte) ([]byte, error) {
			result, callErr := client.Query(string(input))
			return result.Data, callErr
		},
	})
	if err != nil {
		panic(err)
	}
}
