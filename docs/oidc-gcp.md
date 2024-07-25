# OpenID Connect Trust between GCP and Inventory

Workload identity federation is a secure mechanism allowing access of GCP services withough service accounts key. The latter is unsecure and potentially dangarous authentication mechanism. Instead of using service account keys to authenticate, the requesting client presents an identity tokens which after a successfull validation on service side is exchanged to a short-lived access token.

In inventory case there is a need to establish trust between the inventory service account and GKE clusters using GCP "Workload Identity Federation" concepts and more concretly "Workload Identity Federation with Kubernetes". The concrete implementation is based on RFC 8693 OAuth 2.0 Token Exchange standard.

Inventory identity when running in a K8S cluster is carried by OIDC identiy tokens issued by the K8S cluster. To allow trust between a GKE cluster and the inventory workload, the inventory service accoint tokens need to be excahnged with a short lived access tokens carrying an identity issued by GCP.

As an example here is a token of an inventory pod instance with subject "system:serviceaccount:namespace:inventory".

```json
{
  "aud": [
    "inventory"
  ],
  "exp": ...,
  "iat": ...,
  "iss": "https://public end point of the oidc issuer",
  "jti": "...",
  "kubernetes.io": {
    "namespace": "namespace",
    "serviceaccount": {
      "name": "worker",
      "uid": "..."
    }
  },
  "sub": "system:serviceaccount:namespace:inventory"
}
```

This token is presented to SecurityTokenService endpoint as part of a `token-exchange` request :

```
uri: https://sts.googleapis.com/v1/token
method: POST
...
grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&audience=//iam.googleapis.com/projects/<PROJECT ID>/locations/global/workloadIdentityPools/<Workload Identity Pool>/providers/<PROVIDER>
&scope=https://www.googleapis.com/auth/iam
&requested_token_type=urn:ietf:params:oauth:token-type:access_token
&subject_token=....​⬤
```

After successful verification it shall be replaced with a short lived access token.

```json
{
  "azp": "...",
  "aud": "...",
  "sub": "...",
  "scope": "https://www.googleapis.com/auth/sqlservice.login https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/compute https://www.googleapis.com/auth/appengine.admin https://www.googleapis.com/auth/userinfo.email openid",
  "exp": "...",
  "expires_in": "...",
  "email": "service_account@some_project_id.iam.gserviceaccount.com",
  "email_verified": "true",
  "access_type": "online"
}
```

The user denoted by such token is valid for GCP and can access services in accoring to the granted roles and permissions.

A key role in verifying the inventory token plays the "Provider" configuraiton attached to the "Workload Identity Federation Pool". In our example here, the Provider can enforce a number of conditions, additional to the token validition. For example,

```bash
assertion.sub == system:serviceaccount:namespace:inventory && assertion['kubernetes.io']['namespace'] in ['namespace']"
```

Finally, when the access tokens are used to perform operations on a GKE cluster, the standard K8S RBAC model is used to grant the required permissions.

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: inventory-viewer
rules:
- apiGroups:
  resources:
  - pods
  verbs:
  - get
  - list
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: inventory-viewer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: inventory-viewer
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: service_account@some_project_id.iam.gserviceaccount.com
```

The concrete configurations to achieve trust between a GKE cluster and an external K8S cluster are thoroughly described in the GCP documentation, with the following references.

## References

- [RFC 8693 OAuth 2.0 Token Exchange](https://www.rfc-editor.org/rfc/rfc8693)
- [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation)
- [Configure Workload Identity Federation with Kubernetes](https://cloud.google.com/iam/docs/workload-identity-federation-with-kubernetes)
