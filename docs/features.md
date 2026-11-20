# Features

A feature is a reusable shape that cuts or adds material on parts that other items own. It makes no part of its
own. Joinery is the typical case: a dovetail, an ari-tsugi, a kanawa-tsugi, or a mortise and tenon. A feature
gives other items one tested construction to call instead of a copy.

Features live in `features/<name>/`. The gallery of finished products stays in `models/`.

## Feature or model

| The item | It goes in |
| --- | --- |
| Makes bodies that are parts of its own, such as a hinge leaf, a pin, a gear or a nut | `models/<name>/` |
| Only shapes bodies that a caller passes in, such as a dovetail cut into two boards | `features/<name>/` |

A model that other models import stays a model. `models/crankhandle` and `models/spurgear` already work this
way. A feature may import a model, as `internal/m2fastener` imports the M2 preset of `models/fastener`.

A feature never becomes a gallery entry. A feature's name must differ from every model name, because image
presets are named `<name>/<shot>` and share one list.

## Files

Each feature directory holds these files, with the same roles they have in a model:

| File | Role |
| --- | --- |
| `README.md` | What the feature is, its build status line, its images, and the demo's host blocks. |
| `instructions.md` | The construction contract. See [Write the construction contract](model-pipeline.md#1-write-the-construction-contract). |
| `steps.md` | The Decad steps. See [Compile the contract into Decad steps](model-pipeline.md#2-compile-the-contract-into-decad-steps). |
| `<name>.go` | The Go API. |
| `demo.go` | `BuildDemo`, the minimal build used by the tests and the images. |
| `<name>_test.go` | The geometry checks. |
| `images/` | `hero.png`, `index.png`, the angle views and `exploded.png`. |

Follow the [model build pipeline](model-pipeline.md) for the contract, the steps, the Go code and the
verification. Put `docs/features/<name>.md` beside the model pages in `docs/models/` once the API exists.

## API

A feature builds every body it needs from one parameter struct. A joint with two mating halves returns both
halves from that struct, so the halves cannot drift apart.

The caller supplies the host bodies and the frame where the feature goes. The feature cuts or adds its shape
there. It returns each sketch it made, so the caller can verify the sketches before using their geometry. See
`m2fastener.PocketTool` for the same pattern.

A model imports a feature as `github.com/lestrrat-3d/decad-gallery/features/<name>`. A model never copies a
feature's geometry.

## Demo build

`BuildDemo(ctx, doc, params)` builds the smallest assembly that exercises the feature. For a dovetail, that is
two host blocks with the joint cut into them, assembled. The demo is a test fixture and the source of the
images. It is not a gallery model.

The tests run the demo for every parameter branch and check these points:

- Every sketch reports `sketch.FullyConstrained`, `DOF == 0`, no redundant constraints and no conflicts. This
  includes optional sketches. See `AGENTS.md`, "Sketch constraints".
- The mating halves assemble at the given frame and do not overlap. Use `internal/meshgap` for the gap, and
  `Document.Verify`, `decad.Intersect` and `decad.Union` for overlap when the halves touch in exact floats.
- The bounds of each host block and each half match the parameters.
- Similar bodies sit in named containers. See [Body containers](model-pipeline.md#body-containers).
- Corners follow [Corners and roots](model-pipeline.md#corners-and-roots). A joinery corner that must match
  its mate is a mating corner there.

Fit clearances for wood joinery are separate from this page.

## Images

Every feature shows its demo in its `README.md` from several angles, with an exploded view that pulls the
halves apart along the direction they slide together. Size and place the exploded view by
[Exploded views](model-pipeline.md#exploded-views).

- Render every image from the demo's real Decad bodies with `tools/render/`, using
  `-out features/<name>/images`. See the [renderer](rendering.md).
- Give each feature a 480 × 270 `index.png` and a 1600 × 900 `hero.png`.
- Regenerate the images in the same change as any geometry change.
- A change to a feature regenerates the images of every model that imports it.

The root `README.md` gallery and `docs/images/gallery.png` show models only. A feature does not appear in them.

## Index

`features/README.md` lists every feature in alphabetical order by name. Its columns are the feature name linked
to its directory, the index image, a one-line description, and the build status.

## Build status

Put `**Build status:** Built and tested.` or `**Build status:** Not built yet.` on the line under the README
title, and fill the Build status column of `features/README.md` with `Built` or `Not built yet`. A new feature
starts as not built yet. Change the status only when the user says someone printed and tested it.
