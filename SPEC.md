# Soksak composition contract 0.0.1

The file at identity-home/settings.json is the installation composition record. For the release
identity this is ~/.soksak/settings.json. Other identities use their own home and never modify the
release composition. The file declares every selected unit's exact
identity, activation state, absolute install path, manifest location and installation source.
Loaders resolve no sibling checkout and guess no kind-specific directory.

## Installation modes

- installed units are managed by the updater.
- development units are never written by the updater, which returns development-unit.

Source and mode are separate axes. Source records acquisition provenance: an exact Git commit, a
SHA-256-pinned archive, or an explicit absolute local path. Mode records update policy. Changing an
installed Git or archive unit to development preserves its source and prevents updater writes. A
Git checkout is not implicitly development; only the settings selection changes its mode.

Changing mode, path, activation or binding replaces the settings document atomically. Every write
uses compare-and-swap against generation and advances it by exactly one. The runtime publishes one
composition.changed event after resolving the new document; loaders, updater and UI do not poll.

## Unit kinds

The common identity covers plugin, sidecar, kit, contract, spec and service. Every unit has exact
kind, id and version identity. Version ranges are not part of this contract.

## Paths

Install and development paths are clean absolute paths. Symbolic links are rejected by the host
that reads the filesystem. Manifest paths are safe paths relative to the install path.

## Unit manifests

Every install path contains soksak-unit.json. The manifest repeats exact unit identity and declares:

- exact unit dependencies;
- exact public contracts implemented;
- named contract requirements consumed;
- relative entrypoints.

Unit dependencies and contract bindings are separate edges. A dependency requires one exact unit.
A binding selects the exact provider for one named consumer requirement. No provider is selected by
directory order, install order or fallback.

## Resolved graph

The resolver publishes nodes, dependency and binding edges, and issues. Each node reports its
installed or development mode and one status: resolved, disabled or rejected.

A missing manifest, dependency, binding or provider; a disabled provider; an exact contract
mismatch; or a dependency cycle rejects the affected node and its dependents. It does not stop an
unrelated resolved node. Invalid settings syntax or identity is the only document-level failure.

The resolved graph is computed output. It is exposed through status and commands and is not copied
back into settings.json.

## Failure

Unknown fields, unknown enum values, relative paths, unpinned sources and duplicate exact unit
identities are rejected. No legacy settings reader, path fallback or implicit provider exists.
