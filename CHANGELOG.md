<!-- markdownlint-disable no-duplicate-heading -->
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://rats.org/spec/v2.0.0.html).

## [0.3.1] - 2025-11-13

### Changed

* all CLI flags disabled by default

## [0.3.0] - 2025-11-13

### Changed

* fixed strict semver filter if all input data not semver
* **Removed** `Options.ReleaseOnly` and flag `--release-only`,
  now use only ``Options.Format` or flag `--format` instead
* flag `--semver` disabled by default
* flag `--deduplicate` disabled by default
* removed `go.work` and made cmd part of single project

## [0.2.0] - 2025-09-19

### Added

* Redesigned initial release

## [0.1.0] - 2025-09-16

### Added

* Initial prototype release

<!-- links -->

[0.2.0]: <https://github.com/WoozyMasta/rats/compare/v0.1.0...v0.2.0>
[0.1.0]: <https://github.com/WoozyMasta/rats/releases/tag/v0.1.0>
