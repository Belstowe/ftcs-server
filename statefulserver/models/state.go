package models

import (
	"encoding/gob"

	"github.com/Belstowe/ftcs-server/statefulserver/reversi"
)

type State struct {
	GameStarted bool
	Board       reversi.Board
	Move        reversi.MoveState
}

func NewState() *State {
	return &State{
		GameStarted: false,
		Board:       *reversi.NewBoard(),
		Move:        reversi.MoveWhite,
	}
}

type RequestState struct{}

type SendState struct {
	State
}

type StateToMaster struct {
	State
}

type StateFromMaster struct {
	State
}

func init() {
	gob.Register(RequestState{})
	gob.Register(StateToMaster{})
	gob.Register(StateFromMaster{})
}
