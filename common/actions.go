package main

import (
	"context"
	"slices"

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
	entries, _ := mod.Entries(ctx)

	return dag.Directory().
		WithDirectory(".github/workflows",
			dag.Directory().
				With(func(dir *dagger.Directory) *dagger.Directory {
					const name = `workflows`
					if slices.Contains(entries, name) {
						return dir.WithDirectory(".", mod.Directory(name))
					}
					return dir
				}).
				With(func(dir *dagger.Directory) *dagger.Directory {
					const name = `hdall`
					if slices.Contains(entries, name) {
						return dir.WithDirectory(".", m.CompileDhall(ctx,
							mod.Directory("dhall")))
					}
					return dir
				}))
}
