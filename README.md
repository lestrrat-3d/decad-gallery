# decad gallery

![Decad spur gear hero image](models/spurgear/images/hero.png)

A gallery of parametric 3D shapes, patterns, and assemblies built with
[decad](https://github.com/lestrrat-3d/decad). Each model is a real solid body with adjustable
dimensions: you change the tooth count, thickness, bore or other inputs, and the model rebuilds. Items may
use geometry from other items, so a gear can become part of a hand-crank siren or a wind turbine.

## Models

The [models](models/) are the spur gear, helical gear, herringbone gear, cycloidal drive, and bevel gear
pair. Each model's page shows rendered images and lists the inputs you can change.

Every model directory holds two specification files. `instructions.md` states the inputs and defaults,
units, derived values, geometric construction, feature order, valid parameter range, and representative
cases. `steps.md` maps each construction requirement to a Sketch or Decad operation and names the geometry
check at each boundary.

## For developers

The [docs](docs/) explain how the models are built, how to add one, and how the images are rendered. The
[repository layout and checks](docs/repository.md) page lists what lives where.

[`fusion360-gear-generator`](https://github.com/lestrrat-3d/fusion360-gear-generator) translates the Go
construction into Python code for Fusion add-ins.

## License

This repository uses the [PolyForm Noncommercial License 1.0.0](LICENSE).
