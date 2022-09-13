package game

const (
	ROCK     = 0
	PAPER    = 1
	SCISSORS = 2
)

func (game *Game) CheckResult() {
	turns := game.Turns
	rockGesture, paperGesture, scissorsGesture := game.calculateGestures()
	if rockGesture.Count > 0 && paperGesture.Count > 0 && scissorsGesture.Count > 0 ||
		rockGesture.Count == game.Players ||
		paperGesture.Count == game.Players ||
		scissorsGesture.Count == game.Players {
		game.Winner = append(game.Winner, -1)
		game.WinnerGesture = 255
		return
	}
	var winningGesture uint8
	switch {
	case rockGesture.Count == 0:
		winningGesture = SCISSORS
	case paperGesture.Count == 0:
		winningGesture = ROCK
	case scissorsGesture.Count == 0:
		winningGesture = PAPER
	}
	for _, turn := range turns {
		if turn.Gesture == winningGesture {
			game.Winner = append(game.Winner, turn.UserId)
		}
	}
	game.WinnerGesture = winningGesture
}

func (game *Game) Save() {
	DB.Save(&game)
}

func (game *Game) Create() {
	DB.Create(&game)
}

func (game *Game) calculateGestures() (rGesture GestureStat, pGesture GestureStat, sGesture GestureStat) {
	turns := game.Turns
	rGesture, pGesture, sGesture = GestureStat{0, 0}, GestureStat{1, 0}, GestureStat{2, 0}
	for i := 0; i < len(turns); i++ {
		switch turns[i].Gesture {
		case ROCK:
			rGesture.Count += 1
		case PAPER:
			pGesture.Count += 1
		case SCISSORS:
			sGesture.Count += 1
		}
	}
	return rGesture, pGesture, sGesture
}
