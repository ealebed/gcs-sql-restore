# ADR-004: Cloud SQL private IP only

## Status
Accepted

## Date
2026-08-07

## Context
Creating a Cloud SQL instance with a public IP failed under organization policy
`constraints/sql.restrictPublicIp` in project `ylebi-rnd`. The PoC restore path
uses Cloud SQL Import from GCS (Admin API), which does not require client
connectivity to the instance IP.

## Decision
Provision Cloud SQL with **private IP only**:

- Dedicated VPC + subnet
- Private Services Access (`servicenetworking.googleapis.com`) peering range
- `ipv4_enabled = false`
- `enable_private_path_for_google_cloud_services = true`

Verify data via **Cloud SQL Studio** in the Console (works without public IP).

## Alternatives Considered

### Request org-policy exception for public IP
- Pros: Simpler networking for ad-hoc `mysql` clients
- Cons: Fights security baseline; unnecessary for Import + Studio
- Rejected

### Use existing `default` VPC
- Pros: Less Terraform
- Cons: Couples PoC to shared network state; harder to tear down cleanly
- Rejected for PoC isolation; can be revisited later

## Consequences
- `terraform apply` creates VPC/PSA resources (slightly longer first apply)
- Direct `mysql` from a laptop needs VPN/bastion/Auth Proxy in the VPC
- GCS → Import → archive flow is unchanged
