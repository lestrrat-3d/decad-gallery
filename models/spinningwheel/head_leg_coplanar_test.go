//go:build coplanar_test

package spinningwheel

import (
	"context"
	"strconv"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

//nolint:containedctx // This context stops the real joinery sequence after the selected component.
type headLegStopContext struct {
	context.Context
	builder          *builder
	index            int
	target, tool     *decad.Body
	cancel           context.CancelFunc
	captured         bool
	stopped          bool
	beforeBodies     []*decad.Body
	beforeOperations []*decad.Operation
	container        *decad.Container
	toolContainer    *decad.Container
	captureError     error
}

func (c *headLegStopContext) Err() error {
	if c.builder.a.Parts[c.index].Body != c.target {
		c.stopped = true
		c.cancel()
	}
	if !c.captured && c.builder.name == "head-leg" {
		operations := c.builder.doc.OperationHistory().Operations()
		if len(operations) != 0 && operations[len(operations)-1].Kind() == decad.OperationUnion {
			outputs := operations[len(operations)-1].Outputs()
			if len(outputs) == 1 {
				c.captured = true
				c.tool = outputs[0]
				c.beforeOperations = operations
				c.beforeBodies = c.builder.doc.Bodies()
				c.container, c.captureError = c.builder.doc.ContainerOf(c.target)
				if c.captureError == nil {
					c.toolContainer, c.captureError = c.builder.doc.ContainerOf(c.tool)
				}
			}
		}
	}
	return c.Context.Err()
}

func newHeadLegConsumer(t *testing.T) (*builder, *headLegStopContext) {
	t.Helper()
	doc := NewDocument()
	b, err := newAssemblyBuilder(t.Context(), doc, DefaultParams())
	require.NoError(t, err)
	b.frame()
	require.NoError(t, b.err)
	index := b.partIndex("head-leg")
	require.GreaterOrEqual(t, index, 0)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	probe := &headLegStopContext{
		Context: ctx, builder: b, index: index, target: b.a.Parts[index].Body, cancel: cancel,
	}
	b.ctx = probe
	// Use the production method, including its actual tool constructors and Cut.
	// The context stops its later work after machine registers the head-leg result.
	b.frameJoinery()
	require.True(t, probe.captured, "the real head-leg tool Union must precede its Cut")
	require.NoError(t, probe.captureError)
	return b, probe
}

func requireHeadLegCut(t *testing.T, b *builder, probe *headLegStopContext) *decad.Body {
	t.Helper()
	require.True(t, probe.stopped, "the head-leg Cut must finish before the later joinery stops")
	require.ErrorIs(t, b.err, context.Canceled)
	result := b.a.Parts[probe.index].Body
	require.True(t, result != probe.target)
	require.True(t, result.IsSolid())
	operations := b.doc.OperationHistory().Operations()
	require.Equal(t, len(probe.beforeOperations)+1, len(operations))
	for i, operation := range probe.beforeOperations {
		require.True(t, operations[i] == operation, "operation %d changed", i)
	}
	cut := operations[len(operations)-1]
	require.True(t, cut.Kind() == decad.OperationCut)
	inputs, outputs := cut.Inputs(), cut.Outputs()
	require.Equal(t, 2, len(inputs))
	require.Equal(t, 1, len(outputs))
	require.True(t, inputs[0].Body == probe.target && inputs[1].Body == probe.tool)
	require.Equal(t, "target", inputs[0].Role)
	require.Equal(t, "tool", inputs[1].Role)
	require.True(t, inputs[0].Use == decad.InputConsumed && inputs[1].Use == decad.InputConsumed)
	require.True(t, outputs[0] == result)
	live := b.doc.Bodies()
	require.Equal(t, len(probe.beforeBodies)-1, len(live))
	for i, previous := range probe.beforeBodies {
		found := false
		for _, body := range live {
			found = found || body == previous
		}
		want := previous != probe.target && previous != probe.tool
		require.True(t, found == want, "live state changed for prior body %d", i)
	}
	container, err := b.doc.ContainerOf(result)
	require.NoError(t, err)
	require.True(t, container == probe.container)
	return result
}

func requirePublicHeadLegState(t *testing.T, doc *decad.Document, bodies []*decad.Body, operations []*decad.Operation) {
	t.Helper()
	afterBodies, afterOperations := doc.Bodies(), doc.OperationHistory().Operations()
	require.Equal(t, len(bodies), len(afterBodies))
	require.Equal(t, len(operations), len(afterOperations))
	for i, body := range bodies {
		require.True(t, body == afterBodies[i], "live body %d changed", i)
	}
	for i, operation := range operations {
		require.True(t, operation == afterOperations[i], "operation %d changed", i)
	}
}

type headLegLimits struct{ work, records, bytes uint64 }

func setHeadLegLimits(t *testing.T, prefix string, limits headLegLimits) {
	t.Helper()
	t.Setenv(prefix+"_WORK", strconv.FormatUint(limits.work, 10))
	t.Setenv(prefix+"_RECORDS", strconv.FormatUint(limits.records, 10))
	t.Setenv(prefix+"_BYTES", strconv.FormatUint(limits.bytes, 10))
}

func headLegLimitCases(exact headLegLimits) []struct {
	name   string
	limits headLegLimits
	short  bool
} {
	return []struct {
		name   string
		limits headLegLimits
		short  bool
	}{
		{"exact", exact, false},
		{"work-one-short", headLegLimits{exact.work - 1, exact.records, exact.bytes}, true},
		{"records-one-short", headLegLimits{exact.work, exact.records - 1, exact.bytes}, true},
		{"bytes-one-short", headLegLimits{exact.work, exact.records, exact.bytes - 1}, true},
	}
}

func requireDetachedHeadLeg(t *testing.T, result *decad.Body) *decad.Body {
	t.Helper()
	detached, err := decad.Copy(t.Context(), result)
	require.NoError(t, err)
	require.True(t, detached != result, "the verifier must read a separate body identity")
	mesh, err := detached.Tessellate(t.Context(), units.Millimeters(.12), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	return detached
}

// This component gate runs in Decad's tagged integration job. The ordinary full
// assembly tests remain required and do not enable the SDK's test-only issuer.
func TestSpinningWheelHeadLegTaggedConsumer(t *testing.T) {
	const workCap, recordCap, byteCap = 32_000_000, 4_194_304, 2 << 30
	setHeadLegLimits(t, "COPLANAR_TEST", headLegLimits{workCap, recordCap, byteCap})
	setHeadLegLimits(t, "VERIFY_TEST", headLegLimits{workCap, recordCap, byteCap})
	b, probe := newHeadLegConsumer(t)
	result := requireHeadLegCut(t, b, probe)
	detached := requireDetachedHeadLeg(t, result)
	boundary, volume, work, records, bytes, err := decad.TestCoplanarVerify(t.Context(), detached, .12)
	require.NoError(t, err)
	require.True(t, boundary && volume)
	cutWork, cutRecords, cutBytes := decad.TestCoplanarLastUsage()
	for _, count := range []uint64{cutWork, cutRecords, cutBytes, work, records, bytes} {
		require.Positive(t, count)
	}
	t.Logf("Cut work=%d records=%d bytes=%d; detached VerifyAll work=%d records=%d bytes=%d",
		cutWork, cutRecords, cutBytes, work, records, bytes)
	for _, test := range headLegLimitCases(headLegLimits{cutWork, cutRecords, cutBytes}) {
		t.Run("Cut/"+test.name, func(t *testing.T) {
			setHeadLegLimits(t, "COPLANAR_TEST", test.limits)
			cutBuilder, observed := newHeadLegConsumer(t)
			if !test.short {
				requireDetachedHeadLeg(t, requireHeadLegCut(t, cutBuilder, observed))
				return
			}
			require.ErrorIs(t, cutBuilder.err, decad.ErrUnsupported)
			require.False(t, observed.stopped)
			require.True(t, cutBuilder.a.Parts[observed.index].Body == observed.target)
			requirePublicHeadLegState(t, cutBuilder.doc, observed.beforeBodies, observed.beforeOperations)
			for _, source := range []struct {
				body      *decad.Body
				container *decad.Container
			}{{observed.target, observed.container}, {observed.tool, observed.toolContainer}} {
				container, err := cutBuilder.doc.ContainerOf(source.body)
				require.NoError(t, err)
				require.True(t, container == source.container)
			}
		})
	}
	beforeBodies, beforeOperations := b.doc.Bodies(), b.doc.OperationHistory().Operations()
	for _, test := range headLegLimitCases(headLegLimits{work, records, bytes}) {
		t.Run("detached-VerifyAll/"+test.name, func(t *testing.T) {
			setHeadLegLimits(t, "VERIFY_TEST", test.limits)
			boundary, volume, w, r, z, err := decad.TestCoplanarVerify(t.Context(), detached, .12)
			if test.short {
				require.ErrorIs(t, err, decad.ErrUnsupported)
				require.False(t, boundary || volume)
			} else {
				require.NoError(t, err)
				require.True(t, boundary && volume)
				require.True(t, w <= test.limits.work && r <= test.limits.records && z <= test.limits.bytes)
			}
			requirePublicHeadLegState(t, b.doc, beforeBodies, beforeOperations)
		})
	}
}
