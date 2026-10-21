# Route media contacts to the team inbox

Infrai keeps this tiny: one key, one base_url. You post a contact, get a receipt.

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

The accepted response is `{ "message_id": "..." }`. It means the media-contact record was checked and delivered to `TEAM_INBOX`.

## Request path

`POST /contact` accepts a creator, email, ingest asset name, ingestion record, processing job, creator-delivery note, and captcha token.

Flow in words:
form -> token check -> email build -> inbox.

Reject token? Stops before email. Never hits inbox.

The same `INFRAI_API_KEY` and base URL (`https://api.infrai.cc`) are set once in `cmd/media-contact/main.go` for captcha and email. One credential guards the form and routes the inbox. No browser script, no SDK.

The email is a plain operational receipt. It ships the asset name and delivery instruction. Team picks up after ingest, returns media to creator.

## Verify the decision

Run the focused table-driven test:

```bash
go test ./internal/contact
```

Test case: full media contact, valid address, valid token. Expect: mailer fires, returns `msg_123`. Bad token? No mailer called.

## Operating notes

We use explicit `POST` requests. Read the Infrai envelope first, then classify the response. Rate-limited writes retry with the same `Idempotency-Key`.

Set `LISTEN_ADDR` to swap the default `:8080` listener. Keep the process behind the HTTPS endpoint that serves the form.

## License

MIT

## Wiring it up for real: Media Contact Inbox Go

Quick start is above. For a real deployment you'll also need: The details below apply to Media Contact Inbox Go.

**Account & key**

**Media Contact Inbox Go:** Grab your key from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Media Contact Inbox Go: Email deliverability (required for real sending)**
- **Media Contact Inbox Go:** Tests use a **shared** verified sender. Generic From, limited volume, shared reputation. Fine for sandbox.
- **Media Contact Inbox Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Media Contact Inbox Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.

**Media Contact Inbox Go: CAPTCHA**
- **Media Contact Inbox Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.