# OpenID Connect Trust between AWS and Inventory

This document describes how to establish trust between an [OpenID
Connect](http://openid.net/connect/) IdP and your AWS account.

The benefits of establishing such a trust is that you don't need to maintain
static, long-lived credentials for the Inventory system when collecting AWS
resources.

Instead, by having trust between an OpenID Connect IdP and your AWS account you
use signed JWT tokens, which are exchanged for temporary, short-lived security
credentials when accessing AWS resources.

In this document we will be running the Inventory system within a Kubernetes
cluster, and will use Kubernetes as the OpenID Connect Provider, which is
trusted by AWS.

In this setup a signed [JWT](https://jwt.io/) token will be created for a
Kubernetes service account, which will later be exchanged for temporary security
credentials via the [AWS STS](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html)
service in order to access AWS resources.

# Requirements

You need a Kubernetes cluster with
[ServiceAccountIssuerDiscovery](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-issuer-discovery)
flag enabled.

The Inventory system will be deployed in the Kubernetes cluster. Check the
[deployment/kustomize](../deployment/kustomize) directory for sample
[kustomize](https://kustomize.io/) manifests, which you can use to deploy the
Inventory.

You also need to create an OpenID Connect (OIDC) identity provider in IAM. For
more details on how to do that, please refer to
[this documentation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html).

In order to find out the OpenID Connect Provider URL for your Kubernetes
cluster, execute the following command.

``` shell
kubectl get --raw /.well-known/openid-configuration
```

Sample response looks like this.

``` javascript
{
  "issuer": "https://foobar.example.org",
  "jwks_uri": "https://foobar.example.org/openid/v1/jwks",
  "response_types_supported": [
    "id_token"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256"
  ]
}
```

You should use the `issuer` field when creating the IdP in AWS as the Provider
URL.

Once you've created the Identity Provider in AWS you should also create a IAM
Web Identity Role with the respective Trust Policies and permissions. For more
details on how to do that, please refer to
[this documentation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-idp_oidc.html#idp_oidc_Create).

# Configuration

The AWS specific configuration used by the Inventory system resides in the `aws`
section of the [configuration file](../examples/config.yaml).

The AWS client used by the Inventory worker may be initialized either by using
static credentials defined in the [shared config and credentials
file](https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html), or via
[temporary security
credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
by using the AWS STS service.

When using temporary security credentials a signed [JWT](https://jwt.io/) token
is exchanged for short-lived security credentials. The way the JWT token is
retrived is configured via a _token retriever_.

The currently supported token retrievers, which can be set in the configuration
file are:

- `none`
- `kube_sa_token`
- `token_file`

When using the `none` token retriever the AWS client configuration will be
initialized using static credentials defined in the [shared config and
credentials
file](https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html).

When the token retriever is set to `kube_sa_token` the AWS client will be
initialized using a Web Identity Credentials Provider, which uses a Kubernetes
service account token. The Kubernetes service account token will be issued for
the specified user and audiences and with the set expiry duration for the STS
credentials.

This service account token is then exchanged for temporary security credentials
when accessing AWS services.

Example configuration with `kube_sa_token` looks like this.

``` yaml
aws:
  region: eu-central-1  # Frankfurt
  default_region: eu-central-1  # Frankfurt
  app_id: gardener-inventory  # Optional application specific identifier
  credentials:
    token_retriever: kube_sa_token
    kube_sa_token:
      kubeconfig: /path/to/kubeconfig
      namespace: inventory
      service_account: worker
      duration: 30m
      audiences:
        - gardener-inventory-playground
      role_arn: arn:aws:iam::account:role/name
      role_session_name: gardener-inventory-worker
```

When using the `token_file` retriever the AWS client is initialized using a Web
Identity Credentials Provider, which reads JWT tokens from a specified path.

Examples where `token_file` retriever is useful is with [service account token
projection](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#launch-a-pod-using-service-account-token-projection).

Example configuration which uses `token_file` looks like this.

``` yaml
aws:
  region: eu-central-1  # Frankfurt
  default_region: eu-central-1  # Frankfurt
  app_id: gardener-inventory  # Optional application specific identifier
  credentials:
    token_retriever: token_file
    token_file:
      path: /path/to/identity/token
      duration: 30m
      role_arn: arn:aws:iam::account:role/name
      role_session_name: gardener-inventory-worker
```

# References

Please refer to the following links for additional information on the topic.

- [What is OpenID Connect](https://openid.net/developers/how-connect-works/)
- [OpenID Connect Core 1.0 spec](https://openid.net/specs/openid-connect-core-1_0.html)
- [Temporary security credentials in IAM](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
- [Create an OpenID Connect (OIDC) identity provider in IAM](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html)
- [Create a role for OpenID Connect federation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-idp_oidc.html)
- [AWS Security Token Service API Reference](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html)
- [Kubernetes Service Account Token Projection](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#launch-a-pod-using-service-account-token-projection)
