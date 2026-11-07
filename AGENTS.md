# Agent instructions

## Repository purpose

- Maintain and grow a gallery of parametric Decad models for mechanical parts and assemblies,
  including gears, mechanisms, and standard hardware.
- Follow `docs/model-pipeline.md` when adding or changing models.
- Keep construction specifications, Decad Go code, and geometry checks aligned.

## Repository ownership

- Put each model in `models/<name>/`; give it a `README.md`.
- Keep each item's instructions, decad Go construction, build entry point, and geometry checks with that item.
- Put build steps shared by multiple items in repository tooling; do not duplicate them across item directories.
- Reuse another item's construction by reference or import; do not copy its geometry code.
- Treat this repository's decad construction as the source for downstream ports.
- Keep Fusion Python add-in code and its packaging in `fusion360-gear-generator`.

## Sketch constraints

- Verify every model Sketch before its geometry is consumed or the build returns.
- Require `Sketch.Verify(ctx)` to report `sketch.FullyConstrained`, `DOF == 0`, no redundant constraints, and no conflicts.
- Test this requirement for every parameter branch and every optional sketch, including bore and tool sketches.
- Anchor each sketch to its intended origin or reference entity.
- Constrain other geometry with geometric and dimensional constraints. Do not fix every computed point to force `DOF == 0`.
- Do not treat a successful `Solve`, a valid profile, or `DOF == 0` alone as proof of full constraint.

## Verification

- Run `python3 scripts/check_repository.py` after changing repository structure.
- Run the affected item's focused build and geometry checks once its build exists.
- Check a reused item through the consuming assembly before claiming the connection works.

## Model images

- For every new model, update the root `README.md` overview and gallery hero image.
- Add every new model to `models/README.md` with its description and index image.
- Follow `docs/adding-models.md` for the model's hero and index images.
- Build images from real Decad bodies with `tools/render/`; do not draw substitute geometry.
- Add a renderer integration test for each new model preset.
