// Module for Claircore functions

package main

import (
	"context"

	"github.com/quay/clair-workflows/claircore/internal/dagger"
)

// Claircore holds actions for the claircore repo.
type Claircore struct{}

// Test runs the tests for all the go modules in the Claircore repository.
func (m *Claircore) Test(
	ctx context.Context,
	// Source to use for testing.
	//
	// If omitted, `https://github.com/quay/claircore@main` will be used.
	//
	//+ignore=[".git"]
	//+optional
	source *dagger.Directory,
	// Run unit tests.
	//
	//+optional
	//+default=true
	unit bool,
	// Build and run tests with the race detector.
	//
	//+optional
	race bool,
	// Build and run tests with coverage information.
	//
	//+optional
	cover bool,
	// Run tests with upstream FIPS 140 support. Requires go >=1.24.
	//
	//+optional
	fips bool,
) (string, error) {
	if source == nil {
		source = dag.Git("https://github.com/quay/claircore").Branch("main").Tree()
	}
	opts := dagger.CommonTestOpts{
		Race:     race,
		Cover:    cover,
		Unit:     unit,
		Database: nil,
		Fips:     fips,
	}
	return dag.Common().Test(ctx, source, opts)
}

// Actions creates a [dagger.Directory] containing generated GitHub Actions
// workflows.
//
// Use the "export" command to output to the desired directory:
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
