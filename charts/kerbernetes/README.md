# Kerbernetes Helm Chart

Kerbernetes is a Kubernetes authentication service that integrates with Kerberos and LDAP for secure access control. This Helm chart allows you to deploy and manage the Kerbernetes service in your Kubernetes cluster.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installation

To install the chart with the release name `kerbernetes`:

```bash
helm repo add froz42 oci://ghcr.io/froz42/kerbernetes
helm install kerbernetes froz42/kerbernetes
```

You can find the setup instructions [here](https://github.com/froz42/kerbernetes/wiki/Setup.md)

## Values

The following table lists the configurable parameters of the Kerbernetes chart and their default values.

| Parameter                | Description                           | Default                       |
| ------------------------ | ------------------------------------- | ----------------------------- |
| `replicaCount`           | Number of replicas for the deployment | `1`                           |
| `serviceAccountName`     | Name of the service account           | `kerbernetes-api-sa`          |
| `image.repository`       | Image repository                      | `ghcr.io/froz42/kerbernetes`  |
| `image.tag`              | Image tag                             | `v1.1.0`                      |
| `image.pullPolicy`       | Image pull policy                     | `IfNotPresent`                |
| `httpPort`               | HTTP port for the service             | `3000`                        |
| `ldap.enabled`           | Enable LDAP integration               | `false`                       |
| `ldap.url`               | LDAP server URL                       | `ldap://ldap.example.com`     |
| `ldap.userBaseDN`        | User base DN for LDAP                 | `ou=users,dc=example,dc=com`  |
| `ldap.userFilter`        | User filter for LDAP                  | `(uid=%s)`                    |
| `ldap.groupBaseDN`       | Group base DN for LDAP                | `ou=groups,dc=example,dc=com` |
| `ldap.groupFilter`       | Group filter for LDAP                 | `(member=%s)`                 |
| `ldap.bindDN`            | Bind DN for LDAP                      | `cn=read,dc=example,dc=com`   |
| `service.type`           | Kubernetes service type               | `ClusterIP`                   |
| `service.port`           | Service port                          | `3000`                        |
| `secrets.keytabSecret`   | Name of the keytab secret             | `krb5-keytab`                 |
| `secrets.ldapSecret`     | Name of the LDAP secret               | `ldap`                        |
| `readinessProbe.enabled` | Enable readiness probe                | `true`                        |
| `readinessProbe.*`       | Readiness probe configuration         | See `values.yaml`             |
| `livenessProbe.enabled`  | Enable liveness probe                 | `true`                        |
| `livenessProbe.*`        | Liveness probe configuration          | See `values.yaml`             |
| `ingress.enabled`        | Enable ingress                        | `false`                       |
| `ingress.className`      | Ingress class name                    | `""`                          |
| `ingress.annotations`    | Ingress annotations                   | `{}`                          |
| `ingress.hosts`          | Ingress hosts configuration           | See `values.yaml`             |
| `ingress.tls`            | Ingress TLS configuration             | `[]`                          |

> **Note**: The `secrets.keytabSecret` parameter is required and must reference a Kubernetes secret containing the key `krb5.keytab`. This key stores the Kerberos keytab file, which is essential for authenticating with the KDC.

## Customization

You can customize the chart by overriding the default values in `values.yaml`. For example:

```bash
helm install kerbernetes froz42/kerbernetes --set replicaCount=3 --set ldap.enabled=true
```

## Uninstallation

To uninstall the `kerbernetes` release:

```bash
helm uninstall kerbernetes
```

This command removes all the Kubernetes components associated with the chart and deletes the release.

## License

MIT License

Copyright (c) 2025 froz

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
