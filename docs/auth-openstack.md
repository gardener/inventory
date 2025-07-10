# OpenStack

Users of the Inventory can provide OpenStack service specific credentials, which
are then used against the specified OpenStack Keystone service.  Multiple
credentials can be provided per service, with the unique property being
project_id, which is expected to be unique globally. With this setup, users can
collect resources from multiple regions/clusters, as long as the credentials are
configured properly.

Make sure the credentials you provide have the roles needed to access the
respective resources. If there are resources with different policies in the same
service type that need to be collected, you can list different credentials for
both. This might cause an error in inventory logs, but won't stop the collection
process.

A full example can be found at `./examples/config.yaml`

Example configuration for OpenStack collection.

``` yaml
# OpenStack specific configuration
openstack:
  is_enabled: true

  # The `credentials' section provides named credentials, which are used by the
  # various OpenStack services. The currently supported authentication
  # mechanisms are `password' for username and password, `app_credentials' for
  # Application Credentials and `vault_secret' for credentials provided by a
  # Vault secret..
  credentials:
    # Example of using username/password for authentication
    foo:
      domain: <domain>
      auth_endpoint: <endpoint>
      project: <project_name>
      region: <region>
      authentication: password
      password:
        username: "<username>"
        password_file: "<path-to-password-file>"
    # Example of using Application Credentials for authentication
    bar:
      domain: <domain>
      auth_endpoint: <endpoint>
      project: <project_name>
      region: <region>
      authentication: app_credentials
      app_credentials:
        app_credentials_id: "<app-id>"
        app_credentials_secret_file: "<path-to-secret-file>"

  # OpenStack services configuration
  services:
    # Used for collecting OpenStack Servers
    compute:
      use_credentials:
        - foo
        - bar
    # Used for collecting OpenStack Networks and Subnets
    network:
      use_credentials:
        - foo
    # Used for collecting OpenStack Containers and Objects
    object_storage:
      use_credentials:
        - bar
    # Used for collecting OpenStack LoadBalancers
    load_balancer:
      use_credentials:
        - foo
        - bar
    # Used for collecting OpenStack Project metadata
    identity:
      use_credentials:
        - foo
        - bar
```

The supported authentication methods when configuring named credentials are
`password`, `app_credentials` and `vault_secret`.

The `services` section is used for configuring collection from the respective
OpenStack service. Each service may specify one or more named credentials, which
will be used during collection.

In order to configure OpenStack credentials from a Vault secret we first need to
configure a Vault server in the Inventory config. Example Vault configuration
with a single Vault server is provided below.

``` yaml
# Vault settings.
vault:
  is_enabled: true

  # The Vault servers for which API clients will be created.
  servers:
    # Dev Vault server example
    vault-dev:
      # Endpoint of the Vault server
      endpoint: http://localhost:8200/

      # Optional TLS settings
      # tls:
      #   ca_cert: /path/to/ca.crt
      #   ca_cert_bytes: "PEM-encoded CA bundle"
      #   ca_path: /path/to/ca/cert/files
      #   client_cert: /path/to/client.crt
      #   client_key: /path/to/client.key
      #   tls_server_name: SNI-host
      #   insecure: false

      # The supported Auth Methods are `token' and `jwt'.
      auth_method: token

      # Auth settings when using `token' auth method.
      token_auth:
        token_path: /path/to/my/token
```

Please refer to the [examples/config.yaml](../examples/config.yaml) file for
additional details and examples on how to configure Vault servers with different
Authentication Methods (e.g. [JWT Auth
Method](https://developer.hashicorp.com/vault/docs/auth/jwt)) as well.

Once we configure Vault servers we can then configure OpenStack named
credentials, which use Vault secrets. The following example shows a single named
credential configured from a Vault secret.

``` yaml
# OpenStack specific configuration
openstack:
  is_enabled: true

  credentials:
    # Example of using Vault secret for OpenStack credentials
    foo:
      domain: my-domain
      auth_endpoint: https://example.org/v3
      project: my-project-name
      region: region-1
      authentication: vault_secret
      vault_secret:
        # Must refer to a Vault server defined in `vault.servers'
        server: vault-dev

        # Mount point of a KV v2 secret engine
        secret_engine: kvv2

        # Path to the secret
        secret_path: path/to/my/secret

  # OpenStack services configuration
  services:
    # Used for collecting OpenStack Servers
    compute:
      use_credentials:
        - foo
    # Used for collecting OpenStack Networks and Subnets
    network:
      use_credentials:
        - foo
    # Used for collecting OpenStack Containers and Objects
    object_storage:
      use_credentials:
        - foo
    # Used for collecting OpenStack LoadBalancers
    load_balancer:
      use_credentials:
        - foo
    # Used for collecting OpenStack Project metadata
    identity:
      use_credentials:
        - foo
```

When using `vault_secret` for configuring OpenStack credentials the Vault secret
may provide a `v3password` (username and password pair) or
`v3applicationcredential` ([Application
Credentials](https://docs.openstack.org/keystone/queens/user/application_credentials.html))
secret.

The following example creates a `v3password` secret in Vault for username and
password authentication.

``` yaml
vault kv put kvv2/path/to/my/secret \
      kind=v3password \
      username=my-username \
      password=my-p4ssw0rd
```

This example creates a `v3applicationcredential` secret in Vault for Application
Credentials authentication.

``` yaml
vault kv put kvv2/path/to/my/secret \
      kind=v3applicationcredential \
      application_credential_id=app-id \
      application_credential_secret=app-s3cr37
```
