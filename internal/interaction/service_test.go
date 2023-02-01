package interaction

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/database/inmemory"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/attendeeservice"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"
)

func TestNewServiceInteractor(t *testing.T) {
	type args struct {
		repo      database.Repository
		attClient attendeeservice.AttendeeService
		ccClient  cncrdadapter.CncrdAdapter
	}

	type expected struct {
		err error
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "should return error when repository is missing",
			expected: expected{
				err: errors.New("repository must not be nil"),
			},
		},
		{
			name: "should return error when attendee client is missing",
			args: args{
				repo: inmemory.NewInMemoryProvider(),
			},
			expected: expected{
				err: errors.New("no attendee service client provided"),
			},
		},
		{
			name: "should return error when payment adapter client is missing",
			args: args{
				repo:      inmemory.NewInMemoryProvider(),
				attClient: &AttendeeServiceMock{},
			},
			expected: expected{
				err: errors.New("no payment adapter client provided"),
			},
		},
		{
			name: "should succeed when all values are set",
			args: args{
				repo:      inmemory.NewInMemoryProvider(),
				attClient: &AttendeeServiceMock{},
				ccClient:  &CncrdAdapterMock{},
			},
			expected: expected{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewServiceInteractor(tt.args.repo, tt.args.attClient, tt.args.ccClient, &securityConfig)
			if tt.expected.err != nil {
				require.EqualError(t, err, tt.expected.err.Error())
				require.Nil(t, i)
			} else {
				require.NoError(t, err)
				require.NotNil(t, i)
			}
		})
	}

}
