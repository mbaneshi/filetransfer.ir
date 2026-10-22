package migrations

import _ "embed"

//go:embed 0001_users.sql
var Users string

//go:embed 0002_files.sql
var Files string
