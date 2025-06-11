package main

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/a-h/templ"
)

// Piece represents a chess piece
type Piece string

const (
	Empty       Piece = ""
	WhitePawn   Piece = "♙"
	WhiteRook   Piece = "♖"
	WhiteKnight Piece = "♘"
	WhiteBishop Piece = "♗"
	WhiteQueen  Piece = "♕"
	WhiteKing   Piece = "♔"
	BlackPawn   Piece = "♟"
	BlackRook   Piece = "♜"
	BlackKnight Piece = "♞"
	BlackBishop Piece = "♝"
	BlackQueen  Piece = "♛"
	BlackKing   Piece = "♚"
)

// Square represents a square on the board
type Square struct {
	Row int
	Col int
}

// PieceColor represents the color of a piece
type PieceColor string

const (
	White PieceColor = "white"
	Black PieceColor = "black"
)

// GameState holds the current state of the chess game.
type GameState struct {
	Board          [8][8]Piece
	CurrentPlayer  PieceColor
	SelectedSquare *Square
	mu             sync.Mutex
}

// Global game state (for simplicity in this example)
var game *GameState

func (gs *GameState) ResetBoard() {
	gs.Board = [8][8]Piece{
		{BlackRook, BlackKnight, BlackBishop, BlackQueen, BlackKing, BlackBishop, BlackKnight, BlackRook},
		{BlackPawn, BlackPawn, BlackPawn, BlackPawn, BlackPawn, BlackPawn, BlackPawn, BlackPawn},
		{Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty},
		{Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty},
		{Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty},
		{Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty},
		{WhitePawn, WhitePawn, WhitePawn, WhitePawn, WhitePawn, WhitePawn, WhitePawn, WhitePawn},
		{WhiteRook, WhiteKnight, WhiteBishop, WhiteQueen, WhiteKing, WhiteBishop, WhiteKnight, WhiteRook},
	}
	gs.CurrentPlayer = White
	gs.SelectedSquare = nil
}

func main() {
	// Initialize the game state
	game = &GameState{}
	game.ResetBoard()

	http.HandleFunc("/", handleGetBoard)
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/reset", handleReset)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func handleGetBoard(w http.ResponseWriter, r *http.Request) {
	templ.Handler(page(game)).ServeHTTP(w, r)
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	game.mu.Lock()
	defer game.mu.Unlock()
	game.ResetBoard()
	templ.Handler(chessboardWithLabels(game)).ServeHTTP(w, r)
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	row, _ := strconv.Atoi(r.FormValue("row"))
	col, _ := strconv.Atoi(r.FormValue("col"))
	to := Square{Row: row, Col: col}

	game.mu.Lock()
	defer game.mu.Unlock()

	if game.SelectedSquare == nil {
		// Attempt to select a piece
		if game.Board[to.Row][to.Col] != Empty && isCorrectPlayer(game.Board[to.Row][to.Col], game.CurrentPlayer) {
			game.SelectedSquare = &to
		}
	} else {
		// A piece is already selected, attempt to move it
		from := game.SelectedSquare

		// Deselect if clicking the same square
		if from.Row == to.Row && from.Col == to.Col {
			game.SelectedSquare = nil
			templ.Handler(chessboardWithLabels(game)).ServeHTTP(w, r)
			return
		}

		// Check if the move is valid according to chess rules
		if isValidMove(game, *from, to) {
			// Move the piece
			game.Board[to.Row][to.Col] = game.Board[from.Row][from.Col]
			game.Board[from.Row][from.Col] = Empty

			// Switch player
			if game.CurrentPlayer == White {
				game.CurrentPlayer = Black
			} else {
				game.CurrentPlayer = White
			}
		}
		// Deselect after any move attempt (valid or invalid)
		game.SelectedSquare = nil
	}

	templ.Handler(chessboardWithLabels(game)).ServeHTTP(w, r)
}

// isValidMove checks if a move is valid for the given piece type.
func isValidMove(g *GameState, from, to Square) bool {
	piece := g.Board[from.Row][from.Col]
	targetPiece := g.Board[to.Row][to.Col]

	// Cannot capture your own piece
	if targetPiece != Empty && isCorrectPlayer(targetPiece, g.CurrentPlayer) {
		return false
	}

	switch piece {
	case WhitePawn, BlackPawn:
		return isValidPawnMove(g, from, to)
	case WhiteRook, BlackRook:
		return isValidRookMove(g, from, to)
	case WhiteKnight, BlackKnight:
		return isValidKnightMove(from, to)
	case WhiteBishop, BlackBishop:
		return isValidBishopMove(g, from, to)
	case WhiteQueen, BlackQueen:
		return isValidQueenMove(g, from, to)
	case WhiteKing, BlackKing:
		return isValidKingMove(from, to)
	}
	return false
}

// isValidPawnMove checks pawn-specific move logic.
func isValidPawnMove(g *GameState, from, to Square) bool {
	targetPiece := g.Board[to.Row][to.Col]
	rowDiff := to.Row - from.Row
	colDiff := to.Col - from.Col

	if g.CurrentPlayer == White {
		// Move one step forward
		if colDiff == 0 && targetPiece == Empty && rowDiff == -1 {
			return true
		}
		// Move two steps forward from start
		if colDiff == 0 && targetPiece == Empty && from.Row == 6 && rowDiff == -2 && g.Board[from.Row-1][from.Col] == Empty {
			return true
		}
		// Capture
		if math.Abs(float64(colDiff)) == 1 && rowDiff == -1 && targetPiece != Empty {
			return true
		}
	} else { // Black Player
		// Move one step forward
		if colDiff == 0 && targetPiece == Empty && rowDiff == 1 {
			return true
		}
		// Move two steps forward from start
		if colDiff == 0 && targetPiece == Empty && from.Row == 1 && rowDiff == 2 && g.Board[from.Row+1][from.Col] == Empty {
			return true
		}
		// Capture
		if math.Abs(float64(colDiff)) == 1 && rowDiff == 1 && targetPiece != Empty {
			return true
		}
	}
	return false
}

// isValidRookMove checks if the move is a valid straight line and the path is clear.
func isValidRookMove(g *GameState, from, to Square) bool {
	if from.Row != to.Row && from.Col != to.Col {
		return false // Not a straight line
	}
	return isPathClear(g, from, to)
}

// isValidKnightMove checks for the L-shaped knight move.
func isValidKnightMove(from, to Square) bool {
	absRowDiff := math.Abs(float64(to.Row - from.Row))
	absColDiff := math.Abs(float64(to.Col - from.Col))
	return (absRowDiff == 2 && absColDiff == 1) || (absRowDiff == 1 && absColDiff == 2)
}

// isValidBishopMove checks if the move is a valid diagonal and the path is clear.
func isValidBishopMove(g *GameState, from, to Square) bool {
	if math.Abs(float64(to.Row-from.Row)) != math.Abs(float64(to.Col-from.Col)) {
		return false // Not a diagonal
	}
	return isPathClear(g, from, to)
}

// isValidQueenMove combines rook and bishop logic.
func isValidQueenMove(g *GameState, from, to Square) bool {
	isStraight := from.Row == to.Row || from.Col == to.Col
	isDiagonal := math.Abs(float64(to.Row-from.Row)) == math.Abs(float64(to.Col-from.Col))
	if !isStraight && !isDiagonal {
		return false
	}
	return isPathClear(g, from, to)
}

// isValidKingMove checks for a one-square move in any direction.
func isValidKingMove(from, to Square) bool {
	absRowDiff := math.Abs(float64(to.Row - from.Row))
	absColDiff := math.Abs(float64(to.Col - from.Col))
	return absRowDiff <= 1 && absColDiff <= 1
}

// isPathClear checks if there are any pieces between 'from' and 'to'.
func isPathClear(g *GameState, from, to Square) bool {
	rowStep := 0
	if to.Row > from.Row {
		rowStep = 1
	} else if to.Row < from.Row {
		rowStep = -1
	}

	colStep := 0
	if to.Col > from.Col {
		colStep = 1
	} else if to.Col < from.Col {
		colStep = -1
	}

	currRow, currCol := from.Row+rowStep, from.Col+colStep
	for currRow != to.Row || currCol != to.Col {
		if g.Board[currRow][currCol] != Empty {
			return false // Path is blocked
		}
		currRow += rowStep
		currCol += colStep
	}
	return true // Path is clear
}

// isCorrectPlayer checks if a piece belongs to the current player.
func isCorrectPlayer(p Piece, player PieceColor) bool {
	isWhite := isWhitePieceMove(p)
	if player == White {
		return isWhite
	}
	return !isWhite
}

// isWhitePieceMove determines the color of a piece.
func isWhitePieceMove(p Piece) bool {
	switch p {
	case WhitePawn, WhiteRook, WhiteKnight, WhiteBishop, WhiteQueen, WhiteKing:
		return true
	default:
		return false
	}
}
