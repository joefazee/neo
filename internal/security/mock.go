package security

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockMaker struct {
	mock.Mock
}

func (m *MockMaker) CreateToken(userID uuid.UUID, duration time.Duration, version int64, scope string) (string, *Payload, error) {
	args := m.Called(userID, duration, version, scope)
	if args.Get(1) == nil {
		return args.String(0), nil, args.Error(2)
	}
	return args.String(0), args.Get(1).(*Payload), args.Error(2)
}

func (m *MockMaker) VerifyToken(token string) (*Payload, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Payload), args.Error(1)
}
