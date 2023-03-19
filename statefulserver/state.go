package statefulserver

import "github.com/google/uuid"

type SharedState struct {
	ID uuid.UUID
}

func NewSharedState() SharedState {
	return SharedState{ID: uuid.New()}
}
