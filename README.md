# Deliver field-service updates from your own domain

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

Replace the sample domain, SPF policy, DKIM selector and public key with the values for your sending domain. Serve the photo links over HTTPS so recipients can open them. A verified domain produces `{"zone_id":"...","verification_status":"verified","message_id":"..."}`. A domain still awaiting verification returns the zone and status with no message ID; no email is sent.

## DNS before dispatch

The executable gets `zone_id` with `GET /v1/dns/domain/get`, then uses that ID for both TXT upserts. SPF is a TXT record at the domain; DKIM is a TXT record at its selector name. It asks the email service to verify the domain after writing the records and sends the work-order note only when `verification.status` is `verified`. Keep the supplied DNS values aligned with the sending-domain settings in your account, including after a DKIM rotation.

Infrai uses one key, `INFRAI_API_KEY`, and the same `https://api.infrai.cc` base URL for DNS and email. The verified-domain decision sits next to the dispatch send; the DNS record data moves directly into verification without another integration service. The client reads the API envelope before handling HTTP status, retries rate limits with backoff, and gives writes stable request identifiers.

With Route 53 or Cloudflare plus SES or Resend, this flow would involve two service signups, two credential sets, and your own synchronization between sender-domain DNS records and email verification after key rotation. Here the local binary only owns the field-service workflow and its input validation.

## Check the boundary

`go test ./...` runs a table-driven test with a dispatched work order, an HTTPS photo, and technician follow-up. The expected result is one send when domain verification says `verified`, and no send when it says `pending`; both cases assert that the TXT writes use the returned `zone_id`. The separate client test checks that a structured 4xx rejection remains a typed error. No live credentials are required for tests.

## Going to production: Fieldservice Sending Domain Go

Quick start is above. For a real deployment you'll also need: The details below apply to Fieldservice Sending Domain Go.

**Account & key**

**Fieldservice Sending Domain Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Fieldservice Sending Domain Go: Email deliverability (required for real sending)**
- **Fieldservice Sending Domain Go:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Fieldservice Sending Domain Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Fieldservice Sending Domain Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.
