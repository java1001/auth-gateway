# Auth Gateway

Multi-site authentication gateway in Go using Gin, GORM, PostgreSQL, Resend, JWT, Google OAuth, and Twitter/X OAuth 2.0 with PKCE.

## What It Does

- Handles email/password signup, login, verification, and logout for multiple independent sites.
- Handles Google and Twitter/X social login using site-scoped OAuth URLs.
- Routes every authenticated request to the correct per-site database using the `site` claim inside the JWT.
- Stores each site's users in a separate PostgreSQL database.

## Environment

Copy `.env.example` to `.env` and fill in the values.

Important rules:

- `ALLOW_SITES` must contain every site host that appears in `SITE_*_HOST`.
- The gateway exits at startup if the allowlist and site registry diverge.
- `JWT_SECRET` and `OAUTH_STATE_SECRET` must be different values.
- Changing `ALLOW_SITES` requires a restart.

## Run

```bash
go run ./cmd/server
```

## Initialize Databases

Run the one-shot initializer to create the per-site tables before starting the server:

```bash
go run ./cmd/initdb
```

To initialize just one site, pass its host:

```bash
go run ./cmd/initdb --db=site3.com
```

This command:

- loads `.env`
- validates `ALLOW_SITES` against `SITE_*` entries
- connects to every configured site database, or just the one passed via `--db`
- ensures the `pgcrypto` extension exists
- runs `AutoMigrate` for `User` and `OAuthAccount`
- exits after the tables are ready

## API

Public:

- `POST /auth/signup`
- `POST /auth/login`
- `GET /auth/verify-email?token=...`
- `GET /auth/google/:site/login`
- `GET /auth/google/:site/callback`
- `GET /auth/twitter/:site/login`
- `GET /auth/twitter/:site/callback`

Protected:

- `GET /auth/me`
- `POST /auth/logout`

Health:

- `GET /health`

## Routing Model

1. Signup and login use the incoming `Host` header to select the site database.
2. JWTs carry a `site` claim and no per-site secret is needed.
3. Google OAuth uses a signed state JWT with only a CSRF nonce.
4. Twitter OAuth uses a nonce-backed state store plus PKCE.
5. OAuth callbacks validate the `:site` path segment against `ALLOW_SITES` before continuing.

## Example Reverse Proxy

```nginx
server {
    listen 443 ssl;
    server_name api.site1.com;

    location /auth/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

## curl Examples

```bash
curl -X POST https://api.site1.com/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'

curl -X POST https://api.site1.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'

curl https://api.site1.com/auth/me \
  -H "Authorization: Bearer <token>"

open https://api.site1.com/auth/google/site1.com/login
open https://api.site1.com/auth/twitter/site1.com/login
```

## Google OAuth

1. Create one OAuth 2.0 Web application client.
2. Add this redirect URI:
   - `https://auth.yourdomain.com/auth/google/site1.com/callback`
3. Add additional site-specific callback URLs for each new site you onboard.
4. Copy the client ID and secret into `.env`.

## Twitter OAuth

1. Create one OAuth 2.0 Web app in the Twitter/X developer portal.
2. Add this callback URI:
   - `https://auth.yourdomain.com/auth/twitter/site1.com/callback`
3. Add additional site-specific callback URLs for each new site you onboard.
4. Copy the client ID and secret into `.env`.

## Add a New Site

1. Add the site host to `ALLOW_SITES`.
2. Add matching `SITE_<LABEL>_HOST` and `SITE_<LABEL>_DSN` values.
3. Register the new site-specific OAuth callback URLs with Google and Twitter/X.
4. Restart the gateway.

## Security Notes

- Passwords use bcrypt with cost 12.
- Google userinfo must report `email_verified=true`.
- Twitter/X emails can be synthetic placeholders if no email is available.
- OAuth access and refresh tokens are encrypted at rest with AES-256-GCM.
- OAuth state and user session JWTs use different secrets.
- Token fragments are returned after social login so the JWT does not appear in query logs.

## Client Integration

Use the gateway as the single auth backend for every site. The client only needs to know:

- the site domain it belongs to
- the gateway base URL
- where to store the returned JWT

### 1. Email / Password Flow

Call signup and login against the site domain or its proxy route.

```bash
curl -X POST https://api.site1.com/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'

curl -X POST https://api.site1.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

Expected login response:

```json
{
  "token": "<jwt>",
  "user": {
    "id": "<uuid>",
    "email": "user@example.com"
  }
}
```

Store `token` in `localStorage`, a secure cookie, or your app state depending on your frontend architecture.

### 2. Social Login Flow

Start the login from the site-specific OAuth URL:

```text
https://api.site1.com/auth/google/site1.com/login
https://api.site1.com/auth/twitter/site1.com/login
```

After the provider callback completes, the gateway redirects the browser to:

```text
https://site1.com/auth/callback#token=<jwt>&provider=google
```

On your frontend callback page, read the fragment, save the token, then route the user onward:

```javascript
const params = new URLSearchParams(window.location.hash.slice(1));
const token = params.get("token");

if (token) {
  localStorage.setItem("auth_token", token);
  window.location.href = "/dashboard";
}
```

### 3. Authenticated API Calls

Send the JWT as a Bearer token. The gateway reads the `site` claim and routes to the correct database automatically.

```javascript
const token = localStorage.getItem("auth_token");

const response = await fetch("https://api.site1.com/auth/me", {
  headers: {
    Authorization: `Bearer ${token}`,
  },
});

const profile = await response.json();
```

### 4. Verification Links

Email verification links point back to the site domain so the correct tenant database is resolved from the `Host` header.

```text
https://site1.com/auth/verify-email?token=<verification-token>
```

### 5. Frontend Checklist

- Use the site domain for signup and login requests.
- Use the site-specific OAuth login URL for social auth.
- Handle the callback fragment on `/auth/callback`.
- Persist the JWT securely and send it in the `Authorization` header.
- Never send `site` manually from the client; it already lives inside the token.
