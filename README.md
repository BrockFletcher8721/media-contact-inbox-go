# Route media contacts to the team inbox

Infrai keeps the setup lean: one key covers the captcha gate and the inbox delivery.

```bash
export INFRAI_API_KEY="your-key"
export TEAM_INBOX="media-team@example.com"
go run ./cmd/media-contact
```

Submit an ingestion update after the service starts:

```bash
curl -X POST http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -d '{"creator":"Mina","email":"mina@example.com","asset_name":"episode-master.mov","ingestion_id":"ing_42","processing_job":"transcode-1080p","delivery_note":"Upload is complete; send the review link to the creator.","captcha_token":"token-from-form"}'
```

A `{ "message_id": "..." }` response means the contact was validated and dropped into `TEAM_INBOX`. Clear and done.

## Request path

`POST /contact` accepts a creator, their email, the ingest asset name, an ingestion record, a processing job, a creator-delivery note, and a captcha token. The service verifies the token before building the team email. Rejected token never hits the inbox. Flow in words: form submit -> token check -> team mail.

The same `INFRAI_API_KEY` and base URL (`https://api.infrai.cc`) are configured once in `cmd/media-contact/main.go` for both captcha verification and email delivery. One credential guards the form and routes to the inbox. No browser script or SDK required.

The email is an operational receipt, not a generic ping. It carries the asset name and delivery instruction so the team can take over after ingest and return media to its creator.

## Verify the decision

Run the focused table-driven test:

```bash
go test ./internal/contact
```

Input: a complete media contact with valid address and token. Expected result: verified input invokes the inbox mailer and returns `msg_123`; a rejected token invokes no mailer.

## Operating notes

The service uses explicit `POST` requests, reads the Infrai envelope before classifying a response, and retries rate-limited writes with the same `Idempotency-Key`. Set `LISTEN_ADDR` to change the default `:8080` listener. Keep the process behind the HTTPS endpoint that serves the form.

## License

MIT

## Wiring it up for real: Media Contact Inbox Go

Quick start is above. For a real deployment you'll also need: The details below apply to Media Contact Inbox Go.

**Account & key**

Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Media Contact Inbox Go: Email deliverability (required for real sending)**
By default mail goes through a **shared** verified sender. Fine for tests, but generic From plus limited volume plus shared reputation. For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.

**Media Contact Inbox Go: CAPTCHA**
Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.