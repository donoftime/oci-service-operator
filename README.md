# OCI Service Operator for Kubernetes

## Introduction

The OCI Service Operator for Kubernetes (OSOK) makes it easy to create, manage, and connect to Oracle Cloud Infrastructure (OCI) resources from a Kubernetes environment. Kubernetes users can simply install OSOK and perform actions on OCI resources using the Kubernetes API removing the need to use the [OCI CLI](https://docs.oracle.com/en-us/iaas/Content/API/Concepts/cliconcepts.htm) or other [OCI developer tools](https://docs.oracle.com/en-us/iaas/Content/devtoolshome.htm) to interact with a service API.

OSOK is based on the [Operator Framework](https://operatorframework.io/), an open-source toolkit used to manage Operators. It uses the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) library, which provides high-level APIs and abstractions to write operational logic and also provides tools for scaffolding and code generation for Operators.

**Services Supported**
1. [Autonomous Database Service](https://www.oracle.com/in/autonomous-database/) — [OSOK docs](docs/adb.md)
1. [Oracle Streaming Service](https://docs.cloud.oracle.com/iaas/Content/Streaming/Concepts/streamingoverview.htm) — [OSOK docs](docs/oss.md)
1. [MySQL DB System Service](https://www.oracle.com/mysql/) — [OSOK docs](docs/mysql.md)
1. [OCI Cache with Redis](https://www.oracle.com/cloud/cache/) — [OSOK docs](docs/redis.md)
1. [OpenSearch Service](https://www.oracle.com/cloud/search/) — [OSOK docs](docs/opensearch.md)
1. [Queue Service](https://www.oracle.com/cloud/queue/) — [OSOK docs](docs/queue.md)
1. [API Gateway](https://www.oracle.com/cloud/networking/api-gateway/) — [OSOK docs](docs/apigateway.md)
1. [NoSQL Database](https://www.oracle.com/database/nosql-cloud.html) — [OSOK docs](docs/nosql.md)
1. [Functions (FaaS)](https://www.oracle.com/cloud/cloud-native/functions/) — [OSOK docs](docs/functions.md)
1. [Container Instances](https://www.oracle.com/cloud/cloud-native/container-instances/) — [OSOK docs](docs/containerinstances.md)
1. [OCI Data Flow](https://www.oracle.com/big-data/data-flow/) — [OSOK docs](docs/dataflow.md)
1. [OCI Object Storage](https://www.oracle.com/cloud/storage/object-storage/) — [OSOK docs](docs/objectstorage.md)
1. [OCI PostgreSQL Database](https://www.oracle.com/cloud/database/) — [OSOK docs](docs/postgresql.md)

## Installation

See the [Installation](docs/installation.md#install-operator-sdk) instructions for detailed installation and configuration of OCI Service Operator for Kubernetes.

## Documentation

See the [Documentation](docs/README.md#oci-service-operator-for-kubernetes) for complete details on installation, security and service related configurations of OCI Service Operator for Kubernetes.

## Release Bundle

The OCI Service Operator for Kubernetes is packaged as Operator Lifecycle Manager (OLM) Bundle for making it easy to install in Kubernetes Clusters. The bundle can be downloaded as docker image using below command.

```
docker pull iad.ocir.io/oracle/oci-service-operator-bundle:1.1.9
```

## Samples

Samples for managing OCI Services/Resources using `oci-service-operator`, can be found [here](config/samples).

## Changes

See [CHANGELOG](CHANGELOG.md).

## Contributing
`oci-service-operator` project welcomes contributions from the community. Before submitting a pull request, please [review our contribution guide](./CONTRIBUTING.md).

## Security

Please consult the [security guide](./SECURITY.md) for our responsible security
vulnerability disclosure process.

## License

Copyright (c) 2021 Oracle and/or its affiliates.

Released under the Universal Permissive License v1.0 as shown at <https://oss.oracle.com/licenses/upl/>.
