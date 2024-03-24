package sqlitetoolbox

import "database/sql"

type DoFn func(db *sql.DB) error
type PassDoFn func(cb DoFn) error
