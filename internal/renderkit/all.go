package renderkit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
)

// JobPath returns filepath.Join(root, filepath.FromSlash(m.Dir), filepath.FromSlash(j.Out)).
func JobPath(root string, m Model, j Job) string {
	return filepath.Join(root, filepath.FromSlash(m.Dir), filepath.FromSlash(j.Out))
}

// RenderAll runs every job of every model in order with Output JobPath(root, m, j) and Kit j.Kit. It writes one
// "render <model> <shot> <path>" line per job to log, keeps going after a failure, and returns errors.Join of one
// "<model>/<shot>: <err>" error per failed job, or nil.
func RenderAll(ctx context.Context, root string, models []Model, log io.Writer) error {
	var failures []error
	for _, m := range models {
		for _, j := range m.Jobs {
			path := JobPath(root, m, j)
			_, _ = fmt.Fprintf(log, "render %s %s %s\n", m.Name, j.Shot, path)
			if err := Run(ctx, m, j.Shot, Options{Output: path, Kit: j.Kit}); err != nil {
				failures = append(failures, fmt.Errorf("%s/%s: %w", m.Name, j.Shot, err))
			}
		}
	}
	return errors.Join(failures...)
}
