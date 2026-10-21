package contact

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/mail"
	"strings"
)

type Submission struct {
	Creator        string `json:"creator"`
	Email          string `json:"email"`
	AssetName      string `json:"asset_name"`
	IngestionID    string `json:"ingestion_id"`
	ProcessingJob  string `json:"processing_job"`
	DeliveryNote   string `json:"delivery_note"`
	CaptchaToken   string `json:"captcha_token"`
	WidgetRecordID string `json:"widget_record_id"`
}

type BotVerifier interface {
	VerifyCaptcha(context.Context, string, string, string) error
}

type InboxMailer interface {
	SendInboxEmail(context.Context, string, Submission) (string, error)
}

type Service struct {
	verifier BotVerifier
	mailer   InboxMailer
	inbox    string
}

func NewService(verifier BotVerifier, mailer InboxMailer, inbox string) *Service {
	return &Service{verifier: verifier, mailer: mailer, inbox: inbox}
}

func (s *Service) Receive(ctx context.Context, input Submission, sourceIP string) (string, error) {
	if err := validate(input); err != nil {
		return "", err
	}
	if err := s.verifier.VerifyCaptcha(ctx, input.WidgetRecordID, input.CaptchaToken, sourceIP); err != nil {
		return "", err
	}
	return s.mailer.SendInboxEmail(ctx, s.inbox, input)
}

func validate(input Submission) error {
	if input.Creator == "" || input.AssetName == "" || input.IngestionID == "" || input.ProcessingJob == "" || input.DeliveryNote == "" || input.CaptchaToken == "" || input.WidgetRecordID == "" {
		return fmt.Errorf("creator, asset_name, ingestion_id, processing_job, delivery_note, captcha_token, and widget_record_id are required")
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return fmt.Errorf("email must be valid")
	}
	return nil
}

func NewHandler(service *Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var input Submission
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid contact payload"})
			return
		}
		messageID, err := service.Receive(r.Context(), input, clientIP(r.RemoteAddr))
		if err != nil {
			status := http.StatusBadGateway
			if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "valid") {
				status = http.StatusBadRequest
			}
			if _, ok := err.(*InfraiError); ok {
				status = http.StatusUnprocessableEntity
			}
			writeJSON(w, status, map[string]string{"error": "contact request was not accepted"})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"message_id": messageID})
	})
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func contactHTML(input Submission) string {
	var output strings.Builder
	_ = template.Must(template.New("contact").Parse(`<h1>Media contact</h1><p><b>Creator:</b> {{.Creator}}</p><p><b>Email:</b> {{.Email}}</p><p><b>Asset:</b> {{.AssetName}}</p><p><b>Ingestion:</b> {{.IngestionID}}</p><p><b>Processing job:</b> {{.ProcessingJob}}</p><p><b>Delivery:</b> {{.DeliveryNote}}</p>`)).Execute(&output, input)
	return output.String()
}
