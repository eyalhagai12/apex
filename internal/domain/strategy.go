package domain

import (
	"strconv"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusInactive Status = "inactive"
	StatusActive   Status = "active"
	StatusError    Status = "error"
)

type Strategy struct {
	ID         uuid.UUID
	Name       string
	Status     Status
	Version    uint64
	Identifier string
	Path       string
}

func (s *Strategy) FileName() string {
	return "Version-" + strconv.FormatUint(s.Version, 10)
}

func NewStrategy(name string) *Strategy {
	id := uuid.New()
	return &Strategy{
		ID:      id,
		Name:    name,
		Status:  StatusPending,
		Version: 0,
	}
}
