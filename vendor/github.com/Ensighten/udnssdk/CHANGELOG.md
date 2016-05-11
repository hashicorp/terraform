# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [1.2.1] - 2016-06-13
### Fixed
* `omitempty` tags fixed for `ProbeInfoDTO.PoolRecord` & `ProbeInfoDTO.ID`
* Check `*http.Response` values for nil before access

## [1.2.0] - 2016-06-09
### Added
* Add probe detail serialization helpers

### Changed
* Flatten udnssdk.Response to mere http.Response
* Extract self-contained passwordcredentials oauth2 TokenSource
* Change ProbeTypes to constants

## [1.1.1] - 2016-05-27
### Fixed
* remove terraform tag for `GeoInfo.Codes`

## [1.1.0] - 2016-05-27
### Added
* Add terraform tags to structs to support mapstructure

### Fixed
* `omitempty` tags fixed for `DirPoolProfile.NoResponse`, `DPRDataInfo.GeoInfo`, `DPRDataInfo.IPInfo`, `IPInfo.Ips` & `GeoInfo.Codes`
* ProbeAlertDataDTO equivalence for times with different locations

### Changed
* Convert RawProfile to use mapstructure and structs instead of round-tripping through json
* CHANGELOG.md: fix link to v1.0.0 commit history

## [1.0.0] - 2016-05-11
### Added
* Support for API endpoints for `RRSets`, `Accounts`,  `DirectionalPools`, Traffic Controller Pool `Probes`, `Events`, `Notifications` & `Alerts`
* `Client` wraps common API access including OAuth, deferred tasks and retries

[Unreleased]: https://github.com/Ensighten/udnssdk/compare/v1.2.1...HEAD
[1.2.1]: https://github.com/Ensighten/udnssdk/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/Ensighten/udnssdk/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/Ensighten/udnssdk/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/Ensighten/udnssdk/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/Ensighten/udnssdk/compare/v0.0.0...v1.0.0
