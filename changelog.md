
## Changes in version 1.32.0

### New Features

* tekton: reconstruct owning LighthouseJob when rerun parents are absent (Maximilien Raulic)
* tekton: assemble reconstructed LighthouseJob with empty status (Maximilien Raulic)
* tekton: build a LighthouseJobSpec from rerun PipelineRun metadata (Maximilien Raulic)

### Bug Fixes

* tekton: ensure canonical metadata parity across rerun paths (Maximilien Raulic)
* tekton: harden LighthouseJob reconstruction (Maximilien Raulic)
* tekton: set rerun LighthouseJob StartTime (Maximilien Raulic)
* tekton: make rerun LighthouseJob creation idempotent (Maximilien Raulic)
* tekton: reset status on cloned LighthouseJob to avoid stale terminal state (Maximilien Raulic)

### Code Refactoring

* tekton: extract rerun parent resolution (Maximilien Raulic)

### Documentation

* tekton: document rerun LighthouseJob reconstruction (Maximilien Raulic)

### Chores

* fmt: fix unrelated gofmt and goimports issues (Maximilien Raulic)
