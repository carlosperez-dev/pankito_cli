package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	_ "github.com/mattn/go-sqlite3"
)

func main() {

	menu := []string{
		"Review",
		"Add Card",
		"Create Deck",
		"Delete Card",
		"Delete Deck",
		"Quit",
	}

	file := "playita"

	db, err := newDb(file)
	if err != nil {
		log.Fatal("Error when starting db", err)
	}
	for {
		OpenMenu(menu, db)
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

type BaseDeckWithCardCount struct {
	Id            int
	Name          string
	CardsToReview int
}

type ReviewDeck struct {
	Cards []BaseCard
}

func OpenMenu(menu []string, db *DB) {
	menuOptions := selectOption(menu, "Menu")
	creationOptions := []string{"Add more", "Return to menu"}
	deleteOptions := []string{"Continue deleting", "Return to menu"}
	if menuOptions == menu[0] {
		ReviewHandler(db)
	} else if menuOptions == menu[1] {
		AddCardHandler(db, creationOptions)
	} else if menuOptions == menu[2] {
		AddDeckHandler(db, creationOptions)
	} else if menuOptions == menu[3] {
		// DeleteCardHandler(db, deleteOptions)
	} else if menuOptions == menu[4] {
		DeleteDeckHandler(db, deleteOptions)
	} else if menuOptions == menu[5] {
		os.Exit(0)
	}
}

func DeleteDeckHandler(db *DB, deletionOptions []string) {
	decks := db.getExistingDecks()
	if len(decks) == 0 {
		clearConsole()
		fmt.Print("No decks found 😔 \n ")
		return
	}
	templates := &promptui.SelectTemplates{
		Active:   "▸ {{.Name }}",
		Inactive: "  {{.Name| faint }} ",
		Selected: "✔ {{.Name| green }}",
	}

	searcher := func(input string, index int) bool {
		deck := decks[index]
		name := strings.Replace(strings.ToLower(deck.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}
	prompt := promptui.Select{
		Label:             "Decks",
		Items:             decks,
		Templates:         templates,
		Searcher:          searcher,
		StartInSearchMode: true,
		HideHelp:          true,
		Size:              4,
	}

	i, _, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}

	confirmDelete(db, decks[i].Id)
	postDeleteDeckMenu(db, deletionOptions)
}

func postDeleteDeckMenu(db *DB, deletionOptions []string) {
	j := selectOption(deletionOptions, "Options")
	if j == deletionOptions[0] {
		DeleteDeckHandler(db, deletionOptions)
	} else if j == deletionOptions[1] {
		clearConsole()
	}
}

func confirmDelete(db *DB, deckId int) {
	prompt := promptui.Prompt{
		Label:     "Delete",
		IsConfirm: true,
		Default:   "y",
	}
	validate := func(s string) error {
		if len(s) == 1 && strings.Contains("YyNn", s) || prompt.Default != "" && len(s) == 0 {
			return nil
		}
		return errors.New("invalid input")
	}
	prompt.Validate = validate

	_, err := prompt.Run()
	confirmed := !errors.Is(err, promptui.ErrAbort)
	if err != nil && confirmed {
		fmt.Println("ERROR: ", err)
		return
	}
	if confirmed {
		deleteDeck(db, deckId)
	} else {
		fmt.Println("Delete cancelled")
		return
	}
}

func deleteDeck(db *DB, deckId int) {
	_, err := db.db.Exec("DELETE FROM Decks WHERE Id = ?;", deckId)
	if err != nil {
		fmt.Printf("Failed to delete deck id: %v with error: %v", deckId, err)
		return
	}
	fmt.Println("Deck Deleted")
}

func ReviewHandler(db *DB) {
	deckId := getDeckOfCardForReview(db)
	if deckId == 0 {
		return
	} else {
		deck := db.getCardsToReview(deckId)
		deck.review(db)
		return
	}
}

func AddDeckHandler(db *DB, creationOptions []string) {
	deck := createDeck()
	db.addNewDeck(deck)
	postAddDeckMenu(db, creationOptions)
}

func postAddDeckMenu(db *DB, creationOptions []string) {
	j := selectOption(creationOptions, "Options")
	if j == creationOptions[0] {
		AddDeckHandler(db, creationOptions)
	} else if j == creationOptions[1] {
		clearConsole()
	}
}

func AddCardHandler(db *DB, creationOptions []string, params ...int) {
	var card *BaseCard
	if len(params) == 0 {
		card = createCard(db)
	} else if len(params) == 1 {
		card = createCard(db, params[0])
	}
	db.addNewCard(card)
	postAddCardMenu(db, creationOptions, card.DeckId)
}

func postAddCardMenu(db *DB, creationOptions []string, deckId int) {
	j := selectOption(creationOptions, "Options")
	if j == creationOptions[0] {
		AddCardHandler(db, creationOptions)
	} else if j == creationOptions[1] {
		clearConsole()
	}
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

func (db *DB) addNewDeck(deck *BaseDeck) {
	stmt := "INSERT INTO Decks(Name) VALUES (?)"
	if _, err := db.db.Exec(stmt, deck.Name); err != nil {
		log.Fatal("Failed to execute INSERT", err)
	}
	fmt.Printf("Deck %s succesfully added!", deck.Name)
}

func (db *DB) addNewCard(card *BaseCard) {
	stmt := "INSERT INTO Cards(DeckId, Front, Back, Interval, EaseFactor, Repetition, ReviewDate) VALUES (?, ?, ?, ?, ?, ?, ?)"
	if _, err := db.db.Exec(stmt, card.DeckId, card.Front, card.Back, 0, 2.5, 0, time.Now()); err != nil {
		log.Fatal("Failed to execute INSERT", err)
	}
	fmt.Printf("Card succesfully added!")
}

func createCard(db *DB, params ...int) *BaseCard {
	var deckId int
	if len(params) == 0 {
		deckId = getDeckOfCard(db)
		if deckId == 0 {
			fmt.Print("Let's create one! \n ")
			deck := createDeck()
			db.addNewDeck(deck)
			deckId = getDeckOfCard(db)
		}
	} else if len(params) == 1 {
		deckId = params[0]
	}
	front := getFrontOfCard()
	back := getBackOfCard()

	return &BaseCard{
		Id:         0,
		DeckId:     deckId,
		Front:      front,
		Back:       back,
		Interval:   0,
		EaseFactor: 0,
		Repetition: 0,
		ReviewDate: time.Now(),
	}

}

func getDeckOfCardForReview(db *DB) int {
	decks := db.getExistingDecksWithCardCount()
	if len(decks) == 0 {
		clearConsole()
		fmt.Print("No cards to review today 🥳 \n ")
		return 0
	}
	templates := &promptui.SelectTemplates{
		Active:   "▸ {{.Name }} ({{.CardsToReview }})",
		Inactive: "  {{.Name| faint }} {{ .CardsToReview | faint}}",
		Selected: "✔ {{.Name| green }} {{ .CardsToReview | green }}",
	}

	searcher := func(input string, index int) bool {
		deck := decks[index]
		name := strings.Replace(strings.ToLower(deck.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}
	prompt := promptui.Select{
		Label:             "Decks (# to review)",
		Items:             decks,
		Templates:         templates,
		Searcher:          searcher,
		StartInSearchMode: true,
		HideHelp:          true,
		Size:              4,
	}

	i, _, err := prompt.Run()

	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}
	return decks[i].Id
}

func getDeckOfCard(db *DB) int {
	decks := db.getExistingDecks()
	if len(decks) == 0 {
		fmt.Print("No decks found 😔 \n ")
		return 0
	}
	templates := &promptui.SelectTemplates{
		Active:   "▸ {{.Name }}",
		Inactive: "  {{.Name| faint }} ",
		Selected: "✔ {{.Name| green }}",
	}

	searcher := func(input string, index int) bool {
		deck := decks[index]
		name := strings.Replace(strings.ToLower(deck.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}
	prompt := promptui.Select{
		Label:             "Decks",
		Items:             decks,
		Templates:         templates,
		Searcher:          searcher,
		StartInSearchMode: true,
		HideHelp:          true,
		Size:              4,
	}

	i, _, err := prompt.Run()

	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}
	return decks[i].Id

}

func (db *DB) getExistingDecks() []BaseDeck {
	stmt := "SELECT Id, Name FROM Decks ORDER BY Id"
	rows, err := db.db.Query(stmt)
	if err != nil {
		log.Fatal("Error querying for cards", err)
	}

	defer rows.Close()

	decks := []BaseDeck{}
	for rows.Next() {
		i := BaseDeck{}
		err = rows.Scan(&i.Id, &i.Name)
		if err != nil {
			log.Printf("Error occurred whilst mapping decks Id: %v - error: %v", &i.Id, err)
		}
		decks = append(decks, i)
	}
	return decks
}

func (db *DB) getExistingDecksWithCardCount() []BaseDeckWithCardCount {
	stmt := "SELECT Decks.Id, Decks.Name, COUNT(Cards.Id) AS CardCount FROM Decks JOIN Cards ON Cards.DeckId = Decks.Id WHERE datetime(Cards.ReviewDate) <= datetime('now') GROUP BY Decks.Id ORDER BY Decks.Id;"
	rows, err := db.db.Query(stmt)
	if err != nil {
		log.Fatal("Error querying for cards", err)
	}
	defer rows.Close()

	decks := []BaseDeckWithCardCount{}
	for rows.Next() {
		i := BaseDeckWithCardCount{}
		err = rows.Scan(&i.Id, &i.Name, &i.CardsToReview)
		if err != nil {
			log.Printf("Error occurred whilst mapping decks Id: %v - error: %v", &i.Id, err)
		}
		decks = append(decks, i)
	}
	return decks
}

func createDeck() *BaseDeck {
	name := setNameOfDeck()
	return &BaseDeck{
		Id:   0,
		Name: name,
	}
}

func getBackOfCard() string {
	validate := func(input string) error {
		if len(input) == 0 {
			return errors.New("back of card cannot be empty")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Back of card",
		Validate: validate,
		Default:  "",
	}

	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
	}
	return result
}

func getFrontOfCard() string {
	validate := func(input string) error {
		if len(input) == 0 {
			return errors.New("front of card cannot be empty")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Front of card",
		Validate: validate,
		Default:  "",
	}

	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
	}
	return result
}

func setNameOfDeck() string {
	validate := func(input string) error {
		if len(input) == 0 {
			return errors.New("name of deck cannot be empty")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Deck name",
		Validate: validate,
		Default:  "",
	}

	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
	}
	return result
}

func (db *DB) getCardsToReview(deckId int) *ReviewDeck {
	stmt := "SELECT * FROM Cards WHERE datetime(ReviewDate) <= datetime('now') AND DeckId = ? ORDER BY ReviewDate"
	rows, err := db.db.Query(stmt, deckId)
	if err != nil {
		log.Fatal("Error querying for cards", err)
	}

	defer rows.Close()

	reviewDeck := ReviewDeck{
		Cards: []BaseCard{},
	}
	for rows.Next() {
		i := BaseCard{}
		err = rows.Scan(&i.Id, &i.DeckId, &i.Front, &i.Back, &i.Interval, &i.EaseFactor, &i.Repetition, &i.ReviewDate)
		if err != nil {
			log.Printf("Error occurred whilst mapping cards Id: %v - error: %v", &i.Id, err)
		}
		reviewDeck.Cards = append(reviewDeck.Cards, i)
	}
	return &reviewDeck
}

func (d *ReviewDeck) review(db *DB) *ReviewDeck {
	if len(d.Cards) == 0 {
		clearConsole()
		fmt.Print("Review complete! 🎉 \n ")
		return nil
	} else {
		d = d.reviewCard(db)
		return d.review(db)
	}
}

func (d *ReviewDeck) updateReviewDeck(pop bool) *ReviewDeck {
	if len(d.Cards) <= 1 && pop {
		d.Cards = []BaseCard{}
		return d
	} else if pop {
		d.Cards = d.Cards[1:]
		return d
	}
	firstCard := d.Cards[0]
	d.Cards = append(d.Cards[1:], firstCard)
	return d
}

func (d *ReviewDeck) reviewCard(db *DB) *ReviewDeck {
	clearConsole()
	card := &d.Cards[0]
	qualityString := card.viewFrontAndBack()
	quality := parseInput(qualityString)
	pop := card.updateCard(quality, db)
	clearConsole()

	return d.updateReviewDeck(pop)
}

func clearConsole() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}

func (c *BaseCard) viewFrontAndBack() string {
	viewFront(c)
	viewBack(c)
	input := selectQuality()
	return input
}

func viewFront(card *BaseCard) {
	fmt.Println(card.Front)
}

func viewBack(card *BaseCard) {
	prompt := promptui.Prompt{
		Label:     "Press 'Enter' to show answer",
		IsConfirm: false,
	}

	_, err := prompt.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println(card.Back)
}

func selectOption(menu []string, label string) string {
	templates := &promptui.SelectTemplates{
		Active:   "▸ {{ . }}",
		Inactive: "  {{ . | faint }}",
		Selected: "✔ {{ . | green }}",
	}

	prompt := promptui.Select{
		Label:        label,
		Items:        menu,
		HideSelected: true,
		HideHelp:     true,
		Templates:    templates,
	}

	_, result, err := prompt.Run()

	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}

	return result
}

func selectQuality() string {
	possibleQuality := []string{"1", "2", "3", "4", "5"}
	validate := func(input string) error {
		if !slices.Contains(possibleQuality, strings.TrimSpace(input)) {
			return errors.New("score must be between 1 (lowest) - 5 (highest)")
		}
		return nil
	}
	prompt := promptui.Prompt{
		Label:    "Score",
		Validate: validate,
	}

	result, err := prompt.Run()

	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}

	return result
}

func (c *BaseCard) updateCard(quality float32, db *DB) bool {
	if quality > 3 {
		c.Repetition = c.Repetition + 1
		c.EaseFactor = calculateEaseFactor(c.EaseFactor, quality)
		c.Interval = calculateInterval(c.Repetition, c.Interval, c.EaseFactor)
		c.ReviewDate = truncateToDay(c.ReviewDate.AddDate(0, 0, c.Interval))
		_, err := db.db.Exec("UPDATE Cards SET Repetition = ?, EaseFactor = ?, Interval = ?, ReviewDate = ? WHERE Id = ?;", c.Repetition, c.EaseFactor, c.Interval, c.ReviewDate, c.Id)
		if err != nil {
			fmt.Printf("Failed to update card Id: %v with error: %v", c.Id, err)
		}
		return true
	}

	c.Repetition = 0
	c.EaseFactor = calculateEaseFactor(c.EaseFactor, quality)
	c.Interval = calculateInterval(c.Repetition, c.Interval, c.EaseFactor)
	_, err := db.db.Exec("UPDATE Cards SET Repetition = ?, EaseFactor = ?, Interval = ? WHERE Id = ?;", c.Repetition, c.EaseFactor, c.Interval, c.Id)
	if err != nil {
		fmt.Printf("Failed to update card Id: %v with error: %v", c.Id, err)
	}

	return false
}

func parseInput(input string) float32 {
	input = strings.TrimSpace(input)
	quality64, err := strconv.ParseFloat(input, 32)
	if err != nil {
		log.Fatal(err)
	}
	quality := float32(quality64)
	return quality
}

func calculateEaseFactor(ef float32, quality float32) float32 {
	updatedEaseFactor := (ef) + (0.1 - (5-quality)*(0.8+(5-quality)*0.02))
	if updatedEaseFactor < 1.3 {
		return float32(1.3)
	}
	return updatedEaseFactor
}

func calculateInterval(repetition int, previousInterval int, ef float32) int {
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
