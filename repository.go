package sqltoolbox

import (
	"github.com/davidroman0O/sqlite-toolbox/data"
)

type DatabaseConnector interface {
	Open(middlewareManager *data.MiddlewareManager) (*data.MuxDb, error)
	Close() error
}
