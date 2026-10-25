# Adding a model

Put each shape, pattern, or assembly in `models/<name>/`. Give each item a `README.md` for visitors to
the gallery: what the model is, what it looks like, and what a user can vary. Describe its Go API,
build command, geometry checks, and any other items it uses in `docs/models/<name>.md`. Keep
`instructions.md`, `steps.md`, Decad Go source, build entry point, and geometry checks with the item
they describe. Put reusable build tooling in a shared repository location when the first model needs it.

Give each completed item an `images/hero.png` rendered from its Decad body. The index image is
480 × 270 pixels and the hero image is 1600 × 900 pixels. Register the item's image preset in the
[model renderer](rendering.md) and show the image in the item's `README.md`. Add animations through the
same rendering tool when a model needs motion to explain it. Add the item to the table in
[`models/README.md`](../models/).

Follow the [model build pipeline](model-pipeline.md) when adding an item. The instructions own the
dimensions and behavior; the steps map them to Sketch and Decad operations.

An assembly may import reusable geometry from another item. Keep the reusable construction in its owning
item; do not copy it into the assembly. The spur gear in `models/spurgear` establishes the Go package
layout for these imports.

When decad lacks a needed operation or behaves incorrectly, follow [`AGENTS.md`](../AGENTS.md) and raise
the gap before changing the model.
