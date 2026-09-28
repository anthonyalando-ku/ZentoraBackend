# Delivery information rollout

Delivery is informational and excluded from order/payment/conversion totals. Standard Nairobi guidance starts at KES 300; actual delivery charges are confirmed separately. Orders remain available nationwide.

## Existing shipping tables

Migration `internal/db/migrations/009_delivery_information.sql` extends existing tables; it creates no policy table and does not change `001_svc_init.sql`.

- `shipping_methods`: optional stable code, pricing mode, version and update metadata. The new `standard-delivery` method is informational; existing rows remain fixed.
- `shipping_zones`: optional stable code and region code. New Kenya-wide and Nairobi zones map to KE through `shipping_zone_countries`; Nairobi uses KE-30.
- `shipping_method_rates`: rate type, currency, and nullable base_fee for quotes. Nairobi has an indicative KES 300 rate; nationwide delivery requires a quote with no amount.
- `orders.delivery_information`: nullable JSONB snapshot. Old orders remain null; monetary columns are unchanged.

`base_fee` and API fee fields use **major KES units** (300 means KES 300), not cents. Admin accepts whole KES amounts from 1 to 100000. No automatic charging switch is exposed. Structured region data allows future matching; today's message describes Nairobi guidance generally and does not pretend to quote a free-text address.

## API and UI

- `GET /api/v1/delivery-policy`: public information generated from shipping records.
- `GET /api/v1/admin/delivery-policy`: same policy, protected by AdminOnly.
- `PUT /api/v1/admin/delivery-policy`: `{ "expected_version": 1, "nairobi_indicative_fee": 300, "additional_information": "" }`. Unknown fields are rejected; stale versions return 409. Fee, note and version are saved atomically.
- Admin screen: `/admin/delivery`. Coverage is fixed to Kenya nationwide; the preview explains exclusion from totals.
- Product, cart, checkout and help pages share the backend notice, with one-minute query freshness and a generic nonnumeric fallback.
- New guest/registered orders save policy version, method ID, pending-confirmation status, null confirmed fee, currency, and notice. Policy reading happens before the order transaction with a two-second timeout; failure stores a generic notice. The snapshot is inserted atomically with the order.
- Order responses preserve existing capitalized fields and add `DeliveryInformation`. Success, history, admin views and emails preserve the snapshot; policy edits do not rewrite old orders.
- Checkout success retains the backend response instead of calculating a summary from a cleared cart. Conversions continue using `TotalAmount` without the indicative fee.

## Deployment order

1. Back up the database and inspect `SHOW search_path` and existing shipping/order definitions. Use the same database role/search_path as the API. Repository SQL is unqualified; do not assume `merged_schema.sql` is deployed.
2. Apply **only migration 009**, once, with `psql -v ON_ERROR_STOP=1 -f internal/db/migrations/009_delivery_information.sql` using your existing secure connection setup. The file owns its transaction and has a five-second lock timeout. Failure rolls back the migration. Do not replay initial schemas or every SQL file.
3. Deploy backend, then frontend. The new backend requires migration 009. Old UI clients remain compatible; the new UI falls back to a generic notice if the public policy endpoint is unavailable.
4. Verify policy GET returns 300 KES informational guidance and non-admin users cannot update settings. Review legacy orders read-only. Use staging for checkout writes.
5. Roll back application releases if needed, keeping the additive schema and saved history.

No migration has been applied to production by this implementation.

## Merchant feed

The generated feed previously advertised a hardcoded KES 200 nationwide fee. Informational delivery now omits that per-offer override rather than publishing an invented fixed price. Review Merchant Center account-level shipping settings before rollout and refresh previously generated feed output. The application does not modify the external account.

Google documents account-level configuration and product-level overrides at https://support.google.com/merchants/answer/6324484?hl=en . Do not use the indicative Nairobi fee as a guaranteed nationwide charge.

## Verification

Backend checks:

```powershell
go test ./internal/domain/delivery ./internal/handlers/delivery ./internal/service/order ./internal/service/email ./internal/merchant/application/hydration ./internal/app
```

Persistence and real checkout service tests use a **disposable localhost PostgreSQL** database. They create/remove a unique test schema and never load `.env` or production settings:

```powershell
$env:DELIVERY_TEST_DATABASE_URL = 'postgres://TEST_USER@127.0.0.1:TEST_PORT/TEST_DATABASE?sslmode=disable'
go test ./internal/repository/postgres -run TestDeliveryMigrationAndOrderPersistence -count=1
```

Tests apply the migration over the existing shipping schema, preserve old totals, round-trip snapshots through read/list/status paths, reject stale edits, and exercise Nairobi/Mombasa guest and Kisumu registered checkout including unavailable-policy fallback.

UI checks (from ZentoraUI): `npm run build` and `node --test scripts/delivery.test.mjs`. Existing UI lint debt is separate; new delivery modules should pass ESLint.

## Future charging

Use explicit accepted delivery quotes or fixed rates, with separate delivery/payment statuses. Never reinterpret an old indicative snapshot as an actual charge.
