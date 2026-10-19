# decad gallery

![Decad spur gear hero image](models/spurgear/images/hero.png)

This repository collects parametric shapes, patterns, and assemblies built with
[decad](https://github.com/lestrrat-3d/decad). Each item gets a directory under
[`models/`](models/README.md). Items may use geometry from other items, so a gear can
become part of a hand-crank siren or a wind turbine.

The repository owns each item's human-readable construction instructions,
Decad construction steps, Go code, and geometry checks. The
[model build pipeline](docs/model-pipeline.md) describes this path. The first
item is the [spur gear](models/spurgear/README.md).

[`fusion360-gear-generator`](https://github.com/lestrrat-3d/fusion360-gear-generator)
is a downstream consumer. It will translate the Go construction into equivalent
Python code for Fusion add-ins. This repository owns the decad construction and
its build pipeline.

## Repository layout

| Path | Purpose |
| --- | --- |
| `models/<name>/` | One shape, pattern, or assembly per directory. |
| `models/README.md` | Rules for adding items and referring to other items. |
| `tools/render/` | Builds model images from Decad bodies. |
| `docs/model-pipeline.md` | The specification-to-Decad build workflow. |
| `AGENTS.md` | Instructions for coding agents working on models. |
| `.github/` | Repository checks. |

The repository check runs `python3 scripts/check_repository.py`. It checks the
foundation files and requires a README for each item directory. Run
`go test ./...` for the model's geometry checks; CI runs both commands.

## License

This repository uses the [PolyForm Noncommercial License 1.0.0](LICENSE).
