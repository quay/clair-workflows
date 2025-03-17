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
	//+optional
	race bool,
	//+optional
	cover bool,
	//+optional
	fips bool,
) (string, error) {
	if source == nil {
		source = dag.Git("https://github.com/quay/claircore").Branch("main").Tree()
	}
	opts := dagger.CommonTestOpts{
		Race:     race,
		Cover:    cover,
		Unit:     true, // TODO
		Database: nil,
		Fips:     fips,
	}
	return dag.Common().Test(ctx, source, opts)
}

// Actions creates a [dagger.Directory] containing generated GitHub Actions
// workflows. Use the "export" command to output to the desired directory:
//
//	dagger call actions export --path=.
func (m *Claircore) Actions(ctx context.Context) *dagger.Directory {
	dir := dag.
		Gha().
		WithWorkflow(
			dag.Gha().Workflow("CI", dagger.GhaWorkflowOpts{
				PullRequestConcurrency:      "preempt",
				OnPushBranches:              []string{"main"},
				OnPullRequestBranches:       []string{"*"},
				OnPullRequestReadyForReview: true,
			}).WithJob(
				dag.Gha().Job("test", "test", dagger.GhaJobOpts{
					Runner: []string{"ubuntu-latest"},
					Module: "https://github.com/quay/clair-workflows/claircore@dagger",
				}))).
		Generate().
		WithDirectory(".",
			dag.Common().EmbeddedWorkflows(
				dag.CurrentModule().Source(),
			))

	return dir
}
