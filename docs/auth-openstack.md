# Authn/Authz
Users of the Inventory can provide OpenStack service specific credentials,
which are then used against the specified OpenStack Keystone service.
Multiple credentials can be provided per service, with the unique property
being project_id, which is expected to be unique globally. With this setup,
users can collect resources from multiple regions/clusters, as long as the
credentials are configured properly.

Make sure the credentials you provide have the roles needed to access the 
respective resources. If there are resources with different policies in the 
same service type that need to be collected, you can list different credentials
for both. This might cause an error in inventory logs, but won't stop
the collection process.

A full example can be found at `./examples/config.yaml`

Example credentials section:
```
credentials:
  local:
    authentication: password
    password:
      username: <your username>
      password_file: <path-to-file-with-password>
  named_credentials_myproject:
    authentication: app_credentials
    app_credentials:
      app_credentials_id: <id>
      app_credentials_name: <credentials_name>
      app_credentials_secret_file: <secret>
```

The two supported credential types are `password` and `app_credentials`.

The service section must also be configured.
Example services section populated for the network service:

```
services:
  network:
    - use_credentials: named_credentials_myproject #defined above
      domain: <domain>
      region: <region>
      project: <project name>
      project_id: <project id of the project above>
      auth_endpoint: https://<keystone-url>
```

Something to note here is, that if you don't supply any of the fields,
validation will fail and the inventory will not start. Currently, we do not 
explicitly differentiate between the different scopes of auth tokens.
All resource requests are made with project scope.

The project_id field is redundant, since project name, domain and region combined
with credential name uniquely identify a service client instantiation.
It will be removed in a future PR.
