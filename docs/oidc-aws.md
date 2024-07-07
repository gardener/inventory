# OpenID Connect Trust between AWS and Inventory

This document describes how to establish trust between an [OpenID
Connect](http://openid.net/connect/) IdP and your AWS account.

The benefits of establishing such a trust is that you don't need to maintain
static, long-lived credentials for the Inventory system when collecting AWS
resources.

Instead, by having trust between an OpenID Connect IdP and your AWS account you
use temporary, short-lived security credentials when accessing AWS resources.

In this document we will be running the Inventory system within a Kubernetes
cluster, and will use Kubernetes as the OpenID Connect Provider, which is
trusted by AWS.

In this setup a [JWT](https://jwt.io/) token will be created for a Kubernetes
service account, which will later be used to request temporary security
credentials via the [AWS
STS](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html) service
in order to access AWS resources.

# Requirements

You need a Kubernetes cluster with
[ServiceAccountIssuerDiscovery](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-issuer-discovery)
flag enabled.

The Inventory system system will be deployed in the Kubernetes cluster. Check
the [deployment/kustomize](../deployment/kustomize) directory for sample
[kustomize](https://kustomize.io/) manifests, which you can use to deploy the
Inventory.

You also need to create an OpenID Connect (OIDC) identity provider in IAM. For
more details on how to do that, please refer to [this
documentation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html).

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
details on how to do that, please refer to [this
documentation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-idp_oidc.html#idp_oidc_Create).

# Configuration

In order to exchange a Kubernetes service account token for temporary security
credentials when accessing AWS resources during Inventory collection we need to
provide the respective configuration.

The AWS specific configuration used by the Inventory system resides in the `aws`
config section.

Please refer to the [examples/config.yaml](../examples/config.yaml) file to view
the full configuration.

This snippet here contains just the AWS specific configuration, which should be
adjusted according to your setup.

``` yaml
aws:
  region: eu-central-1  # Frankfurt
  default_region: eu-central-1  # Frankfurt
  app_id: gardener-inventory  # Optional application specific identifier
  credentials:
    provider: kube_sa_token
    kube_sa_token:
      kubeconfig: /path/to/kubeconfig
      namespace: inventory
      service_account: inventory-worker
      duration: 30m
      audiences:
        - foo
        - bar
      role_arn: role-arn-goes-here
      role_session_name: role-session-name
```

The currently supported `aws.credentials.provider` values are:

- `default`
- `kube_sa_token`

When using the `default` credentials provider the AWS client configuration will
be initialized using the shared credentials file at `~/.aws/credentials`.

When the credentials provider is set to `kube_sa_token` the AWS client will be
initialized using a Web Identity Credentials Provider, which uses a Kubernetes
service account token. The Kubernetes service account token will be issued for
the specified user and audiences and with the configured expiry duration.

This service account token is then used for exchanging it for temporary security
credentials when accessing AWS services.

# References

Please refer to the following links for additional information on the topic.

- [What is OpenID Connect](https://openid.net/developers/how-connect-works/)
- [OpenID Connect Core 1.0 spec](https://openid.net/specs/openid-connect-core-1_0.html)
- [Temporary security credentials in IAM](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
- [Create an OpenID Connect (OIDC) identity provider in IAM](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html)
- [Create a role for OpenID Connect federation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-idp_oidc.html)
- [AWS Security Token Service API Reference](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html)
