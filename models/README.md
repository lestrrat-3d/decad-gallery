# Models

| Model | Image | Description |
| --- | --- | --- |
| [Spur gear](spurgear/README.md) | <img src="spurgear/images/index.png" alt="Bored spur gear" width="240"> | A straight-toothed involute gear with optional bore and cap chamfers. |
| [Helical gear](helicalgear/README.md) | <img src="helicalgear/images/index.png" alt="Bored helical gear" width="240"> | An involute gear with teeth twisted along its axis. |
| [Herringbone gear](herringbonegear/README.md) | <img src="herringbonegear/images/index.png" alt="Bored herringbone gear" width="240"> | An involute gear with mirrored tooth slopes that meet at its midpoint. |
| [Bevel gear pair](bevelgear/README.md) | <img src="bevelgear/images/index.png" alt="Meshing bevel gear pair" width="240"> | Two meshing gears whose shafts meet at an angle. |

The index images are 480 × 270 pixels. Each model README uses a 1600 × 900
pixel hero image.

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
