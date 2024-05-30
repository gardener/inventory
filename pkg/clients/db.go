package clients

import "github.com/uptrace/bun"

var Db *bun.DB

// SetDB sets the database connection to be used by the workers.
func SetDB(database *bun.DB) {
	Db = database
}
