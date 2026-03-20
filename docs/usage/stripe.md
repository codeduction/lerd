# Stripe / Laravel Cashier

## stripe-mock

A local Stripe API mock for feature tests that exercise Cashier without hitting the real API and without needing a Stripe account.

```yaml
# ~/.config/lerd/services/stripe-mock.yaml
name: stripe-mock
image: docker.io/stripemock/stripe-mock:latest
description: "Local Stripe API mock for Cashier testing"
ports:
  - 12111:12111
```

```bash
lerd service add ~/.config/lerd/services/stripe-mock.yaml
lerd service start stripe-mock
```

Point the Stripe PHP SDK at the mock in your `AppServiceProvider` or test bootstrap:

```php
\Stripe\Stripe::$apiBase = 'http://lerd-stripe-mock:12111';
```

---

## stripe:listen

Forwards live or test webhook events from Stripe to your local app via the Stripe CLI. Requires a real Stripe API key and an active internet connection.

```bash
lerd stripe:listen                         # forwards to https://myapp.test/stripe/webhook
lerd stripe:listen --path /webhooks/stripe # custom webhook path
```

Lerd resolves the current site from the working directory and constructs the target URL automatically. The Stripe CLI must be installed and authenticated (`stripe login`).
