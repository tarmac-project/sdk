package function

import ()

type Function interface {}

type functionClient struct {}

type Config struct {}

func New(config *Config) (*functionClient, error) {
  return &functionClient{}, nil
}

func (c *functionClient) Call() error {
  return nil
}
