package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type dispatchRequest struct {
	Domain             string   `json:"domain"`
	From               string   `json:"from"`
	To                 string   `json:"to"`
	WorkOrderID        string   `json:"work_order_id"`
	Status             string   `json:"status"`
	TechnicianFollowUp string   `json:"technician_follow_up"`
	PhotoURLs          []string `json:"photo_urls"`
	SPF                string   `json:"spf"`
	DKIMName           string   `json:"dkim_name"`
	DKIMContent        string   `json:"dkim_content"`
}

type dispatchResult struct {
	ZoneID             string `json:"zone_id"`
	VerificationStatus string `json:"verification_status"`
	MessageID          string `json:"message_id"`
}

type mailAPI interface {
	call(context.Context, string, string, any, any, string) error
}

func stableID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func dispatch(ctx context.Context, api mailAPI, in dispatchRequest) (dispatchResult, error) {
	var result dispatchResult
	if in.Domain == "" || in.From == "" || in.To == "" || in.WorkOrderID == "" || in.Status == "" || in.SPF == "" || in.DKIMName == "" || in.DKIMContent == "" {
		return result, errors.New("domain, from, to, work_order_id, status, spf, dkim_name and dkim_content are required")
	}
	if !strings.HasSuffix(strings.ToLower(in.From), "@"+strings.ToLower(in.Domain)) {
		return result, errors.New("from must belong to domain")
	}
	for _, photo := range in.PhotoURLs {
		parsed, err := url.Parse(photo)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return result, errors.New("photo_urls must contain HTTPS URLs")
		}
	}
	var zone struct {
		ZoneID string `json:"zone_id"`
	}
	if err := api.call(ctx, "GET", "/v1/dns/domain/get?domain="+url.QueryEscape(in.Domain), nil, &zone, ""); err != nil {
		return result, err
	}
	if zone.ZoneID == "" {
		return result, errors.New("domain has no zone_id")
	}
	result.ZoneID = zone.ZoneID
	for _, record := range []struct{ name, content string }{{in.Domain, in.SPF}, {in.DKIMName, in.DKIMContent}} {
		fields := map[string]any{"zone_id": zone.ZoneID, "record_type": "TXT", "name": record.name, "content": record.content}
		if err := api.call(ctx, "PUT", "/v1/dns/record/upsert", fields, nil, stableID(zone.ZoneID, record.name, record.content)); err != nil {
			return result, err
		}
	}
	var verified struct {
		Verification struct {
			Status string `json:"status"`
		} `json:"verification"`
	}
	if err := api.call(ctx, "POST", "/v1/email/domain/verify", map[string]string{"domain": in.Domain}, &verified, stableID("verify", in.Domain)); err != nil {
		return result, err
	}
	result.VerificationStatus = verified.Verification.Status
	if !strings.EqualFold(result.VerificationStatus, "verified") {
		return result, nil
	}
	lines := []string{"Work order: " + in.WorkOrderID, "Dispatch status: " + in.Status, "Technician follow-up: " + in.TechnicianFollowUp}
	for _, photo := range in.PhotoURLs {
		lines = append(lines, "Photo: "+photo)
	}
	message := map[string]string{"to": in.To, "from": in.From, "subject": "Work order " + in.WorkOrderID + " - " + in.Status, "body": strings.Join(lines, "\n")}
	var sent struct {
		MessageID string `json:"message_id"`
	}
	if err := api.call(ctx, "POST", "/v1/email/send", message, &sent, stableID("dispatch", in.Domain, in.WorkOrderID, in.Status, in.To)); err != nil {
		return result, err
	}
	if sent.MessageID == "" {
		return result, fmt.Errorf("send response missing message_id")
	}
	result.MessageID = sent.MessageID
	return result, nil
}
