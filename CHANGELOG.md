# Changelog

## [1.10.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.9.0...v1.10.0) (2026-01-31)


### Features

* standardize log format to pipe-delimited key=value style ([#75](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/75)) ([076398d](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/076398d1d1d506b3b495c507c6511f388d51f0a7))

## [1.9.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.8.0...v1.9.0) (2026-01-31)


### Features

* add --dry-run flag to preview changes without modifying kubeconfig ([#71](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/71)) ([7f46102](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/7f461020468afd58b868004a69283e4ceb92069a))

## [1.8.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.7.0...v1.8.0) (2026-01-10)


### Features

* add token expiration checking to avoid unnecessary regeneration ([#67](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/67)) ([71f08d5](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/71f08d57b409f33e92f500fe46e088e201916467))

## [1.7.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.6.0...v1.7.0) (2025-12-29)


### Features

* add --config flag to specify custom kubeconfig path ([#62](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/62)) ([269938f](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/269938f7f81b7dc75a8fbf5d8a869a610587efc3))

## [1.6.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.5.0...v1.6.0) (2025-12-29)


### Features

* respect KUBECONFIG environment variable for kubeconfig file resolution ([#56](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/56)) ([0973341](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/0973341401af2f7a5f4018cb4af7386e560e259d))

## [1.5.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.4.0...v1.5.0) (2025-12-19)


### Features

* add --insecure-skip-tls-verify flag ([#53](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/53)) ([93489c7](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/93489c79ec9cc929afd5c9d78105f83de8ae7cd6))

## [1.4.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.3.0...v1.4.0) (2025-12-19)


### Features

* add --cluster flag for selective cluster token updates ([#46](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/46)) ([65429c6](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/65429c6ee23c1660511d6b0a9a93b5062a748ad2))

## [1.3.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.2.1...v1.3.0) (2025-12-18)


### Features

* log file path when created backup of kubeconfig file ([#43](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/43)) ([f0ab2b1](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/f0ab2b11049d8f60fe17fd86608487e3564751b2))

## [1.2.1](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.2.0...v1.2.1) (2025-12-16)


### Bug Fixes

* **kubeconfig:** remove trailing slash from Rancher URL to prevent double slashes ([f112fec](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/f112fec826fce9a4b0a43ff8ea58c4b6b015bc61))

## [1.2.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.1.0...v1.2.0) (2025-12-12)


### Features

* add Windows cross-platform path support ([16946fb](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/16946fb409f675b008c93cfd4660670cefc0652f))

## [1.1.0](https://github.com/chenwei791129/rancher-kubeconfig-updater/compare/v1.0.0...v1.1.0) (2025-12-11)


### Features

* add CLI flags for Rancher credentials ([e70daad](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/e70daadec00ef798dab0eee7c4689520281d585a))

## 1.0.0 (2025-12-10)


### Features

* add GitHub Actions workflow for automated release process ([9313002](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/93130029ffb59a2dec74e230dbd77c2334fe867e))
* Add support for multiple Rancher authentication types ([d70ae36](https://github.com/chenwei791129/rancher-kubeconfig-updater/commit/d70ae36fa4001cf7f7dd7ab42d7538ec3e20ee2c))
