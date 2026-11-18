# decad gallery

![A spinning wheel, a hand-crank fan, a hand-crank siren, a Graham escapement, a Swiss lever escapement, a crank handle, a one-way clutch, a spiral spring, a turbine fan, a plaque stand and a clamp arm beside spur, helical, herringbone, planetary, cycloidal and bevel gears](docs/images/gallery.png)

A gallery of parametric 3D shapes, patterns, and assemblies built with
[decad](https://github.com/lestrrat-3d/decad). Each model is a solid body whose dimensions come from
inputs such as tooth count, thickness, and bore. Items may use geometry from other items, so a gear can
become part of a hand-crank siren or a wind turbine.

## Models

The [model table](models/) lists every model with its description, image, and build status.

## For developers

Every model directory holds two specification files. `instructions.md` states the inputs and defaults,
units, derived values, geometric construction, feature order, valid parameter range, and representative
cases. `steps.md` maps each construction requirement to a Sketch or Decad operation and names the geometry
check at each boundary.

The [docs](docs/) explain how the models are built, how to add one, and how the images are rendered. The
[repository layout and checks](docs/repository.md) page lists what lives where.
The [dependency updater](docs/dependency-updates.md) keeps the root and renderer Decad versions together.

[`fusion360-gear-generator`](https://github.com/lestrrat-3d/fusion360-gear-generator) translates the Go
construction into Python code for Fusion add-ins.
[`fusion-addins`](https://github.com/lestrrat-3d/fusion-addins) holds the Autodesk Fusion add-ins generated from
these models, with a script to install each one into Fusion.

## License

This repository uses the [PolyForm Noncommercial License 1.0.0](LICENSE).
