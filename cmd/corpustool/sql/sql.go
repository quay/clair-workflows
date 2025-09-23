package sql

import (
	"embed"
)

//go:generate find . -name *.sql -exec go run github.com/wasilibs/go-sql-formatter/v15/cmd/sql-formatter@latest --language sqlite --fix {} ;

//go:embed *.sql
var FS embed.FS
