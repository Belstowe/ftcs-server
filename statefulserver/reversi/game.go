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

type cell struct {
	X int
	Y int
}

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

func (b Board) outOfBounds(x int, y int) bool {
	return x < 0 || x > 7 || y < 0 || y > 7
}

func (b Board) GetState(x int, y int) CellState {
	if b.outOfBounds(x, y) {
		return CellFree
	}
	return b[y][x]
}

func (b Board) ValidateHasAdjacentOpponentDisk(x int, y int, playerState CellState) bool {
	var opponentState CellState
	if playerState == CellBlack {
		opponentState = CellWhite
	} else {
		opponentState = CellBlack
	}
	return b.GetState(x-1, y-1) == opponentState ||
		b.GetState(x-1, y) == opponentState ||
		b.GetState(x-1, y+1) == opponentState ||
		b.GetState(x, y-1) == opponentState ||
		b.GetState(x, y+1) == opponentState ||
		b.GetState(x+1, y-1) == opponentState ||
		b.GetState(x+1, y) == opponentState ||
		b.GetState(x+1, y+1) == opponentState
}

func (b Board) GetCellsToColor(x int, y int, playerState CellState) []cell {
	cellsToColor := make([]cell, 0)
	for _, direction := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		right, down := direction[0], direction[1]
		tempCellsToColor := make([]cell, 0)
		xshift, yshift := right, down
		for ; !b.outOfBounds(x+xshift, y+yshift) && b.GetState(x+xshift, y+yshift) != playerState; xshift, yshift = xshift+right, yshift+down {
			tempCellsToColor = append(cellsToColor, cell{X: x + xshift, Y: y + yshift})
		}
		if !b.outOfBounds(x+xshift, y+yshift) {
			cellsToColor = append(cellsToColor, tempCellsToColor...)
		}
	}
	return cellsToColor
}

func (b *Board) Move(x int, y int, playerState CellState) (isValidMove bool) {
	if b.GetState(x, y) != CellFree {
		return false
	}
	if !b.ValidateHasAdjacentOpponentDisk(x, y, playerState) {
		return false
	}
	cellsToColor := b.GetCellsToColor(x, y, playerState)
	if len(cellsToColor) == 0 {
		return false
	}
	b[y][x] = playerState
	for _, cellToColor := range cellsToColor {
		b[cellToColor.Y][cellToColor.X] = playerState
	}
	return true
}
