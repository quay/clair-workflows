// Modernize is a wrapper around running
// golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize.
//
// This wrapper integrates with git and modernize to produce very small diffs.
package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// DryRun signals to not commit changes, and instead just print that changes
// happened.
var dryRun = flag.Bool("n", false, "dry-run")

func main() {
	// Do setup and command-line handling, then call into [Main].
	code := 0
	defer func() {
		if code != 0 {
			os.Exit(code)
		}
	}()

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	var level slog.LevelVar
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: &level,
	})))

	flag.BoolFunc("D", "enable debug logging", func(v string) error {
		ok, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		if ok {
			level.Set(slog.LevelDebug)
		} else {
			level.Set(slog.LevelInfo)
		}
		return nil
	})
	flag.Parse()

	expr := "./..."
	if flag.NArg() > 0 {
		expr = flag.Arg(0)
	}

	if err := Main(ctx, expr); err != nil {
		slog.Error("failed", "error", err)
		code = 1
	}
}

// Main runs `modernize` on all go packages in directories tracked by git
// matching the package pattern "expr".
func Main(ctx context.Context, expr string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dirsSeq, err := listDirs(ctx, pwd)
	if err != nil {
		return err
	}
	dirpkg, err := listPackages(ctx, expr)
	if err != nil {
		return err
	}
	dirs := slices.Collect(dirsSeq)
	slog.DebugContext(ctx, "found git-tracked directories", "dirs", dirs)

	for dir, pkg := range dirpkg {
		slog.DebugContext(ctx, "found package", "dir", dir, "pkg", pkg)
		if !slices.Contains(dirs, dir) {
			continue
		}
		if err := modernize(ctx, dir, pkg); err != nil {
			return err
		}
	}

	return nil
}

// Modernize runs the `modernize` tool over the package "pkg" in the directory
// "dir", one pass at a time. Passes are run individually to allow for
// committing changes after every pass.
//
// If [*dryRun] is true, changes made are reset (via "git checkout") instead of
// being committed.
//
// See also: [passes].
func modernize(ctx context.Context, dir string, pkg string) error {
	var stderr bytes.Buffer
	l := slog.With("dir", dir, "pkg", pkg)
	l.DebugContext(ctx, "modernizing")
	for _, p := range passes {
		l := l.With("pass", p)
		cmd := exec.CommandContext(ctx, "go", "run", cmdpath, "-fix", "-test", "-category", p, ".")
		cmd.Dir = dir
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			l.ErrorContext(ctx, "modernize failed", "output", stderr.String())
			return err
		}
		if gitDiff(ctx, dir) == nil {
			continue
		}
		l.DebugContext(ctx, "made changes")
		msg := fmt.Sprintf("%s: modernize: %s", pkg, p)
		l.InfoContext(ctx, "committing changes", "message", msg, "dry_run", *dryRun)
		if *dryRun {
			l.DebugContext(ctx, "resetting repo state")
			if err := gitReset(ctx, dir); err != nil {
				return err
			}
			continue
		}
		if err := gitCommit(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

// ListDirs returns a sequence of all the tracked directories under "dir".
//
// "Dir" is prepended to all the results of the underlying `git` command.
func listDirs(ctx context.Context, dir string) (iter.Seq[string], error) {
	cmd := exec.CommandContext(ctx, "git", "ls-tree", "-rtd", "--format", "%(path)", "HEAD", dir)
	seq, err := runLines(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return func(yield func(string) bool) {
		for p := range seq {
			if !yield(filepath.Join(dir, p)) {
				return
			}
		}
	}, nil
}

// ListPackages returns a sequence of "directory", "package name" pairs for
// packages matching the package pattern "expr".
func listPackages(ctx context.Context, expr string) (iter.Seq2[string, string], error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-f", "{{.Dir}}\t{{.Name}}", expr)
	lines, err := runLines(ctx, cmd)
	if err != nil {
		return nil, err
	}

	seq := func(yield func(string, string) bool) {
		for l := range lines {
			fs := strings.Split(l, "\t")
			if !yield(fs[0], fs[1]) {
				return
			}
		}
	}
	return seq, nil
}

// RunLines is a helper that runs "cmd" and returns the lines as a sequence of
// strings.
func runLines(ctx context.Context, cmd *exec.Cmd) (iter.Seq[string], error) {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	l := slog.With("cmd", cmd.Args)
	s := bufio.NewScanner(rc)

	if err := cmd.Start(); err != nil {
		l.ErrorContext(ctx, "command failed", "output", stderr.String())
		return nil, err
	}
	seq := func(yield func(string) bool) {
		defer rc.Close()

		for s.Scan() {
			if !yield(s.Text()) {
				return
			}
		}

		if err := s.Err(); err != nil {
			l.ErrorContext(ctx, "reading from inferior command", "error", err)
		}
		if err := cmd.Wait(); err != nil {
			l.ErrorContext(ctx, "waiting for inferior command", "error", err)
		}
	}
	return seq, nil
}

// GitDiff reports an error if there are changes in tracked files in "dir".
func gitDiff(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "diff", "--exit-code")
	cmd.Dir = dir
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// GitCommit runs the equivalent of "git commit -asm $msg".
func gitCommit(ctx context.Context, msg string) error {
	return exec.CommandContext(ctx, "git", "commit", "--all", "--signoff", "--message", msg).Run()
}

// GitReset resets files in "dir" to their state in the index.
func gitReset(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", "--", ".")
	cmd.Dir = dir
	return cmd.Run()
}

// Cmdpath is the path passed to `go run`.
const cmdpath = `golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest`

// Passes is all current "modernize" passes.
var passes = []string{
	"forvar",
	"slicescontains",
	"minmax",
	"sortslice",
	"efaceany",
	"mapsloop",
	"fmtappendf",
	"testingcontext",
	"omitzero",
	"bloop",
	"rangeint",
	"stringsseq",
	"stringscutprefix",
	"waitgroup",
}
