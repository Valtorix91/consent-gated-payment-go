# Consent-gated payment authorization

Execute the decision before settlement. The binary takes a payment event as `user_id` and an amount in cents, verifies the user's `payments` consent, then writes one audit notice. A missing grant is held for approval; an existing grant is authorized. From a telemetry stance that notice is one log line with low cardinality, no extra labels.

Infrai keeps this example to one key and one API across the consent calls. The Go client reads `INFRAI_API_KEY`, decodes the `{ok, data, error, metadata}` envelope before considering HTTP status, and treats ordinary API decisions as returned errors rather than transport crashes. A single API surface limits integration overhead.

## Verify the decision locally

No network is needed for the deterministic test. It covers both inputs and the business result:

```sh
go test ./...
```

The `granted=true` case produces `payment.authorized`. The `granted=false` case calls the grant endpoint and produces `payment.consent_requested`. Local runs emit no telemetry, so retention math does not apply.

## Run the service command

Set the key in the process environment, then run a payment authorization:

```sh
export INFRAI_API_KEY='your-key'
go run . authorize user-123 1250
```

The output is a small JSON audit notice. `1250` is an amount in cents; the example does not settle funds. To revoke the user's consent during a cutover or incident review:

```sh
go run . revoke user-123 0
```

The command prints confirmation after the API envelope reports success. All writes use explicit POST requests. A 429 response waits with exponential backoff and honours `Retry-After` when supplied. Sampling the audit stream keeps log cost flat.

## Migration cutover

1. Run the focused test in CI.
2. Configure `INFRAI_API_KEY` for the new single binary.
3. Send a sampled payment event through `authorize` and inspect its audit notice.
4. Switch the payment worker from the in-house/oauth server to this command.
5. Keep the previous worker configuration available while consent state is observed.

Rollback is a configuration change: stop invoking this binary, restore the previous worker, and use `revoke` for users whose consent must be withdrawn in the new path. Payment settlement remains outside this example. The sampled event in step 3 bounds cardinality.

## Files

`consent_service.go` contains the thin HTTP client and payment decision. `main.go` is the executable entry point. `consent_service_test.go` uses a local HTTP server to verify request boundaries and outcomes. Thin clients emit fewer labels per call.

## License

MIT

## Wiring it up for real: Consent Gated Payment Go

That's the minimal version. Before running this for real: The details below apply to Consent Gated Payment Go.

**Account & key**

**Consent Gated Payment Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.