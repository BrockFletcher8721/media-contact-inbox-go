package contact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type InfraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type apiError struct {
	Code string `json:"code"`
	Hint string `json:"hint"`
}

type InfraiError struct {
	Code   string
	Hint   string
	Status int
}

func (e *InfraiError) Error() string {
	return fmt.Sprintf("Infrai request rejected: %s", e.Code)
}

func NewInfraiClient(baseURL, key string, httpClient *http.Client) *InfraiClient {
	return &InfraiClient{baseURL: strings.TrimRight(baseURL, "/"), key: key, http: httpClient}
}

func (c *InfraiClient) VerifyCaptcha(ctx context.Context, widgetRecordID, token, ip string) error {
	body := struct {
		WidgetRecordID string  `json:"widget_record_id"`
		Token          string  `json:"token"`
		IP             string  `json:"ip"`
		Action         string  `json:"action"`
		ScoreThreshold float64 `json:"score_threshold"`
	}{WidgetRecordID: widgetRecordID, Token: token, IP: ip, Action: "media_contact", ScoreThreshold: 0.5}
	return c.call(ctx, http.MethodPost, "/v1/captcha/verify", body, "", nil)
}

func (c *InfraiClient) SendInboxEmail(ctx context.Context, inbox string, input Submission) (string, error) {
	body := struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		HTML    string `json:"html"`
	}{To: inbox, Subject: "Media contact: " + input.AssetName, HTML: contactHTML(input)}
	var result struct {
		MessageID string `json:"message_id"`
	}
	requestID := strings.Join([]string{input.Email, input.AssetName, input.CaptchaToken}, ":")
	if err := c.call(ctx, http.MethodPost, "/v1/email/send", body, requestID, &result); err != nil {
		return "", err
	}
	return result.MessageID, nil
}

func (c *InfraiClient) call(ctx context.Context, method, path string, payload any, idempotencyKey string, output any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(encoded))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		var reply envelope
		if err := json.Unmarshal(body, &reply); err != nil {
			return err
		}
		if !reply.OK {
			if resp.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				time.Sleep(retryAfter(resp.Header.Get("Retry-After"), attempt))
				continue
			}
			if reply.Error == nil {
				return errors.New("Infrai request was not accepted")
			}
			return &InfraiError{Code: reply.Error.Code, Hint: reply.Error.Hint, Status: resp.StatusCode}
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("remote request returned status %d", resp.StatusCode)
		}
		if output != nil && len(reply.Data) > 0 {
			return json.Unmarshal(reply.Data, output)
		}
		return nil
	}
	return errors.New("request retry limit reached")
}

func retryAfter(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}
