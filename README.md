# Provider SOS

`provider-exoscale-sos` is a [Crossplane](https://crossplane.io/) provider
sos that is built using [Upjet](https://github.com/crossplane/upjet) code
generation tools and exposes XRM-conformant managed resources for the SOS
API.

## Getting Started

This sos serves as a starting point for generating a new [Crossplane Provider](https://docs.crossplane.io/latest/packages/providers/) using the [`upjet`](https://github.com/crossplane/upjet) tooling. Please follow the guide linked below to generate a new Provider:

https://github.com/crossplane/upjet/blob/main/docs/generating-a-provider.md

## Developing

Run code-generation pipeline:
```console
go run cmd/generator/main.go "$PWD"
```

Run against a Kubernetes cluster:

```console
make run
```

Build, push, and install:

```console
make all
```

Build binary:

```console
make build
```

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
```Bash
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
TODO

### Configure crossplane providers

```Bash
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
```Bash
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

## End to End test
```bash
$> export EXOSCALE_API_KEY=<your-api-key>
$> export EXOSCALE_API_SECRET=<your-api-secret>
$> export EXOSCALE_SOS_ENDPOINT=<sos-endpoint>

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

For filing bugs, suggesting improvements, or requesting new features, please
open an [issue](https://github.com/exoscale/provider-exoscale-sos/issues).
