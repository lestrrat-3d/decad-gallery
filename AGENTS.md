# Agent instructions

## Active goal

- Build the pipeline that turns human-readable model specifications into parametric decad Go models.
- Port the spur gear pipeline from `../fusion360-gear-generator` first.
- Use `models/spurgear/instructions.md` as the gallery model contract.
- Follow `docs/model-pipeline.md` to compile instructions into Decad steps, Go code, and geometry checks.
- Complete the specified spur gear parameter cases and geometry checks before treating the milestone as done.
- Adapt Fusion-specific steps when the same geometry has a simpler native decad construction.
- Keep Fusion Python add-in generation downstream of this repository's Go construction.

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

- Follow `docs/adding-models.md` for each completed model's hero image.
- Build images from the real Decad body with `tools/render/`; do not draw substitute geometry.
- Add a renderer integration test for each new model preset.
- Regenerate images in the same change as any geometry change; never leave stale images behind.
- A geometry change includes edits to a model's construction, parameters, defaults, or a reused item or decad version that alters its shape.
- Regenerate the changed model's images in its `images/` directory and check its `README.md` still matches them.
- Regenerate the index image that `models/README.md` shows for the changed model, and check its row still matches.
- Regenerate `docs/images/gallery.png` when a changed model appears in it (`gallery/hero` preset in `docs/rendering.md`), and check the root `README.md` still matches.
- A change to a reused item regenerates the images of every model that consumes it.
