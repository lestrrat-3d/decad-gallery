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

## Decad gaps and bugs

- When decad or Sketch lacks a needed feature, stop that model change and follow **Agent coordination**.
- When decad or Sketch behaves incorrectly, reduce the failure to a local reproducer and use the same handoff path.
- State the affected model, expected geometry or behavior, observed result, and smallest reproducing input.
- Do not hide a decad gap or bug with substitute geometry, ad hoc calculations, or a different modeling engine.
- Do not modify decad or Sketch itself without authorization for that repository.

## Agent coordination

- Find the active Decad/Sketch Codex task with `mcp__codex_tui__list_threads`.
- Read that task's recent status with `mcp__codex_tui__read_thread`.
- Send model requirements, real reproducers, and gallery test results with `mcp__codex_tui__send_message_to_thread`.
- Consolidate all known needs for one model into one handoff; include the full downstream build and verification path.
- Coordinate ownership and status with that task before editing sibling repositories or closing a Decad/Sketch blocker.
- Recheck merged changes with the gallery's real model before resuming the port.
- Update the handoff when the other task reports a fix or a new blocker.
- Ask the user only for a decision the two tasks cannot make from existing authorization.

## Verification

- Run `python3 scripts/check_repository.py` after changing repository structure.
- Run the affected item's focused build and geometry checks once its build exists.
- Check a reused item through the consuming assembly before claiming the connection works.

## Model images

- Follow `models/README.md` for each completed model's hero image.
- Build images from the real Decad body with `tools/render/`; do not draw substitute geometry.
- Add a renderer integration test for each new model preset.
