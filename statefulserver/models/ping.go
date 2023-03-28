package models

import "encoding/gob"

type Ping struct{}

type Pong struct{}

func init() {
	gob.Register(Ping{})
	gob.Register(Pong{})
}
