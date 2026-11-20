package renderkit_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/stretchr/testify/require"
)

func TestFitCameraCentresAndMargins(t *testing.T) {
	models := offCentreCubeModels(t)
	camera := lookingDownNegativeZ()
	const aspect, margin = 16.0 / 9, 0.1
	fitted := renderkit.FitCamera(camera, models, aspect, margin)
	minX, maxX, minY, maxY := projectedExtents(fitted, models)
	require.InDelta(t, 0, minX+maxX, 1e-6, "extents are centred horizontally")
	require.InDelta(t, 0, minY+maxY, 1e-6, "extents are centred vertically")
	tanHalf := math.Tan(fitted.FOV * math.Pi / 360)
	halfX, halfY := (maxX-minX)/2, (maxY-minY)/2
	require.InDelta(t, 1-margin*2, math.Max(halfX/aspect, halfY)/tanHalf, 1e-6, "requested margin is left free")
}

func TestFitCameraKeepsDirectionAndDistance(t *testing.T) {
	models := offCentreCubeModels(t)
	camera := lookingDownNegativeZ()
	fitted := renderkit.FitCamera(camera, models, 16.0/9, 0.1)
	require.InDelta(t, camera.Position.Z-camera.Target.Z, fitted.Position.Z-fitted.Target.Z, 1e-9)
	require.InDelta(t, 0, fitted.Position.X-fitted.Target.X, 1e-9)
}

func TestSceneKeepsCameraAndModels(t *testing.T) {
	models := offCentreCubeModels(t)
	camera := lookingDownNegativeZ()
	scene := renderkit.Scene(camera, models)
	require.Equal(t, camera, scene.Camera)
	require.Len(t, scene.Models, len(models))
	require.Len(t, scene.DirectionalLights, 2)
}
