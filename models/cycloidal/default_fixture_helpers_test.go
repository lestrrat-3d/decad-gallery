package cycloidal_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/stretchr/testify/require"
)

// driveFixture builds one drive per test binary for tests that only read it. A test that adds bodies to the
// document (overlap) or changes it in any other way must build its own drive.
type driveFixture struct {
	once  sync.Once
	doc   *decad.Document
	drive *cycloidal.Drive
	err   error
}

func (f *driveFixture) get(ctx context.Context, p cycloidal.Params) (*decad.Document, *cycloidal.Drive, error) {
	f.once.Do(func() {
		f.doc = chainDocument()
		drive, err := cycloidal.Build(ctx, f.doc, p)
		f.drive, f.err = &drive, err
	})
	return f.doc, f.drive, f.err
}

var defaultDriveFixture, twoDiscDriveFixture driveFixture

func defaultFixture(ctx context.Context, p cycloidal.Params) (*decad.Document, *cycloidal.Drive, error) {
	if p != cycloidal.DefaultParams() {
		return nil, nil, errors.New("default fixture requires exact default parameters")
	}
	return defaultDriveFixture.get(ctx, p)
}

// twoDiscParams is DefaultParams with two discs at the default gap, the drive TestBuildTwoDiscDrive,
// TestBuildContainers, TestCorners, TestPressFitWalls and TestExplode read.
func twoDiscParams() cycloidal.Params {
	p := cycloidal.DefaultParams()
	p.DiscCount, p.DiscGap = 2, cycloidal.MinCamNeck
	return p
}

func twoDiscFixture(ctx context.Context, p cycloidal.Params) (*decad.Document, *cycloidal.Drive, error) {
	if p != twoDiscParams() {
		return nil, nil, errors.New("two-disc fixture requires exact two-disc parameters")
	}
	return twoDiscDriveFixture.get(ctx, p)
}

// sharedDrive returns the shared drive for p when one exists, else builds one in a new document. The caller must
// only read the result.
func sharedDrive(t *testing.T, p cycloidal.Params) (*decad.Document, *cycloidal.Drive) {
	t.Helper()
	var (
		doc   *decad.Document
		drive *cycloidal.Drive
		err   error
	)
	switch p {
	case cycloidal.DefaultParams():
		doc, drive, err = defaultFixture(t.Context(), p)
	case twoDiscParams():
		doc, drive, err = twoDiscFixture(t.Context(), p)
	default:
		doc = chainDocument()
		var built cycloidal.Drive
		built, err = cycloidal.Build(t.Context(), doc, p)
		drive = &built
	}
	require.NoError(t, err)
	return doc, drive
}
