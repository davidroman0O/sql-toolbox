package rows

import (
	"database/sql/driver"
	"io"
)

type rowMapConfig struct {
	cols []string
}

type rowOptions func(*rowMapConfig)

func WithColumns(cols []string) rowOptions {
	return func(c *rowMapConfig) {
		c.cols = cols
	}
}

func GetRowsMap(iterate func(dest []driver.Value) error, opts ...rowOptions) ([]map[string]interface{}, error) {
	config := rowMapConfig{
		cols: []string{},
	}
	for i := 0; i < len(opts); i++ {
		opts[i](&config)
	}
	rows := []map[string]interface{}{}
	dest := make([]driver.Value, len(config.cols))
	for {
		err := iterate(dest)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		// fmt.Println("loop", len(dest))

		row := map[string]interface{}{}
		// pp.Println(dest)
		for i := 0; i < len(config.cols); i++ {
			row[config.cols[i]] = dest[i]
		}
		rows = append(rows, row)
	}
	return rows, nil
}
