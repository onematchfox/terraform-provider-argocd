---
page_title: "Terraform Provider for ArgoCD Version 7 Upgrade Guide"
subcategory: ""
description: |-
  Terraform Provider for ArgoCD Version 7 Upgrade Guide
---

# Terraform Provider for ArgoCD Version 7 Upgrade Guide

Version 7.0.0 of the Terraform Provider for ArgoCD if a major release and
includes changes that you need to consider when upgrading.

~> **Note** This guide aims to help with that process. It focusses solely on the
changes required from version `6.x` to version `7.0.0`. Before upgrading to
version 6.0.0, upgrade to the most recent 6.X version of the provider and ensure
that your environment successfully runs terraform plan. You should not see
changes you don't expect or deprecation notices.

## Table of Contents
- [Terraform Provider for ArgoCD Version 7 Upgrade Guide](#terraform-provider-for-argocd-version-7-upgrade-guide)
  - [Table of Contents](#table-of-contents)
  - [resource/argocd\_account\_token](#resourceargocd_account_token)
  - [resource/argocd\_project\_token](#resourceargocd_project_token)

## resource/argocd_account_token

Replace `renew_before` attribute with `renew_after`.

- Both `renew_before` and `renew_after` achieve the same basic functionality -
  auto-renewing a token after a given period. However, `renew_before` requires
  that the `expires_in` is set on the token to calculate the remaining lifetime
  of a token. This increases the risk of tokens expiring due to Terraform not
  being run in the window determined by `expires_at - renew_before`. In
  comparison, `renew_after` does not require tokens to expire, although
  `expires_in` can be set if this behavior is desired.

## resource/argocd_project_token

Replace `renew_before` attribute with `renew_after`.

- Both `renew_before` and `renew_after` achieve the same basic functionality -
  auto-renewing a token after a given period. However, `renew_before` requires
  that the `expires_in` is set on the token to calculate the remaining lifetime
  of a token. This increases the risk of tokens expiring due to Terraform not
  being run in the window determined by `expires_at - renew_before`. In
  comparison, `renew_after` does not require tokens to expire, although
  `expires_in` can be set if this behavior is desired.
