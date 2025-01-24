package main

import (
	"context"

	"github.com/quay/clair-workflows/common/internal/dagger"
)

func (m *Common) Actions(ctx context.Context) *dagger.Directory {
	return dag.
		Gha().
		WithWorkflow(
			dag.Gha().Workflow("Test Suite", dagger.GhaWorkflowOpts{
				PullRequestConcurrency:      "preempt",
				OnPullRequestReadyForReview: true,
			}).WithJob(
				dag.Gha().Job("test", "test", dagger.GhaJobOpts{
					Runner: []string{"ubuntu-latest"},
					Debug:  true,
					Module: "https://github.com/quay/clair-workflows/claircore@dagger",
				}))).
		Generate()
}

func (m *Common) EmbeddedWorkflows(
	ctx context.Context,
	mod *dagger.Directory,
) *dagger.Directory {
	return dag.Directory().
		WithDirectory(".github/workflows",
			dag.Directory().
				With(func(dir *dagger.Directory) *dagger.Directory {
					embedded := mod.Directory("workflows")
					if embedded == nil {
						return dir
					}
					return dir.WithDirectory(".", embedded)
				}).
				With(func(dir *dagger.Directory) *dagger.Directory {
					dhall := mod.Directory("dhall")
					if dhall == nil {
						return dir
					}
					return dir.WithDirectory(".", m.CompileDhall(ctx, dhall))
				}))
}
