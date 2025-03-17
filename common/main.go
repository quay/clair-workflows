// A generated module for Common functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/quay/clair-workflows/common/internal/dagger"
)

type Common struct{}

// The base image for use with claircore.
func (m *Common) Builder(ctx context.Context) *dagger.Container {
	toolchain, _ := m.GoDistribution(ctx, "", "")
	// what do do with errors?

	return m.UBI("").
		WithMountedDirectory("/usr/local/go", toolchain).
		WithEnvVariable(
			"PATH",
			"/usr/local/go/bin:${PATH}",
			dagger.ContainerWithEnvVariableOpts{Expand: true},
		).
		WithEnvVariable("GOFLAGS", "-trimpath")
}

// Create an environment suitable for building the indicated source.
func (m *Common) BuildEnv(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	cgo bool,
) *dagger.Container {
	download := []string{"go", "mod", "download"}

	return m.Builder(ctx).
		With(addGoCaches(ctx)).
		WithDirectory("/src", source).
		WithWorkdir("/src").
		With(func(c *dagger.Container) *dagger.Container {
			const name = `CGO_ENABLED`
			if !cgo {
				return c.
					WithEnvVariable(name, "0")
			}
			return c.
				WithEnvVariable(name, "1").
				With(InstallPackage("gcc"))
		}).
		WithExec(download)
}

// Create an environment suitable for building the indicated source for release.
func (m *Common) ReleaseEnv(
	ctx context.Context,
	source *dagger.Directory,
) *dagger.Container {
	return m.BuildEnv(ctx, source, false).
		WithEnvVariable(
			"GOFLAGS",
			`${GOFLAGS} "-ldflags=-s -w"`,
			dagger.ContainerWithEnvVariableOpts{Expand: true},
		)
}

// Create an environment suitable for testing the indicated source.
func (m *Common) TestEnv(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	race bool,
	// +optional
	fips bool,
	// +optional
	database *dagger.Service,
) *dagger.Container {
	c := m.BuildEnv(ctx, source, race).
		With(addTestCaches(ctx)).
		WithEnvVariable("CI", "1")

	// TODO(hank) This is probably wrong, figure out what to do.
	if database != nil {
		c = c.
			WithServiceBinding(`db`, database).
			WithEnvVariable(`PG_HOST`, `db`)
	} else {
		c = c.With(PostgreSQLService)
	}

	return c.
		With(enableFIPS(ctx, fips))
}

// Return the result of running tests on the indicated source.
func (m *Common) Test(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	race bool,
	// +optional
	cover bool,
	// +optional
	fips bool,
	// +optional
	unit bool,
	// +optional
	database *dagger.Service,
) (string, error) {
	cmd := []string{"go", "test"}
	if !unit {
		cmd = append(cmd, `-tags=integration`)
	}
	if race {
		cmd = append(cmd, `-race`)
	}
	if cover {
		cmd = append(cmd, `-cover`)
	}
	cmd = append(cmd, "./...")

	ms, err := source.Glob(ctx, `**/go.mod`)
	if err != nil {
		return "", err
	}
	c, err := m.TestEnv(ctx, source, race, fips, database).Sync(ctx)
	if err != nil {
		return "", err
	}
	slices.Sort(ms)
	out := make([]string, len(ms))
	eg, ctx := errgroup.WithContext(ctx)

	for i, n := range ms {
		eg.Go(func() error {
			log, err := c.
				WithWorkdir(path.Dir(n)).
				WithExec(cmd).
				Stdout(ctx)
			if err != nil {
				return err
			}
			out[i] = strings.TrimSpace(log)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return "", err
	}
	return strings.Join(out, "\n"), nil
}

func PostgreSQLService(c *dagger.Container) *dagger.Container {
	const (
		user      = `claircore`
		plaintext = `hunter2hunter2` // Needs to be at least 14 characters, otherwise crypto/hmac trips FIPS-140 rules.
	)
	addr := `docker.io/library/postgres:` + Versions["postgresql"]
	pass := dag.SetSecret(`POSTGRES_PASSWORD`, plaintext)
	srv := dag.Container().
		From(addr).
		WithEnvVariable(`POSTGRES_USER`, user).
		WithSecretVariable(`POSTGRES_PASSWORD`, pass).
		WithEnvVariable(`POSTGRES_INITDB_ARGS`, `--no-sync`).
		WithMountedCache(`/var/lib/postgresql/data`, dag.CacheVolume(`claircore-postgresql`)).
		WithExposedPort(5432).
		AsService(dagger.ContainerAsServiceOpts{
			UseEntrypoint: true,
		})
	dsn := fmt.Sprintf(`host=db user=%s password=%s database=%[1]s sslmode=disable`, user, plaintext)
	return c.
		WithEnvVariable(`POSTGRES_CONNECTION_STRING`, dsn).
		WithServiceBinding(`db`, srv)
}

func addGoCaches(ctx context.Context) dagger.WithContainerFunc {
	return func(c *dagger.Container) *dagger.Container {
		return c.
			With(cacheDir(ctx, "go-build", "GOCACHE")).
			With(cacheDir(ctx, "go-mod", "GOMODCACHE"))
	}
}

func addTestCaches(ctx context.Context) dagger.WithContainerFunc {
	return cacheDir(ctx, `clair-testing`, "")
}

func enableFIPS(ctx context.Context, enable bool) dagger.WithContainerFunc {
	return func(c *dagger.Container) *dagger.Container {
		if enable {
			return c.WithEnvVariable("GODEBUG", "fips140=only")
		}
		return c
	}
}
