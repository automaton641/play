package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)

const playersCount int64 = 2
const attributesCount int64 = 4
const playsCount int64 = 4
const attributesLimit int64 = 1024

var randomSource rand.Source = rand.NewSource(time.Now().UnixNano())
var random *rand.Rand = rand.New(randomSource)

type player struct {
	index      int64
	attributes []int64
}

func newPlayer(index int64) *player {
	player := new(player)
	player.index = index
	player.attributes = make([]int64, attributesCount)
	player.attributes[0] = 16
	player.attributes[1] = 0
	player.attributes[2] = 0
	player.attributes[3] = 0
	return player
}

func (player *player) play() int64 {
	return int64(random.Intn(int(playsCount)))
}

func (player *player) changeAttribute(attributeIndex, offset int64) {
	player.attributes[attributeIndex] += offset
	if player.attributes[attributeIndex] > attributesLimit {
		player.attributes[attributeIndex] = attributesLimit
	} else if player.attributes[attributeIndex] < 0 {
		player.attributes[attributeIndex] = 0
	}
}

type turnMemory struct {
	turn       int64
	play       int64
	attributes [][]int64
}

func newTurnMemory(turn, play int64, attributes [][]int64) *turnMemory {
	turnMemory := new(turnMemory)
	turnMemory.turn = turn
	turnMemory.play = play
	turnMemory.attributes = attributes
	return turnMemory
}

type gameMemory struct {
	winner      int64
	turns       int64
	turnsMemory []*turnMemory
}

func newGameMemory(winner, turns int64) *gameMemory {
	gameMemory := new(gameMemory)
	gameMemory.winner = winner
	gameMemory.turns = turns
	gameMemory.turnsMemory = make([]*turnMemory, 0)
	return gameMemory
}

type game struct {
	memory             *gameMemory
	winner             int64
	shouldRun          bool
	players            []*player
	currentPlayerIndex int64
	currentEnemyIndex  int64
	currentPlay        int64
	turn               int64
}

func newGame() *game {
	game := new(game)
	game.shouldRun = true
	game.winner = -1
	game.players = make([]*player, playersCount)
	for playerIndex := int64(0); playerIndex < playersCount; playerIndex++ {
		game.players[playerIndex] = newPlayer(playerIndex)
	}
	game.memory = newGameMemory(-1, 0)
	return game
}

func (game *game) changeAttribute(playerIndex, attributeIndex, offset int64) {
	game.players[playerIndex].changeAttribute(attributeIndex, offset)
}

func (game *game) changeCurrentPlayerAttribute(attributeIndex, offset int64) {
	game.players[game.currentPlayerIndex].changeAttribute(attributeIndex, offset)
}

func (game *game) changeCurrentEnemyAttribute(attributeIndex, offset int64) {
	game.players[game.currentEnemyIndex].changeAttribute(attributeIndex, offset)
}

func (game *game) getAttribute(playerIndex, attributeIndex int64) int64 {
	return game.players[playerIndex].attributes[attributeIndex]
}

func (game *game) getCurrentPlayerAttribute(attributeIndex int64) int64 {
	return game.getAttribute(game.currentPlayerIndex, attributeIndex)
}

func (game *game) getCurrentEnemyAttribute(attributeIndex int64) int64 {
	return game.getAttribute(game.currentEnemyIndex, attributeIndex)
}

func (game *game) applyPlay() {
	switch game.currentPlay {
	case 0:
		game.changeCurrentPlayerAttribute(0, 256)
		game.changeCurrentEnemyAttribute(2, -32)
		game.changeCurrentEnemyAttribute(3, -32)
	case 1:
		game.changeCurrentPlayerAttribute(2, 16)
		game.changeCurrentPlayerAttribute(3, 32+4)

	case 2:
		game.changeCurrentPlayerAttribute(1, 8)
		game.changeCurrentPlayerAttribute(2, 32)
		game.changeCurrentEnemyAttribute(1, -8)
	case 3:
		offset := (game.getCurrentPlayerAttribute(2)*24+game.getCurrentPlayerAttribute(3)*24)*32 - (game.getCurrentEnemyAttribute(1)*12+game.getCurrentEnemyAttribute(2)*12)*16
		game.changeCurrentEnemyAttribute(0, -offset*(64+32))
	default:
		panic("Unknown play")
	}
}

func (game *game) fightDraw(lifes []int64) {
	if lifes[0] == lifes[1] {
		sums := make([]int64, playersCount)
		for i := int64(1); i < attributesCount; i++ {
			sums[0] += game.getAttribute(0, i)
			sums[1] += game.getAttribute(1, i)
		}
		if sums[0] == sums[1] {
			game.winner = 2
			return
		}
		if sums[0] > sums[1] {
			game.winner = 0
			return
		}
		game.winner = 1
		return
	}
	if lifes[0] > lifes[1] {
		game.winner = 0
		return
	}
	game.winner = 1
	return
}

func (game *game) validateMemory() {
	game.memory.winner = game.winner
	game.memory.turns = game.turn
}

func (game *game) saveTurn() {
	attributes := make([][]int64, playersCount)
	for playerIndex := int64(0); playerIndex < playersCount; playerIndex++ {
		attributes[playerIndex] = make([]int64, attributesCount)
		for attributeIndex := int64(0); attributeIndex < attributesCount; attributeIndex++ {
			attributes[playerIndex][attributeIndex] = game.players[playerIndex].attributes[attributeIndex]
		}
	}
	turnMemory := newTurnMemory(game.turn, game.currentPlay, attributes)
	game.memory.turnsMemory = append(game.memory.turnsMemory, turnMemory)
}

func (game *game) saveMemoryJSON() {
	memory := make([]interface{}, 0)
	fileName := "general.json"
	previousMemory, err := ioutil.ReadFile(fileName)
	shouldPrepend := true
	if err != nil {
		if err.Error() == "open "+fileName+": no such file or directory" {
			fmt.Println("error: no such file or directory")
			shouldPrepend = false
		} else {
			panic(err.Error())
		}
	}
	if shouldPrepend {
		previousMemoryArray := make([]interface{}, 0)
		json.Unmarshal(previousMemory, &previousMemoryArray)
		for _, gameMemory := range previousMemoryArray {
			memory = append(memory, gameMemory)
		}
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	gameMap := make(map[string]interface{})
	gameMap["winner"] = game.memory.winner
	gameMap["turns"] = game.memory.turns
	turnsMemory := make([]map[string]interface{}, len(game.memory.turnsMemory))
	for index, turnMemory := range game.memory.turnsMemory {
		turnsMemory[index] = make(map[string]interface{})
		turnsMemory[index]["attributes"] = turnMemory.attributes
		turnsMemory[index]["play"] = turnMemory.play
		turnsMemory[index]["turn"] = turnMemory.turn
	}
	gameMap["turnsMemory"] = turnsMemory
	memory = append(memory, gameMap)
	memoryJSON, err := json.MarshalIndent(memory, "", "    ")
	if err != nil {
		panic(err)
	}
	if _, err := file.WriteString(string(memoryJSON)); err != nil {
		panic(err)
	}
}

func (game *game) checkWinner() {
	lifes := make([]int64, playersCount)
	lifes[0] = game.getAttribute(0, 0)
	lifes[1] = game.getAttribute(1, 0)
	if lifes[0] <= 0 {
		if lifes[1] <= 0 {
			game.fightDraw(lifes)
			return
		}
		game.winner = 1
		return
	} else if lifes[1] <= 0 {
		game.winner = 0
		return
	} else if game.turn >= 16 {
		game.fightDraw(lifes)
		return
	}
}

func (game *game) run() {
	for game.shouldRun {
		game.currentPlayerIndex = game.turn % playersCount
		game.currentEnemyIndex = (game.currentPlayerIndex + 1) % playersCount
		game.currentPlay = game.players[game.currentPlayerIndex].play()
		game.saveTurn()
		game.applyPlay()
		game.turn++
		game.checkWinner()
		if game.winner >= 0 {
			game.shouldRun = false
		}
	}
	game.validateMemory()
	game.saveMemoryJSON()
}

func loadMemory(fileName string) []*gameMemory {
	memory, err := ioutil.ReadFile(fileName)
	if err != nil {
		if err.Error() == "open "+fileName+": no such file or directory" {
			fmt.Println("error: no such file or directory")
			return make([]*gameMemory, 0)
		}
		panic(err.Error())
	}
	memoryArray := make([]interface{}, 0)
	json.Unmarshal(memory, &memoryArray)
	loadedMemory := make([]*gameMemory, 0)
	for _, game := range memoryArray {
		gameMap := game.(map[string]interface{})
		gameWinner := int64(gameMap["winner"].(float64))
		gameTurns := int64(gameMap["turns"].(float64))
		gameMemory := newGameMemory(gameWinner, gameTurns)
		gameTurnsMemory := gameMap["turnsMemory"].([]interface{})
		for _, turn := range gameTurnsMemory {
			turnMap := turn.(map[string]interface{})
			turnTurn := int64(turnMap["turn"].(float64))
			turnPlay := int64(turnMap["play"].(float64))
			turnAttributes := make([][]int64, playersCount)
			for playerIndex, turnPlayerAttributes := range turnMap["attributes"].([]interface{}) {
				turnAttributes[playerIndex] = make([]int64, attributesCount)
				for attributeIndex, attribute := range turnPlayerAttributes.([]interface{}) {
					turnAttributes[playerIndex][attributeIndex] = int64(attribute.(float64))
				}
			}
			turnMemory := newTurnMemory(turnTurn, turnPlay, turnAttributes)
			gameMemory.turnsMemory = append(gameMemory.turnsMemory, turnMemory)
		}
		loadedMemory = append(loadedMemory, gameMemory)
	}
	return loadedMemory
}

func printMemory() {
	memory := loadMemory("general.json")
	fmt.Println(memory)
}

func main() {
	//game := newGame()
	//game.run()
	printMemory()
}
