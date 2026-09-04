# Auth Gateway Svelte Client

This client uses the gateway's current API contract:

- `POST /auth/signup` sends an email verification code.
- `POST /auth/verify-email` verifies `{ email, code }`.
- `POST /auth/login` returns `access_token` and `refresh_token`.
- Expired access tokens are rotated through `POST /auth/refresh`.
- Google and Twitter return tokens in the `/callback` URL fragment.
- Password recovery uses `POST /auth/forgot-password` followed by `POST /auth/reset-password`.

## Run locally

```bash
npm install
cp .env.example .env
npm run dev
```

`API_URL` must point to the gateway auth base URL, normally
`http://localhost:8080/auth`. `SITE_HOST` is optional and is useful when the
browser hostname is not the tenant hostname. The backend must have that host
configured in both `ALLOW_SITES` and `SITE_*_HOST`; otherwise SiteResolver
returns `SITE_NOT_FOUND`.

For production, use the site-specific API hostname and configure the reverse
proxy to forward the tenant host expected by the gateway. The client never
sends a `site` header; the gateway derives it for login and reads it from the
JWT for protected requests.

## OAuth

The social buttons open:

```text
/auth/google/{site}/login
/auth/twitter/{site}/login
```

The callback page reads `access_token` and `refresh_token` from the URL
fragment, stores them, clears the fragment from browser history, and loads the
user profile from `/auth/me`.
