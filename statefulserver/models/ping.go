package models

import (
	"encoding/gob"

	"github.com/google/uuid"
)

type Ping struct {
	ID uuid.UUID
}

type Pong struct {
	ID uuid.UUID
}

func init() {
	gob.Register(Ping{})
	gob.Register(Pong{})
}
