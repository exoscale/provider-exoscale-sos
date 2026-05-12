# Provider Exoscale SOS

`provider-exoscale-sos` is a [Crossplane](https://crossplane.io/) provider
built using [Upjet](https://github.com/crossplane/upjet) code generation tools.
It exposes XRM-conformant managed resources for the
[Exoscale SOS (Simple Object Storage)](https://www.exoscale.com/object-storage/)
API, enabling you to manage S3-compatible object storage declaratively from
Kubernetes.

> **Note:** This provider is generated from the [AWS S3 Terraform provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket).
> Exoscale SOS implements the S3-compatible API, so most common operations work
> seamlessly. A small number of AWS-specific features (such as certain IAM ARN
> formats or AWS-proprietary storage classes) are not available on Exoscale SOS —
> the relevant resources or fields are clearly noted where applicable.

This provider is designed to be installed **alongside
[provider-exoscale](https://github.com/exoscale/provider-exoscale)**, which
manages IAM roles and other Exoscale resources needed for advanced SOS
configurations such as cross-zone replication.

## Supported Resources

| Resource | Description |
|----------|-------------|
| **Bucket** | Create and manage SOS buckets |
| **BucketACL** | Manage bucket access control lists |
| **BucketCORSConfiguration** | Configure cross-origin resource sharing rules |
| **BucketVersioning** | Enable or suspend object versioning |
| **BucketReplicationConfiguration** | Set up cross-zone bucket replication |

Ready-to-use example manifests for all supported resources are available in the
[`examples-generated/`](examples-generated/) directory.

## Getting Started

### Prerequisites

- An existing Kubernetes cluster
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) installed and configured
- [Helm](https://helm.sh/docs/intro/install/) installed
- An [Exoscale](https://portal.exoscale.com/register) account with API credentials

## Example

### Install Crossplane

```bash
$> helm repo add crossplane-stable https://charts.crossplane.io/stable
$> helm repo update

$> helm install crossplane crossplane-stable/crossplane \
   --namespace crossplane-system \
   --create-namespace

$> kubectl wait deployment crossplane \
   --namespace crossplane-system \
   --for=condition=Available \
   --timeout=120s
```

### Install exoscale provider

```bash
export PROVIDER_EXOSCALE_VERSION=v0.1.0

$> cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-exoscale
spec:
  package: xpkg.upbound.io/exoscale/provider-exoscale:$PROVIDER_EXOSCALE_VERSION
EOF

$> kubectl wait provider/provider-exoscale \
   --for=condition=Healthy \
   --timeout=120s
```

### Install exoscale provider sos

```bash
export PROVIDER_EXOSCALE_SOS_VERSION=v0.1.0

$> cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-exoscale-sos
spec:
  package: xpkg.upbound.io/exoscale/provider-exoscale-sos:$PROVIDER_EXOSCALE_SOS_VERSION
EOF

$> kubectl wait provider/provider-exoscale-sos \
   --for=condition=Healthy \
   --timeout=120s
```

### Configure crossplane providers

Exoscale SOS is **endpoint-specific**: each zone exposes its own S3-compatible endpoint. Create one `ClusterProviderConfig` per zone, each referencing a secret that embeds the zone's endpoint URL.

```bash
$> export EXOSCALE_API_KEY=<your-api-key>
$> export EXOSCALE_API_SECRET=<your-api-secret>

$> kubectl create secret generic exoscale-credentials-ch-gva-2 \
   --namespace crossplane-system \
   --from-literal=credentials="{\"key\": \"$EXOSCALE_API_KEY\", \"secret\": \"$EXOSCALE_API_SECRET\", \"endpoint\": \"https://sos-ch-gva-2.exo.io\"}"

$> kubectl create secret generic exoscale-credentials-at-vie-1 \
   --namespace crossplane-system \
   --from-literal=credentials="{\"key\": \"$EXOSCALE_API_KEY\", \"secret\": \"$EXOSCALE_API_SECRET\", \"endpoint\": \"https://sos-at-vie-1.exo.io\"}"

$> kubectl create secret generic exoscale-credentials \
   --namespace crossplane-system \
   --from-literal=credentials="{\"key\": \"$EXOSCALE_API_KEY\", \"secret\": \"$EXOSCALE_API_SECRET\"}"

$> cat <<EOF | kubectl apply -f -
apiVersion: sos.m.exoscale.ch/v1beta1
kind: ClusterProviderConfig
metadata:
  name: sos-ch-gva-2
spec:
  credentials:
    source: Secret
    secretRef:
      name: exoscale-credentials-ch-gva-2
      namespace: crossplane-system
      key: credentials
EOF

$> cat <<EOF | kubectl apply -f -
apiVersion: sos.m.exoscale.ch/v1beta1
kind: ClusterProviderConfig
metadata:
  name: sos-at-vie-1
spec:
  credentials:
    source: Secret
    secretRef:
      name: exoscale-credentials-at-vie-1
      namespace: crossplane-system
      key: credentials
EOF

$> cat <<EOF | kubectl apply -f -
apiVersion: exoscale.m.exoscale.ch/v1beta1
kind: ClusterProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: exoscale-credentials
      namespace: crossplane-system
      key: credentials
EOF
```

### Simple Leader-Follower Setup

The following sets up cross-zone replication from a source bucket in `ch-gva-2` to a destination bucket in `at-vie-1`. This example uses `provider-exoscale` to create the IAM role that authorizes the replication operation.

```bash
export SOURCE=bucket-src-$(date +%N)
export DESTINATION=bucket-dest-$(date +%N)

$> cat <<EOF | kubectl apply -f -
apiVersion: sos.sos.m.exoscale.ch/v1alpha1
kind: Bucket
metadata:
  name: $SOURCE
  namespace: crossplane-system
spec:
  forProvider:
    objectLockEnabled: true
  providerConfigRef:
    kind: ClusterProviderConfig
    name: sos-ch-gva-2
EOF

$> cat <<EOF | kubectl apply -f -
apiVersion: sos.sos.m.exoscale.ch/v1alpha1
kind: Bucket
metadata:
  name: $DESTINATION
  namespace: crossplane-system
spec:
  forProvider:
    objectLockEnabled: true
  providerConfigRef:
    kind: ClusterProviderConfig
    name: sos-at-vie-1
EOF

$> cat <<EOF | kubectl apply -f -
apiVersion: sos.sos.m.exoscale.ch/v1alpha1
kind: BucketVersioning
metadata:
  name: $SOURCE
  namespace: crossplane-system
spec:
  forProvider:
    bucketRef:
      name: $SOURCE
      namespace: crossplane-system
    versioningConfiguration:
      - status: Enabled
  providerConfigRef:
    kind: ClusterProviderConfig
    name: sos-ch-gva-2
EOF

$> kubectl wait bucketversioning.sos.sos.m.exoscale.ch/$SOURCE \
   --namespace crossplane-system \
   --for=condition=Ready \
   --timeout=120s

$> cat <<EOF | kubectl apply -f -
apiVersion: sos.sos.m.exoscale.ch/v1alpha1
kind: BucketVersioning
metadata:
  name: $DESTINATION
  namespace: crossplane-system
spec:
  forProvider:
    bucketRef:
      name: $DESTINATION
      namespace: crossplane-system
    versioningConfiguration:
      - status: Enabled
  providerConfigRef:
    kind: ClusterProviderConfig
    name: sos-at-vie-1
EOF

$> kubectl wait bucketversioning.sos.sos.m.exoscale.ch/$DESTINATION \
   --namespace crossplane-system \
   --for=condition=Ready \
   --timeout=120s

$> cat <<EOF | kubectl apply -f -
apiVersion: iam.exoscale.m.exoscale.ch/v1alpha1
kind: IAMRole
metadata:
  name: sos-replication-$SOURCE-$DESTINATION
  namespace: crossplane-system
spec:
  forProvider:
    name: sos-replication:$SOURCE:$DESTINATION
    description:  "sos bucket replication between $SOURCE and $DESTINATION"
    editable: true
    policy:
      defaultServiceStrategy: deny
      services:
        sos:
          type: rules
          rules:
            - action: allow
              expression: "parameters.bucket == '$SOURCE' && (operation in ['get-object'])"
            - action: allow
              expression: "parameters.bucket == '$DESTINATION' && (operation.startsWith('put-object') || operation.startsWith('delete-object') || operation.startsWith('abort-multipart-upload'))"
EOF

$> kubectl wait iamrole.iam.exoscale.m.exoscale.ch/sos-replication-$SOURCE-$DESTINATION \
   --namespace crossplane-system \
   --for=condition=Ready \
   --timeout=120s

$ export IAMROLE=$(kubectl get iamrole.iam.exoscale.m.exoscale.ch/sos-replication-$SOURCE-$DESTINATION -n crossplane-system -o yaml | yq .status.atProvider.id -r)

$> cat <<EOF | kubectl apply -f -
apiVersion: sos.sos.m.exoscale.ch/v1alpha1
kind: BucketReplicationConfiguration
metadata:
  name: sos-replication-$SOURCE-$DESTINATION
  namespace: crossplane-system
spec:
  forProvider:
    bucketRef:
      name: $SOURCE
      namespace: crossplane-system
    role: "arn:aws:iam::third-party:$IAMROLE"
    rule:
      - id: multi-zone
        status: Enabled
        priority: 1
        filter:
          - prefix: ""
        deleteMarkerReplication:
          - status: Enabled
        destination:
          - bucket: arn:aws:s3:::$DESTINATION
  providerConfigRef:
    kind: ClusterProviderConfig
    name: sos-ch-gva-2
EOF

$> exo storage upload -r ./ sos://$SOURCE/
$> exo storage list sos://$DESTINATION
```

## Developing

> Based on the [Upjet documentation](https://github.com/crossplane/upjet/tree/main/docs).

Run the code-generation pipeline:

```bash
$> make generate
```

Run the provider locally against an existing Kubernetes cluster:

```bash
$> make run
```

Check deployed resources:

```bash
$> watch kubectl get managed -A
```

### Updating Examples and Tests

When making changes to resource definitions in the `apis/` directory, make sure to review and update the end-to-end test manifests in [`cluster/test/`](cluster/test/) to cover the changes.

## End-to-End Tests

```bash
$> export EXOSCALE_API_KEY=<your-api-key>
$> export EXOSCALE_API_SECRET=<your-api-secret>
$> export EXOSCALE_SOS_ENDPOINT=<sos-endpoint>  # e.g. https://sos-ch-gva-2.exo.io

$> mkdir -p .work
$> cat > .work/uptest_datasource.yaml << EOF
suffix: local
EOF

$> make e2e \
   PROVIDER_NAME=provider-exoscale-sos \
   UPTEST_EXAMPLE_LIST=$(find cluster/test/*/*.yaml | tr '\n' ',') \
   UPTEST_CLOUD_CREDENTIALS="{\"key\": \"$EXOSCALE_API_KEY\", \"secret\": \"$EXOSCALE_API_SECRET\", \"endpoint\": \"$EXOSCALE_SOS_ENDPOINT\"}" \
   UPTEST_DATASOURCE_PATH=./.work/uptest_datasource.yaml
```

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please open an [issue](https://github.com/exoscale/provider-exoscale-sos/issues).
