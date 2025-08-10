# Kerbernetes (k10s)

## Description

Kerbernetes is a Kubernetes-based project that integrates LDAP and Kerberos authentication for managing access control. It provides seamless authentication and role-based access management for Kubernetes clusters.

## Features

- Kerberos-based authentication endpoint.
- LDAP integration for user and group management.
- Automatic reconciliation of Kubernetes RoleBindings and ClusterRoleBindings.

## Setup

### Prerequisites

- **k8s** cluster
- **KDC** server
- **LDAP** server (optional, for LDAP group bindings)
- **Helm** for deploying the Kerbernetes chart

## Contributing

Contributions are welcome! To contribute:

1. Fork the repository.
2. Create a new branch:

   ```bash
   git checkout -b feature-name
   ```

3. Commit your changes following the conventionnal commit message format:

   ```bash
   git commit -m "feat: add new feature"
   ```

4. Push to your branch:

   ```bash
   git push origin feature-name
   ```

5. Open a pull request.
