package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	_ "github.com/mattn/go-sqlite3"
)

func main() {

	menu := []string{
		"Add Card",
		"Review Deck",
		"Create Deck",
		"Quit",
	}
	file := "pankito"
	db, err := newDb(file)
	if err != nil {
		log.Fatal("Error when starting db", err)
	}

	OpenMenu(menu, db)
}

func OpenMenu(menu []string, db *DB) {
	option := SelectOption(menu)
	if option == menu[0] {
		card := CreateCard(db)
		AddNewCard(db, card)
		OpenMenu(menu, db)
	} else if option == menu[1] {
		deckId := GetDeckOfCard(db)
		StartDeckReview(GetCardsToReview(db, deckId), db)
		OpenMenu(menu, db)
	} else if option == menu[2] {
		deck := CreateDeck()
		AddNewDeck(db, deck)
		OpenMenu(menu, db)
	} else if option == menu[3] {
		os.Exit(0)
	}
}

type DB struct {
	db *sql.DB
}

type BaseCard struct {
	Id         int
	DeckId     int
	Front      string
	Back       string
	Interval   int
	EaseFactor float32
	Repetition int
	ReviewDate time.Time
}

type BaseDeck struct {
	Id   int
	Name string
}

func newDb(file string) (*DB, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return nil, errors.New("cannot instantiate a db with empty/whitespace name")
	}
	create := "CREATE TABLE IF NOT EXISTS [Cards] ( Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, DeckId INTEGER NOT NULL,Front TEXT NOT NULL, Back TEXT NOT NULL, Interval INTEGER NOT NULL, EaseFactor DECIMAL(10,8) NOT NULL, Repetition INTEGER NOT NULL, ReviewDate DATETIME NOT NULL, FOREIGN KEY(DeckId) REFERENCES Decks(Id)); CREATE TABLE IF NOT EXISTS [Decks] ( Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, Name TEXT NOT NULL);"

	db, err := sql.Open("sqlite3", file+".db")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(create); err != nil {
		return nil, err
	}

	return &DB{
		db: db,
	}, nil
}

func AddNewDeck(db *DB, deck BaseDeck) {
	stmt := "INSERT INTO Decks(Name) VALUES (?)"
	if _, err := db.db.Exec(stmt, deck.Name); err != nil {
		log.Fatal("Failed to execute INSERT", err)
	}
	log.Printf("Deck added")
}

func AddNewCard(db *DB, card BaseCard) {
	stmt := "INSERT INTO Cards(DeckId, Front, Back, Interval, EaseFactor, Repetition, ReviewDate) VALUES (?, ?, ?, ?, ?, ?, ?)"
	if _, err := db.db.Exec(stmt, card.DeckId, card.Front, card.Back, 0, 2.5, 0, time.Now()); err != nil {
		log.Fatal("Failed to execute INSERT", err)
	}
	log.Printf("Card added")
}

func AddCardsToReview(db *DB) []BaseCard {
	deck := make([]BaseCard, 0)
	for i := 0; i < 10; i++ {
		deck = append(deck, BaseCard{
			Front:      fmt.Sprintf("Test %v?", i),
			Back:       fmt.Sprintf("Answer %v", i),
			Interval:   0,
			EaseFactor: 2.5,
			Repetition: 0,
			ReviewDate: time.Now(),
		})
		AddNewCard(db, deck[len(deck)-1])
	}
	return deck
}

func CreateCard(db *DB) BaseCard {
	deckId := GetDeckOfCard(db)
	front := GetFrontOfCard()
	back := GetBackOfCard()

	return BaseCard{
		Id:         0,
		DeckId:     deckId,
		Front:      front,
		Back:       back,
		Interval:   0,
		EaseFactor: 0,
		Repetition: 0,
		ReviewDate: time.Time{},
	}

}

func GetDeckOfCard(db *DB) int {
	decks := GetExistingDecks(db)
	templates := &promptui.SelectTemplates{
		Active:   "➡️ {{ .Name | blue | underline}}",
		Inactive: " {{ .Name | faint }}",
		Selected: "✅ {{ .Name | green }}",
	}

	searcher := func(input string, index int) bool {
		deck := decks[index]
		name := strings.Replace(strings.ToLower(deck.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}
	prompt := promptui.Select{
		Label:     "Select deck",
		Items:     decks,
		Templates: templates,
		Searcher:  searcher,
	}

	i, _, err := prompt.Run()

	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}
	return decks[i].Id

}

func GetExistingDecks(db *DB) []BaseDeck {
	stmt := "SELECT Id, Name FROM Decks ORDER BY Id"
	rows, err := db.db.Query(stmt)
	if err != nil {
		log.Fatal("Error querying for cards", err)
	}

	defer rows.Close()

	data := []BaseDeck{}
	for rows.Next() {
		i := BaseDeck{}
		err = rows.Scan(&i.Id, &i.Name)
		if err != nil {
			log.Printf("Error occurred whilst mapping decks Id: %v - error: %v", &i.Id, err)
		}
		data = append(data, i)
	}
	return data
}

func CreateDeck() BaseDeck {
	name := GetNameOfDeck()
	return BaseDeck{
		Id:   0,
		Name: name,
	}
}

func GetBackOfCard() string {
	validate := func(input string) error {
		if len(input) == 0 {
			return errors.New("back of card cannot be empty")
		}
		return nil
	}

	username := ""

	prompt := promptui.Prompt{
		Label:    "Back of card",
		Validate: validate,
		Default:  username,
	}

	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
	}
	return result
}

func GetFrontOfCard() string {
	validate := func(input string) error {
		if len(input) == 0 {
			return errors.New("front of card cannot be empty")
		}
		return nil
	}

	username := ""

	prompt := promptui.Prompt{
		Label:    "Front of card",
		Validate: validate,
		Default:  username,
	}

	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
	}
	return result
}

func GetNameOfDeck() string {
	validate := func(input string) error {
		if len(input) == 0 {
			return errors.New("name of deck cannot be empty")
		}
		return nil
	}

	name := ""

	prompt := promptui.Prompt{
		Label:    "Deck name",
		Validate: validate,
		Default:  name,
	}

	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
	}
	return result
}

func GetCardsToReview(db *DB, deckId int) []BaseCard {
	stmt := "SELECT * FROM Cards WHERE datetime(ReviewDate) <= datetime('now') AND DeckId = ? ORDER BY ReviewDate"
	rows, err := db.db.Query(stmt, deckId)
	if err != nil {
		log.Fatal("Error querying for cards", err)
	}

	defer rows.Close()

	data := []BaseCard{}
	for rows.Next() {
		i := BaseCard{}
		err = rows.Scan(&i.Id, &i.DeckId, &i.Front, &i.Back, &i.Interval, &i.EaseFactor, &i.Repetition, &i.ReviewDate)
		if err != nil {
			log.Printf("Error occurred whilst mapping cards Id: %v - error: %v", &i.Id, err)
		}
		data = append(data, i)
	}
	return data
}

func StartDeckReview(deck []BaseCard, db *DB) []BaseCard {
	if len(deck) == 0 {
		fmt.Print("No cards to review! 🎉 \n ")
		return nil
	} else {
		updatedDeck := ReviewCard(deck, db)
		return StartDeckReview(updatedDeck, db)
	}
}

func UpdateReviewDeck(reviewDeck []BaseCard, pop bool) []BaseCard {
	updatedReviewDeck := make([]BaseCard, 0)
	if pop {
		return append(updatedReviewDeck, reviewDeck[1:]...)
	} else {
		updatedReviewDeck = append(updatedReviewDeck, reviewDeck[1:]...)
		return append(updatedReviewDeck, reviewDeck[0])
	}
}

func ReviewCard(reviewDeck []BaseCard, db *DB) []BaseCard {
	card := &reviewDeck[0]
	qualityString := ViewFrontAndBack(card)
	quality := ParseInput(qualityString)
	pop := UpdateCard(card, quality, db)

	return UpdateReviewDeck(reviewDeck, pop)
}

func ViewFrontAndBack(card *BaseCard) string {
	ViewFront(card)
	ViewBack(card)
	input := SelectQuality()
	return input
}

func ViewFront(card *BaseCard) {
	format := fmt.Sprintf("\x1b[%dm\n > %s \x1b[0m", 34, "Press 'Enter' to show answer")
	fmt.Println(format)
	format = fmt.Sprintf("\x1b[%dm\n%s\x1b[0m", 93, card.Front)
	fmt.Println(format)
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func ViewBack(card *BaseCard) {
	format := fmt.Sprintf("\x1b[%dm%s\n\x1b[0m", 32, card.Back)
	fmt.Println(format)
}

func SelectOption(menu []string) string {
	prompt := promptui.Select{
		Label: "Menu",
		Items: menu,
	}

	_, result, err := prompt.Run()

	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}

	return result
}

func SelectQuality() string {
	prompt := promptui.Select{
		Label: "Quality of answer",
		Items: []string{"0", "1", "2", "3", "4", "5"},
	}

	_, result, err := prompt.Run()

	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}

	return result
}

func UpdateCard(card *BaseCard, quality float32, db *DB) bool {
	if quality > 3 {
		card.Repetition = card.Repetition + 1
		card.EaseFactor = CalculateEaseFactor(card.EaseFactor, quality)
		card.Interval = CalculateInterval(card.Repetition, card.Interval, card.EaseFactor)
		card.ReviewDate = truncateToDay(card.ReviewDate.AddDate(0, 0, card.Interval))
		_, err := db.db.Exec("UPDATE Cards SET Repetition = ?, EaseFactor = ?, Interval = ?, ReviewDate = ? WHERE Id = ?;", card.Repetition, card.EaseFactor, card.Interval, card.ReviewDate, card.Id)
		if err != nil {
			fmt.Printf("Failed to update card Id: %v with error: %v", card.Id, err)
		}
		return true
	}

	card.Repetition = 0
	card.EaseFactor = CalculateEaseFactor(card.EaseFactor, quality)
	card.Interval = CalculateInterval(card.Repetition, card.Interval, card.EaseFactor)
	_, err := db.db.Exec("UPDATE Cards SET Repetition = ?, EaseFactor = ?, Interval = ? WHERE Id = ?;", card.Repetition, card.EaseFactor, card.Interval, card.Id)
	if err != nil {
		fmt.Printf("Failed to update card Id: %v with error: %v", card.Id, err)
	}

	return false
}

func ParseInput(input string) float32 {
	input = strings.TrimSpace(input)
	quality64, err := strconv.ParseFloat(input, 32)
	if err != nil {
		log.Fatal(err)
	}
	quality := float32(quality64)
	return quality
}

func CalculateEaseFactor(ef float32, quality float32) float32 {
	updatedEaseFactor := (ef) + (0.1 - (5-quality)*(0.8+(5-quality)*0.02))
	if updatedEaseFactor < 1.3 {
		return float32(1.3)
	}
	return updatedEaseFactor
}

func CalculateInterval(repetition int, previousInterval int, ef float32) int {
	if repetition <= 1 {
		return 1
	} else if repetition == 2 {
		return 6
	} else {
		return int(math.RoundToEven(float64((float32(previousInterval) * ef))))
	}
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
