package cycloidal_test

import (
	"context"
	"errors"
	"sync"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
)

var defaultDriveFixture struct {
	once  sync.Once
	doc   *decad.Document
	drive *cycloidal.Drive
	err   error
}

func defaultFixture(ctx context.Context, p cycloidal.Params) (*decad.Document, *cycloidal.Drive, error) {
	if p != cycloidal.DefaultParams() {
		return nil, nil, errors.New("default fixture requires exact default parameters")
	}
	defaultDriveFixture.once.Do(func() {
		defaultDriveFixture.doc = chainDocument()
		drive, err := cycloidal.Build(ctx, defaultDriveFixture.doc, p)
		defaultDriveFixture.drive, defaultDriveFixture.err = &drive, err
	})
	return defaultDriveFixture.doc, defaultDriveFixture.drive, defaultDriveFixture.err
}
