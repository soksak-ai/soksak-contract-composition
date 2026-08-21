# Soksak composition contract 0.0.1

settings.json is the installation composition record. It declares every selected unit's exact
identity, activation state, absolute install path, manifest location and installation source.
Loaders resolve no sibling checkout and guess no kind-specific directory.

## Installation modes

- installed units come from a pinned Git commit or a SHA-256-pinned archive and may be updated.
- development units come from one explicit absolute path. Their source path equals their install
  path and the updater returns development-unit without writing the tree.

A Git checkout is not implicitly development. A checkout installed at an exact commit uses
installed; only an explicitly selected editable path uses development.

Changing mode or path replaces the settings document atomically. The runtime publishes a
composition change after resolving the new document; loaders and UI do not poll.

## Unit kinds

The common identity covers plugin, sidecar, kit, contract, spec and service. Every unit has exact
kind, id and version identity. Version ranges are not part of this contract.

## Paths

Install and development paths are clean absolute paths. Symbolic links are rejected by the host
that reads the filesystem. Manifest paths are safe paths relative to the install path.

## Failure

Unknown fields, unknown enum values, relative paths, unpinned sources and duplicate exact unit
identities are rejected. No legacy settings reader, path fallback or implicit provider exists.
