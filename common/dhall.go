package main

import (
	"context"
	"fmt"

	"github.com/quay/clair-workflows/common/internal/dagger"
)

func (m *Common) Dhall(
	ctx context.Context,
) *dagger.Container {
	const tmpl = "https://github.com/dhall-lang/dhall-haskell/releases/download/%s/dhall-json-%s-x86_64-linux.tar.bz2"
	// Only a single arch provided by upstream.
	dist := fmt.Sprintf(tmpl, Versions["dhall"], Versions["dhall-json"])
	tarball := dag.HTTP(dist)
	bin := m.Untar(ctx, tarball).Directory("bin")
	return m.UBI("").
		WithMountedDirectory("/usr/local/dhall/bin", bin).
		WithEnvVariable(
			"PATH",
			"/usr/local/dhall/bin:${PATH}",
			dagger.ContainerWithEnvVariableOpts{Expand: true},
		)
}

// CompileDhall returns the results of `dhall-to-yaml` called on all `*.dhall`
// files in "source", with the common dhall helpers mounted on `./lib`.
func (m *Common) CompileDhall(
	ctx context.Context,
	source *dagger.Directory,
) *dagger.Directory {
	const script = `for f in *.dhall
do
	dhall-to-yaml --documents --generated-comment --preserve-header --file "$f" --output "/out/${f/%dhall/yml}"
done`
	lib := dag.CurrentModule().
		Source().
		Directory("dhall")

	return m.Dhall(ctx).
		WithDirectory("/src", source).
		WithDirectory("/src/lib", lib).
		WithDirectory("/out", dag.Directory()).
		WithWorkdir("/src").
		With(RunBash(script)).
		Directory("/out")
}
