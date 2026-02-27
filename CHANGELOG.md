# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/). This project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- API Gateway service (ApiGateway and ApiGatewayDeployment CRDs)
- Container Instances service (ContainerInstance CRD)
- OCI Compute Instance service (ComputeInstance CRD)
- OCI PostgreSQL Database service (PostgresDbSystem CRD)
- OCI Object Storage service (ObjectStorageBucket CRD)
- OCI Data Flow Application service (DataFlowApplication CRD)
- OCI Networking: InternetGateway, NatGateway, ServiceGateway, DRG, SecurityList, NetworkSecurityGroup, and RouteTable CRDs
- Autonomous Database: ECPU compute model support (computeModel and computeCount fields)
- OCI client interface injection across all service managers for improved testability
- Expanded unit test coverage across all service managers

### Changed
- UpdateRouteTable and UpdateSecurityList now always reconcile rules to match spec
- Networking CRD count expanded to 25 total CRDs

### Removed
- OCI Vault (Key Management) service removed entirely — no Vault CRDs or vendor packages remain
- OCI DevOps service removed
- OCI Service Mesh service removed

## [1.1.1] - 2022-05-23
### Added
- Removed 'FixedLogs' in OSOKLogger
- Pass fixed log information to OSOKLogger through context of a request
- Save retry token before sending request to control-plane

## [1.1.0] - 2022-04-27
### Added
- Support for Service Mesh Service
- Supports OLM version 0.20.0
- Supports operator bundle upgrade

## [1.0.0] - 2021-08-31 Initial Release
### Added
- Support for Autonomous Database Service
- Support for Oracle Streaming Service
- Support for MySQL DB System Service
- Supports OLM version 0.18.1
