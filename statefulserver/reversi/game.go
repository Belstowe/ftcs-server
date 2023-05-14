package reversi

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

type Board [8][8]CellState

func NewBoard() *Board {
	var cells Board
	for cellRow := range cells {
		for cellIndex := range cells[cellRow] {
			cells[cellRow][cellIndex] = CellFree
		}
	}
	cells[3][3] = CellWhite
	cells[3][4] = CellBlack
	cells[4][3] = CellBlack
	cells[4][4] = CellWhite
	return &cells
}
