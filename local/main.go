package main

import (
	"context"

	"github.com/quay/clair-workflows/local/internal/dagger"
)

type Local struct{}

func (m *Local) Workflows(ctx context.Context) *dagger.Directory {
	return dag.Common().EmbeddedWorkflows(dag.CurrentModule().Source())
}
