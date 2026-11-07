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

## Fasteners

- Sink every bolt head in a pocket in the part it joins, so the head's top face lies flush with the part's face. Nothing stands proud.
- Hold every nut in a cavity the part encloses on all sides except the bolt hole, so the part retains the nut and it cannot fall out. Drop the nut in during a print pause. A pocket open to a face is not enough.
- Use `models/m2bolt` and `models/m2nut` for M2 fasteners; reference them, do not copy their geometry.
- Size the pockets from the fastener's constants plus 0.1 mm radial clearance. Make a head pocket as deep as the head is tall, and a nut cavity as high as the nut is thick plus the clearance.
- Keep the clearance hole at least the shank diameter (2.4 mm for M2). Keep 1 mm or more of material under a head pocket and around any pocket.
- Pick a standard bolt length from `m2bolt.StandardLengths()`. A housing too long for one bolt takes shorter bolts from each end, each into a nut cavity near the end of the middle part, not a custom length.
- The tip of a bolt may stay inside a hole beyond its nut. It must not stand outside a face.
- Test that each head ends flush and each nut sits inside its cavity (bounds), and that no fastener overlaps the parts it passes through.

## Verification

- Apply [Part clearances](docs/model-pipeline.md#part-clearances) to every modeled part fit.
- Apply [Gear train clearances](docs/model-pipeline.md#gear-train-clearances) to every gear pair in a model.
- Run `python3 scripts/check_repository.py` after changing repository structure.
- Run the affected item's focused build and geometry checks once its build exists.
- Check a reused item through the consuming assembly before claiming the connection works.

## Model images

- For every new model, update the root `README.md` overview and gallery hero image.
- Add every new model to `models/README.md` with its description and index image.
- Follow `docs/adding-models.md` for the model's hero and index images.
- Build images from real Decad bodies with `tools/render/`; do not draw substitute geometry.
- Add a renderer integration test for each new model preset.
- Regenerate images in the same change as any geometry change; never leave stale images behind.
- A geometry change includes edits to a model's construction, parameters, defaults, or a reused item or decad version that alters its shape.
- Regenerate the changed model's images in its `images/` directory and check its `README.md` still matches them.
- Regenerate the index image that `models/README.md` shows for the changed model, and check its row still matches.
- Regenerate `docs/images/gallery.png` when a changed model appears in it (`gallery/hero` preset in `docs/rendering.md`), and check the root `README.md` still matches.
- A change to a reused item regenerates the images of every model that consumes it.
