# decad gallery

This repository collects parametric shapes, patterns, and assemblies built with
[decad](https://github.com/lestrrat-3d/decad). Each item gets a directory under
[`models/`](models/README.md). Items may use geometry from other items, so a gear can
become part of a hand-crank siren or a wind turbine.

The repository will own each item's human-readable construction instructions, Go
code that builds it with decad, and checks of the resulting geometry. The
first item is the [spur gear](models/spurgear/README.md).

[`fusion360-gear-generator`](https://github.com/lestrrat-3d/fusion360-gear-generator)
is a downstream consumer. It will translate the Go construction into equivalent
Python code for Fusion add-ins. This repository owns the decad construction and
its build pipeline.

## Repository layout

| Path | Purpose |
| --- | --- |
| `models/<name>/` | One shape, pattern, or assembly per directory. |
| `models/README.md` | Rules for adding items and referring to other items. |
| `AGENTS.md` | Instructions for coding agents working on models. |
| `.github/` | Issue templates and repository checks. |

The repository check runs `python3 scripts/check_repository.py`. It checks the
foundation files and requires a README for each item directory. Run
`go test ./...` for the model's geometry checks; CI runs both commands.

## License

This repository uses the [PolyForm Noncommercial License 1.0.0](LICENSE).
