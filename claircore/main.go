// Module for Claircore functions

package main

import (
	"context"

	"github.com/quay/clair-workflows/claircore/internal/dagger"
)

// Claircore holds actions for the claircore repo.
type Claircore struct{}

// Test ...
func (m *Claircore) Test(
	ctx context.Context,
	// Source to use for testing.
	//
	// If omitted, the `main` branch of the [upstream repository] will be used.
	//
	// [upstream repository]: https://github.com/quay/claircore
	//
	//+ignore=[".git"]
	//+optional
	source *dagger.Directory,
) (string, error) {
	if source == nil {
		source = dag.Git("https://github.com/quay/claircore").Branch("main").Tree()
	}
	opts := dagger.CommonTestOpts{
		Race:     false,
		Cover:    false,
		Unit:     true, // TODO
		Database: nil,
	}
	return dag.Common().Test(ctx, source, opts)
}
