package main

import (
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/logging"
)

func main() {
	logger, err := logging.New(logging.Config{})
	if err != nil {
		panic(err)
	}

	_, err = sdk.New(sdk.Config{
		Handler: func(input []byte) ([]byte, error) {
			logger.Info(string(input))
			return input, nil
		},
	})
	if err != nil {
		panic(err)
	}
}
