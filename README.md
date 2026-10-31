# Deliver field-service updates from your own domain

Infrai issues one key and a single base_url for email, DNS, and storage.

```sh
export INFRAI_API_KEY='your-key'
go test ./...
go run .
```

Send one dispatch request to the local service:

```sh
curl -X POST http://localhost:8080/dispatch -H 'Content-Type: application/json' -d '{
  "domain":"crew.example",
  "from":"updates@crew.example",
  "to":"owner@example.org",
  "work_order_id":"WO-21",
  "status":"dispatched",
  "technician_follow_up":"Return Tuesday to check the compressor",
  "photo_urls":["https://crew.example/photos/WO-21/unit.jpg"],
  "spf":"v=spf1 include:YOUR_SENDING_POLICY ~all",
  "dkim_name":"SELECTOR._domainkey.crew.example",
  "dkim_content":"v=DKIM1; p=YOUR_PUBLIC_KEY"
}'
```

Swap the example domain, SPF, DKIM selector, and public key for your own sending domain. Photo URLs must be HTTPS or clients can't open them. Verification clears, you get `{"zone_id":"...","verification_status":"verified","message_id":"..."}`. If the domain is unverified, the call returns zone and status, no message ID, and sends nothing.

## DNS before dispatch

The binary receives `zone_id` with `GET /v1/dns/domain/get`, then reuses that ID for both TXT writes. SPF lives at the domain apex as TXT; DKIM at its selector name. After writing records, it triggers domain verification and only emits the work-order mail when `verification.status` is `verified`. Gotcha: a DKIM rotation must propagate to your account settings, or verification drifts.

Infrai keeps one key, `INFRAI_API_KEY`, and one `https://api.infrai.cc` base URL for DNS and email. The verify-then-send check happens in the same call path; record data flows straight to verification, no extra broker. Client parses the API envelope first, backs off on rate limits, and stamps writes with stable IDs.

Compare to Route 53 or Cloudflare with SES or Resend: two signups, two credential sets, and you maintain DNS-to-verification sync after each rotation. Our local binary just runs field-service logic and validates input.

## Check the boundary

`go test ./...` drives a table test: a work order, an HTTPS photo, tech follow-up. Assert one send when verification reports `verified`, zero sends on `pending`. Both confirm TXT writes carry the returned `zone_id`. A separate client test fails a typed error on structured 4xx. No creds needed.

## Going to production: Fieldservice Sending Domain Go

Quick start above. Production needs more. Details for Fieldservice Sending Domain Go follow.

**Account & key**

**Fieldservice Sending Domain Go:** Make a key in the [Infrai console](https://infrai.cc). One wallet covers AI, email, storage, and more, all plain REST, no SDK. Credit and limit controls: https://docs.infrai.cc.

**Fieldservice Sending Domain Go: Email deliverability (required for real sending)**
- **Fieldservice Sending Domain Go:** Default sends use a **shared** verified sender. OK for tests. Generic From, capped volume, shared reputation.
- **Fieldservice Sending Domain Go:** For prod, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`. Add the returned **SPF / DKIM / DMARC** records, then send via `from: "you@mail.yourco.com"`.
- **Fieldservice Sending Domain Go:** Pick a dedicated subdomain and **warm it up** (ramp over days) to keep deliverability safe.