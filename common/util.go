package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"runtime"
	"slices"
	"strings"
	"time"

	"golang.org/x/text/transform"

	"github.com/quay/clair-workflows/common/internal/dagger"
	"github.com/quay/clair-workflows/common/shquote"
)

func cacheDir(ctx context.Context, name string, env string) dagger.WithContainerFunc {
	ns, _ := dag.CurrentModule().Name(ctx)
	opts := dagger.CacheVolumeOpts{
		Namespace: ns,
	}
	return func(c *dagger.Container) *dagger.Container {
		dir, err := c.EnvVariable(ctx, "XDG_CACHE_HOME")
		if dir == "" || err != nil {
			dir = "/root/.cache"
		}
		cache := dag.CacheVolume(name, opts)
		path := path.Join(dir, name)

		c = c.WithMountedCache(path, cache)
		if env != "" {
			c = c.WithEnvVariable(env, path)
		}
		return c
	}
}

func (m *Common) GoDistribution(
	ctx context.Context,
	//+optional
	version string,
	//+optional
	arch string,
) (*dagger.Directory, error) {
	if version == "" {
		type File struct {
			Filename string `json:"filename"`
			OS       string `json:"os"`
			Arch     string `json:"arch"`
			Version  string `json:"version"`
			SHA256   string `json:"sha256"`
			Size     int64  `json:"size"`
			Kind     string `json:"kind"`
		}
		type Version struct {
			Version string `json:"version"`
			Stable  bool   `json:"stable"`
			Files   []File `json:"files"`
		}
		vf := m.GoVersions(ctx)

		contents, err := vf.Contents(ctx)
		if err != nil {
			return nil, err
		}
		var dl []Version
		if err := json.Unmarshal([]byte(contents), &dl); err != nil {
			return nil, err
		}

		v := slices.MaxFunc(dl, func(a, b Version) int {
			return strings.Compare(a.Version, b.Version)
		})
		version = v.Version
	}
	if !strings.HasPrefix(version, "go") {
		version = "go" + version
	}
	if arch == "" {
		arch = runtime.GOARCH
	}

	dist := fmt.Sprintf(`https://go.dev/dl/%s.linux-%s.tar.gz`, version, arch)
	tarball := dag.HTTP(dist)

	return m.Untar(ctx, tarball).Directory("go"), nil
}

func (m *Common) GoVersions(
	ctx context.Context,
) *dagger.File {
	url := fmt.Sprintf("https://go.dev/dl/?mode=json&as-of=%s", time.Now().Format(time.DateOnly))
	return dag.HTTP(url)
}

func (m *Common) Untar(ctx context.Context, tarball *dagger.File) *dagger.Directory {
	const (
		wd     = `/run/untar`
		arPath = `/tmp/archive`
	)
	cmd := []string{`tar`, `--extract`, `--auto-compress`, `--directory`, wd, `--file`, arPath}

	return m.UBI("").
		With(InstallPackage("gzip", "xz", "zstd", "bzip2")).
		WithFile(arPath, tarball).
		WithMountedDirectory(wd, dag.Directory()).
		WithExec(cmd).
		Directory(wd)
}

func (*Common) UBI(
	//+optional
	tag string,
) *dagger.Container {
	if tag == "" {
		tag = Versions["ubi"]
	}
	ref := fmt.Sprintf(`registry.access.redhat.com/ubi9:%s`, tag)
	return dag.Container().
		From(ref)
}

func InstallPackage(pkgs ...string) dagger.WithContainerFunc {
	var script strings.Builder
	tf := new(shquote.Transformer)

	script.WriteString(`dnf install --assumeyes `)
	for _, pkg := range pkgs {
		p, _, err := transform.String(tf, pkg)
		if err != nil { //???
			panic(err)
		}
		script.WriteString(p)
		script.WriteByte(' ')
	}
	script.WriteString(`&& dnf clean all`)

	return RunBash(script.String())
}

func RunBash(script string) dagger.WithContainerFunc {
	return func(c *dagger.Container) *dagger.Container {
		return c.WithExec([]string{"bash", "-ec", script})
	}
}
