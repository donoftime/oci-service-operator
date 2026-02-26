# OCI Data Flow Application

## Overview

[OCI Data Flow](https://www.oracle.com/big-data/data-flow/) is a fully managed Apache Spark service that lets you run Spark applications at any scale without deploying or managing infrastructure. OSOK's `DataFlowApplication` CRD enables Kubernetes-native management of OCI Data Flow Applications.

## Prerequisites

Before creating a `DataFlowApplication` resource, ensure:

1. **IAM Policies**: The OCI service account used by OSOK must have policies to manage Data Flow applications in the target compartment:
   ```
   allow group <group> to manage dataflow-family in compartment <compartment>
   allow group <group> to read buckets in compartment <compartment>
   allow group <group> to manage objects in compartment <compartment>
   ```

2. **Object Storage**: Your application file (`.py`, `.jar`, `.sql`) must be uploaded to an OCI Object Storage bucket and accessible via an OCI URI.

3. **Spark Application File**: Upload your application to Object Storage:
   ```bash
   oci os object put --bucket-name my-bucket --name spark_app.py --file ./spark_app.py
   ```

## DataFlowApplication CRD

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | OCID | Yes | OCID of the compartment to create the application in |
| `displayName` | string | Yes | User-friendly display name |
| `language` | string | Yes | Spark language: `PYTHON`, `SCALA`, `JAVA`, or `SQL` |
| `driverShape` | string | Yes | VM shape for the driver (e.g. `VM.Standard2.1`, `VM.Standard.E3.Flex`) |
| `executorShape` | string | Yes | VM shape for executors (e.g. `VM.Standard2.1`) |
| `numExecutors` | int | Yes | Number of executor VMs (minimum: 1) |
| `sparkVersion` | string | Yes | Spark version (e.g. `3.2.1`) |
| `fileUri` | string | No | OCI URI to application file (format: `oci://bucket@namespace/path/to/app.py`). Not required for SQL language. |
| `className` | string | No | Java/Scala main class name |
| `arguments` | []string | No | Command-line arguments for the application |
| `configuration` | map[string]string | No | Spark configuration key-value pairs |
| `description` | string | No | User-friendly description |
| `logsBucketUri` | string | No | OCI URI for the logs bucket |
| `warehouseBucketUri` | string | No | OCI URI for the Hive warehouse bucket |
| `archiveUri` | string | No | OCI URI for an archive with custom dependencies |
| `id` | OCID | No | OCID of an existing application to bind to (skips creation) |

**Language enum values:**
- `PYTHON` — Python application (`.py` file)
- `SCALA` — Scala application (`.jar` file)
- `JAVA` — Java application (`.jar` file)
- `SQL` — SQL application (no `fileUri` needed)

**Shape examples:**
- `VM.Standard2.1` — 1 OCPU, 15 GB RAM
- `VM.Standard.E3.Flex` — Flexible shape (requires shape config)
- `VM.Standard2.4` — 4 OCPU, 60 GB RAM

**fileUri format:** `oci://bucket-name@namespace/path/to/application.py`

### Status Fields

| Field | Description |
|-------|-------------|
| `status.ocid` | OCID of the managed Data Flow Application |
| `status.createdAt` | Timestamp when the resource was created |
| `status.conditions` | List of conditions reflecting the current state |

**Condition types:**
- `Active` — Application is active and ready
- `Failed` — Application creation or update failed
- `Provisioning` — Application is being created

## Examples

### Python Application

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: DataFlowApplication
metadata:
  name: my-python-app
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: MyPythonSparkApp
  language: PYTHON
  driverShape: "VM.Standard2.1"
  executorShape: "VM.Standard2.1"
  numExecutors: 2
  sparkVersion: "3.2.1"
  fileUri: "oci://my-bucket@my-namespace/scripts/my_app.py"
  arguments:
    - "--input"
    - "oci://data-bucket@my-namespace/input.csv"
  configuration:
    spark.executor.memory: "4g"
    spark.driver.memory: "2g"
  logsBucketUri: "oci://logs-bucket@my-namespace"
  description: "My Python Spark application"
```

### Scala/Java Application

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: DataFlowApplication
metadata:
  name: my-scala-app
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: MyScalaSparkApp
  language: SCALA
  driverShape: "VM.Standard2.2"
  executorShape: "VM.Standard2.2"
  numExecutors: 3
  sparkVersion: "3.2.1"
  fileUri: "oci://my-bucket@my-namespace/jars/my_app.jar"
  className: "com.example.MySparkApp"
  description: "My Scala Spark application"
```

### Apply the resource

```bash
kubectl apply -f my-dataflow-app.yaml
kubectl get dataflowapplication my-python-app -o yaml
kubectl describe dataflowapplication my-python-app
```

### Check status

```bash
kubectl get dataflowapplications
# NAME             DISPLAYNAME          LANGUAGE   STATUS   OCID                              AGE
# my-python-app    MyPythonSparkApp     PYTHON     Active   ocid1.dataflowapplication.oc1..   5m
```

## Deletion

When a `DataFlowApplication` resource is deleted, OSOK deletes the corresponding OCI Data Flow Application:

```bash
kubectl delete dataflowapplication my-python-app
```

## Binding to Existing Applications

To manage an existing OCI Data Flow Application without creating a new one, set the `id` field:

```yaml
spec:
  id: ocid1.dataflowapplication.oc1..aaaaaaaaxxx
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: ExistingApp
  language: PYTHON
  driverShape: "VM.Standard2.1"
  executorShape: "VM.Standard2.1"
  numExecutors: 1
  sparkVersion: "3.2.1"
```

OSOK will bind to the existing application rather than creating a new one.
