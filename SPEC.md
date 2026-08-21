# Soksak composition contract 0.0.1

The file at identity-home/settings.json records installed plugins, sidecars, kits and their exact
bindings. The release identity uses ~/.soksak/settings.json. Other identities use their own home.

## Explicit component kinds

Plugins, sidecars and kits have separate arrays, types, status and public commands. There is no
public generic component kind.

- plugins are user-facing features with plugin.json, enabled and development state;
- sidecars are process or library dependencies with their own sidecar manifest, enabled and
  development state;
- kits are package or runtime dependencies with their own package manifest, enabled and development
  state;
- contracts and specifications are exact references and conformance material, not installations;
- host services are core build components.

Internal installer helpers may share transaction code. They do not change the public names or
settings shape.

## Installation records

Every plugin, sidecar and kit record has exact id and version, enabled, development, an absolute
install path, its kind-specific manifest path and acquisition source. Development is a boolean and
is independent from acquisition source. The updater never writes a record whose development value
is true.

Source records an exact Git commit; a release repository and exact commit plus a SHA-256 archive;
or an absolute local path. Relative paths, symbolic links and unpinned sources are rejected.

One settings composition contains at most one version for each plugin id, sidecar id or kit id.
Several plugins may bind to the same exact sidecar or kit installation. Use is derived from the
current graph rather than a stored reference count.

## Bindings

A binding has an explicitly typed plugin, sidecar or kit consumer and provider, a named requirement
and exact versions. Exactly one kind is present at each endpoint. No provider is selected by
directory order, install order, naming convention or fallback.

## Atomic changes

The first settings document is generation one. Every replacement uses compare-and-swap against the
current generation and advances by one. After validation and atomic replacement the backend emits
one composition.changed event. Loaders and UI do not poll.

## Repository boundary conformance

Executable code, tests, tasks and scripts do not read or execute another plugin, sidecar, kit,
contract or service source tree. Declarative package dependencies and public contract references are
allowed. Source symbolic links are rejected. Registry and system-test repositories may list ids as
data but do not build component source trees.
