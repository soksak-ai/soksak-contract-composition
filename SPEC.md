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
The first settings document is generation one and reports a change from generation zero.

## Unit kinds

Installable runtime unit identity covers plugin, sidecar and kit. Every unit has exact kind, id and
version identity. Version ranges are not part of this contract.

Plugin is the user-facing activation root. settings.json has one explicit enabled selection for
every installed plugin. Sidecars and kits are not independently enabled; the resolved graph marks
them active only when an enabled, resolved plugin reaches them through dependency or binding edges.
Several plugins may reference one exact sidecar or kit node, which is installed and started once.

Contracts and specifications are exact references in unit manifests and certified conformance
reports in the plugin registry. They are not runtime installations. Host services are core build
components and are not plugin-registry units.

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

Every unit manifest declares exactly one initial provider for each consumed requirement. The
installer copies these declarations into settings bindings. A later settings transaction may select
another installed provider that implements the same exact contract; removing a binding rejects the
consumer rather than restoring the manifest choice implicitly.

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
