package sql

type SQL interface{}

type Config struct {
	Namespace string
	Database  string
}

func New(config Config) (SQL, error) {
	return &sqlClient{}, nil
}

type sqlClient struct{}

func (c *sqlClient) Query(query string) ([]map[string]any, error) {
	return nil, nil
}

type Rows struct {
	Columns []string
	Values  []Row
}

type Row struct {
	Values map[string]any
}

type Result struct{}

func (c *sqlClient) Exec(query string) (Result, error) {
	return nil, nil
}
