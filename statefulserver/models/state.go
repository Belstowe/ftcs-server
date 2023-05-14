package models

import "encoding/gob"

type CellState byte

const (
	CellFree  CellState = iota
	CellWhite CellState = iota
	CellBlack CellState = iota
)

type MoveState byte

const (
	MoveWhite MoveState = iota
	MoveBlack MoveState = iota
	WinWhite  MoveState = iota
	WinBlack  MoveState = iota
)

type State struct {
	GameStarted bool
	Board       [8][8]CellState
	Move        MoveState
}

func NewState() *State {
	var cells [8][8]CellState
	for cellRow := range cells {
		for cellIndex := range cells[cellRow] {
			cells[cellRow][cellIndex] = CellFree
		}
	}
	cells[3][3] = CellWhite
	cells[3][4] = CellBlack
	cells[4][3] = CellBlack
	cells[4][4] = CellWhite
	return &State{
		GameStarted: false,
		Board:       cells,
		Move:        MoveWhite,
	}
}

type RequestState struct{}

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
