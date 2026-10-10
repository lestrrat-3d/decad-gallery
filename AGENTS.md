# Agent instructions

## Repository ownership

- Put each model in `models/<name>/`; give it a `README.md`.
- Keep each item's instructions, decad Go construction, build entry point, and geometry checks with that item.
- Put build steps shared by multiple items in repository tooling; do not duplicate them across item directories.
- Reuse another item's construction by reference or import; do not copy its geometry code.
- Treat this repository's decad construction as the source for downstream ports.
- Keep Fusion Python add-in code and its packaging in `fusion360-gear-generator`.

## Decad gaps and bugs

- When decad lacks a needed feature, stop that model change and ask the user for the missing capability.
- When decad behaves incorrectly, reduce the failure to a local reproducer and ask the user to address the bug.
- State the affected model, expected geometry or behavior, observed result, and smallest reproducing input.
- Do not hide a decad gap or bug with substitute geometry, ad hoc calculations, or a different modeling engine.
- Do not modify decad itself without authorization for that repository.

## Verification

- Run `python3 scripts/check_repository.py` after changing repository structure.
- Run the affected item's focused build and geometry checks once its build exists.
- Check a reused item through the consuming assembly before claiming the connection works.
