# Embedded identity provider

This package embeds [Dex](https://dexidp.io) as NetBird's OpenID Connect
provider. It serves the sign-in pages under `/oauth2`, stores local users, and
federates sign-in to external providers registered through the management
API (`/api/identity-providers`).

## Entra ID without a client secret

An identity provider of type `entra` can authenticate to Entra with a federated
identity credential instead of a client secret. NetBird signs a short-lived JWT
with its own OIDC signing key and sends it as the `client_assertion` of the
token request ([RFC 7523, section 2.2](https://datatracker.ietf.org/doc/html/rfc7523#section-2.2)).
Entra checks it against the JWKS that NetBird already publishes. Nothing
expires on the app registration, and no secret is stored in NetBird.

Requirements:

- The NetBird domain must be reachable from the internet over HTTPS, because
  Entra fetches `https://<netbird domain>/oauth2/.well-known/openid-configuration`
  and `https://<netbird domain>/oauth2/keys`.
- The Dex signing key must use RS256, which is the default.

### 1. Register the provider in NetBird

Create the provider first. The connector ID it returns becomes part of the
credential's subject.

```http
POST /api/identity-providers
Content-Type: application/json

{
  "name": "Entra ID",
  "type": "entra",
  "issuer": "https://login.microsoftonline.com/<tenant id>/v2.0",
  "client_id": "<application (client) id>",
  "client_secret": "",
  "client_auth": "federated"
}
```

The response carries the `id` (for example `entra-d3v8o4q6lnn4g9hqv7l0`) and
`"client_auth": "federated"`. The `client_secret` key must be present because
the schema still requires it, but it must be empty when `client_auth` is
`federated`. Only the `entra` type accepts `federated`. Leaving out
`client_auth`, or sending `secret`, keeps the client secret flow.

`PUT /api/identity-providers/<id>` with `client_auth` switches an existing
provider between the two modes. Switching to `secret` needs a non-empty
`client_secret`.

### 2. Add the federated credential in Entra

On the app registration, open **Certificates & secrets**, then
**Federated credentials**, and add one with the scenario **Other issuer**:

| Field    | Value                                        |
| -------- | -------------------------------------------- |
| Issuer   | `https://<netbird domain>/oauth2`            |
| Subject  | `netbird-identity-provider:<id from step 1>` |
| Audience | `api://AzureADTokenExchange`                 |

The issuer must match exactly, with no trailing slash. With Terraform, use
`azuread_application_federated_identity_credential` in place of
`azuread_application_password`:

```hcl
resource "azuread_application_federated_identity_credential" "netbird" {
  application_id = azuread_application.netbird.id
  display_name   = "netbird-embedded-idp"
  issuer         = "https://${var.netbird_domain}/oauth2"
  subject        = "netbird-identity-provider:${var.connector_id}"
  audiences      = ["api://AzureADTokenExchange"]
}
```

The redirect URI on the app registration stays
`https://<netbird domain>/oauth2/callback`.

### How it works

The `entra` provider with federated client authentication is stored as a Dex
connector of type `oidc-assertion` (package `oidcassertion`). It is Dex's
`oidc` connector with only client authentication changed. The code exchange
and the refresh grant send `client_id`, `client_assertion_type` and
`client_assertion`, and never `client_secret`. Each assertion has:

| Claim or header | Value                                             |
| --------------- | ------------------------------------------------- |
| `iss`           | the Dex issuer, `https://<netbird domain>/oauth2` |
| `sub`           | `netbird-identity-provider:<connector id>`        |
| `aud`           | `api://AzureADTokenExchange`                      |
| `iat`, `nbf`    | now                                               |
| `exp`           | five minutes after `iat`                          |
| `jti`           | a random UUID                                     |
| `kid` header    | the ID of Dex's current signing key               |

Dex rotates its signing key on a schedule and keeps publishing the previous
key for verification, so a rotation needs no change in Entra.

## Sign-in page: the "another login method" link

The password page links back to the login method chooser only when the chooser
would offer more than one method for the client that started the sign-in.
Dex decides on that link from its in-memory connector cache, which can hold
connectors the chooser filters out or that were deleted from storage. The
provider's HTTP handler therefore recounts the connectors from storage, with
the chooser's own grant type and client filters, and drops the link when there
is only one. The templates render the link only when `.BackLink` is set, so a
custom theme served from a directory gets the same behaviour without changes.
