package attendeeservice

import (
	"context"
)

type AttendeeService interface {
	// PaymentsChanged webhook to tell the attendee service to recalculate attendee status.
	PaymentsChanged(ctx context.Context, debitorId uint) error

	// ListMyRegistrationIds which debitorIds should be visible for a non-admin / non-api-token user?
	//
	// If your request was made by an admin or with the api token, this will fail and should not be called.
	// Admin and api token can view all transactions anyway.
	//
	// Forwards the jwt from the request.
	ListMyRegistrationIds(ctx context.Context) ([]int64, error)
}
