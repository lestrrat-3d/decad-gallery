# Models

Put each shape, pattern, or assembly in `models/<name>/`. Give each item a
`README.md` that names its inputs, build command, output, geometry checks,
and any other items it uses. Keep `instructions.md`, `steps.md`, Decad Go
source, build entry point, and geometry checks with the item they describe.
Put reusable build tooling in a shared repository location when the first model
needs it.

Give each completed item an `images/hero.png` rendered from its Decad body.
Register its image preset in the [model renderer](../tools/render/README.md)
and show the image in the item's `README.md`. Add animations through the same
rendering tool when a model needs motion to explain it.

Follow the [model build pipeline](../docs/model-pipeline.md) when adding an
item. The instructions own the dimensions and behavior; the steps map them
to Sketch and Decad operations.

An assembly may import reusable geometry from another item. Keep the reusable
construction in its owning item; do not copy it into the assembly. The spur
gear in `models/spurgear` establishes the Go package layout for these imports.

When decad lacks a needed operation or behaves incorrectly, follow
[`AGENTS.md`](../AGENTS.md) and raise the gap before changing the model.
