# Changelog

## [unreleased]

### Changed

- Add support for Kubernetes 1.23.
- Update GRPC dependencies.
- Update Dex dependencies.
- Drop support for networkingv1beta1.Ingress in favor of networkingv1.Ingress.

## [0.1.0] - 2020-05-05

### Added

- Auth Backend CRD.
- Secure using ingress based annotations.
- Oauth2-proxy implementation as a secure proxy.
- Dex auth backend implementation.
- Main application with a Kubernetes controller.
- Prometheus metrics.
- Initial documentation.

[unreleased]: https://github.com/slok/bilrost/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/slok/bilrost/releases/tag/v0.1.0
