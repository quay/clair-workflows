package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/quay/clair-workflows/cmd/housestyle/internal/bareprints"
)

func main() {
	multichecker.Main(bareprints.Analyzer)
}
