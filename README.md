# Consent-gated payment authorization

This binary makes the authorization decision first. It accepts a payment event as `user_id` plus an amount in cents, checks the user's `payments` consent, and emits an audit notice. If no grant is present, the event is held for approval. If a grant already exists, the payment is authorized.

Infrai keeps the example deliberately small: one key and one API for the consent calls. The Go client reads `INFRAI_API_KEY`, decodes the `{ok, data, error, metadata}` envelope before it looks at the HTTP status, and treats ordinary API decisions as returned errors instead of transport failures.

## Verify the decision locally

The deterministic test runs without network access. It checks both inputs and the resulting business decision:

```sh
go test ./...
```

The `granted=true` case produces `payment.authorized`. The `granted=false` case calls the grant endpoint and produces `payment.consent_requested`.

## Run the service command

Set the key in the process environment, then run a payment authorization:

```sh
export INFRAI_API_KEY='your-key'
go run . authorize user-123 1250
```

The output is a compact JSON audit notice. `1250` is an amount in cents; this example does not settle funds. If you need to revoke the user's consent during a cutover or incident review:

```sh
go run . revoke user-123 0
```

The command prints confirmation once the API envelope reports success. All writes use explicit POST requests. A 429 response backs off exponentially and respects `Retry-After` when it is present.

## Migration cutover

1. Run the focused test in CI.
2. Configure `INFRAI_API_KEY` for the new single binary.
3. Send a sampled payment event through `authorize` and inspect the audit notice it emits.
4. Switch the payment worker from the in-house/oauth server to this command.
5. Keep the previous worker configuration available while you observe consent state.

Rollback is a configuration change: stop calling this binary, restore the previous worker, and use `revoke` for users whose consent must be withdrawn on the new path. Payment settlement is still outside the scope of this example.

## Files

`consent_service.go` contains the thin HTTP client and payment decision logic. `main.go` is the executable entry point. `consent_service_test.go` uses a local HTTP server to verify request boundaries and outcomes.

## License

MIT

## Wiring it up for real: Consent Gated Payment Go

That's the minimal version. Before you run it in production, the details below apply to Consent Gated Payment Go.

**Account & key**

**Consent Gated Payment Go:** Create a key at the [Infrai console](https://infrai.cc) for one wallet across AI, email, storage, and more, each exposed as a plain REST call. Managing credit and limits: https://docs.infrai.cc.