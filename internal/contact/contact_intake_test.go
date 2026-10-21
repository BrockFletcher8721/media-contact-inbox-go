package contact

import (
	"context"
	"errors"
	"testing"
)

type verifierStub struct{ err error }

func (s verifierStub) VerifyCaptcha(context.Context, string, string, string) error { return s.err }

type mailerStub struct {
	called bool
}

func (s *mailerStub) SendInboxEmail(_ context.Context, inbox string, input Submission) (string, error) {
	s.called = true
	if inbox != "team@example.com" || input.AssetName != "episode-master.mov" {
		return "", errors.New("unexpected delivery request")
	}
	return "msg_123", nil
}

func TestReceiveRoutesOnlyVerifiedMediaContact(t *testing.T) {
	input := Submission{
		Creator: "Mina", Email: "mina@example.com", AssetName: "episode-master.mov",
		IngestionID: "ing_42", ProcessingJob: "transcode-1080p",
		DeliveryNote: "Upload is complete; send the review link to the creator.", CaptchaToken: "token-1",
		WidgetRecordID: "widget-1",
	}
	cases := []struct {
		name       string
		verifyErr  error
		wantID     string
		wantMailed bool
	}{
		{name: "verified contact reaches inbox", wantID: "msg_123", wantMailed: true},
		{name: "rejected contact does not send email", verifyErr: &InfraiError{Code: "rejected"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mailer := &mailerStub{}
			service := NewService(verifierStub{err: tc.verifyErr}, mailer, "team@example.com")
			gotID, err := service.Receive(context.Background(), input, "192.0.2.10")
			if (err == nil) != (tc.verifyErr == nil) || gotID != tc.wantID || mailer.called != tc.wantMailed {
				t.Fatalf("Receive() id=%q err=%v mailed=%v", gotID, err, mailer.called)
			}
		})
	}
}
