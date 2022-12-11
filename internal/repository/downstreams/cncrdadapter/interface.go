package cncrdadapter

import (
	"context"
)

type CncrdAdapter interface {
	CreatePaylink(ctx context.Context, request PaymentLinkRequestDto) (PaymentLinkDto, error)
	GetPaylinkById(ctx context.Context, id uint) (PaymentLinkDto, error)
}

type PaymentLinkRequestDto struct {
	DebitorId int64   `json:"debitor_id"`
	AmountDue int64   `json:"amount_due"`
	Currency  string  `json:"currency"`
	VatRate   float64 `json:"vat_rate"`
}

type PaymentLinkDto struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	ReferenceId string  `json:"reference_id"`
	Purpose     string  `json:"purpose"`
	AmountDue   int64   `json:"amount_due"`
	AmountPaid  int64   `json:"amount_paid"`
	Currency    string  `json:"currency"`
	VatRate     float64 `json:"vat_rate"`
	Link        string  `json:"link"`
}
