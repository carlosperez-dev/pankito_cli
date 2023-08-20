package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

func SetUp() *sql.DB {
	db := CreateDatabase()
	return db
}

func CreateDatabase() *sql.DB {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal("Failed to create DB")
	}
	create := "CREATE TABLE IF NOT EXISTS [Cards] ( Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, DeckId INTEGER NOT NULL,Front TEXT NOT NULL, Back TEXT NOT NULL, Interval INTEGER NOT NULL, EaseFactor DECIMAL(10,8) NOT NULL, Repetition INTEGER NOT NULL, ReviewDate DATETIME NOT NULL, FOREIGN KEY(DeckId) REFERENCES Decks(Id)); CREATE TABLE IF NOT EXISTS [Decks] ( Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, Name TEXT NOT NULL);"

	if _, err := db.Exec(create); err != nil {
		log.Fatal("Failed to create table: Cards")
	}
	return db
}
func TestGivenNoDbExists_ThenShouldCreateOne(t *testing.T) {
	var data = struct {
		dbName string
		err    error
	}{
		dbName: "pankito",
		err:    nil,
	}

	// Given a non-whitespace name
	db, err := newDb(data.dbName)
	if err != nil {
		t.Errorf("error creating db, %v", err)
	}

	// A database should be created
	err = db.db.Ping()
	if err != nil {
		t.Errorf("error pinging db, %v", err)
	}

	tearDown(data.dbName)

}

func TestGivenEmptyDbName_ShouldRaiseError(t *testing.T) {
	expectedError := "cannot instantiate a db with empty/whitespace name"

	// Given an empty name
	name := ""

	// When trying to create a database
	_, err := newDb(name)

	// The correct error should be raised
	if err != nil {
		if err.Error() != expectedError {
			t.Errorf("incorrect error raised, got: %v, expected :%v", err, expectedError)
		}
	} else {
		tearDown(name)
		t.Error("created unnamed db")
	}
}

func TestGivenACardIsDueToday_ShouldReturnCardDue(t *testing.T) {
	db := SetUp()
	defer db.Close()
	AddDeck(db)
	AddCardsDueToday(db, 1)
	AddCardsDueInFuture(db, 1)
	dbStruct := DB{
		db,
	}
	cards := GetCardsToReview(&dbStruct, 1)
	if len(cards) > 1 {
		t.Error("Returned wrong number of cards due today")
	}
	for _, card := range cards {
		year, month, day := card.ReviewDate.Date()
		expectedYear, expectedMonth, expectedDay := time.Now().Date()
		if year != expectedYear || month != expectedMonth || day != expectedDay {
			t.Errorf("Cards not due today were returned date returned: %v-%v-%v; expected: %v-%v-%v,", year, month, day, expectedYear, expectedMonth, expectedDay)
		}
	}
}

func TestGivenACardsDueInFuture_ShouldReturnNoCards(t *testing.T) {
	db := SetUp()
	defer db.Close()
	AddDeck(db)
	AddCardsDueInFuture(db, 5)
	dbStruct := DB{
		db,
	}
	cards := GetCardsToReview(&dbStruct, 1)
	if len(cards) > 0 {
		t.Errorf("Returned cards due in future, %v", cards)
	}
}

func TestGivenACardIsDueInPast_ShouldReturnCard(t *testing.T) {
	db := SetUp()
	defer db.Close()
	AddDeck(db)
	AddCardsDueInPast(db, 1)
	AddCardsDueInFuture(db, 1)
	dbStruct := DB{
		db,
	}
	cards := GetCardsToReview(&dbStruct, 1)
	if len(cards) > 1 {
		t.Errorf("Returned wrong number of cards due in past, %v", cards)
	}
	for _, card := range cards {
		year, month, day := card.ReviewDate.Date()
		currentYear, currentMonth, currentDay := time.Now().Date()
		if year > currentYear || month > currentMonth || day > currentDay {
			t.Errorf("Cards in future were returned: %v-%v-%v; today: %v-%v-%v,", year, month, day, currentYear, currentMonth, currentDay)
		}
	}
}

func PastCard(id int) BaseCard {
	return BaseCard{
		Front:      fmt.Sprintf("Past Front %v", id),
		Back:       fmt.Sprintf("Past Back %v", id),
		Interval:   0,
		EaseFactor: 2.5,
		Repetition: 0,
		ReviewDate: truncateToDay(time.Now().AddDate(0, 0, -1)),
	}
}

func CurrentCard(id int) BaseCard {
	return BaseCard{
		Id:         id,
		Front:      fmt.Sprintf("Current Front %v", id),
		Back:       fmt.Sprintf("Current Back %v", id),
		Interval:   0,
		EaseFactor: 2.5,
		Repetition: 0,
		ReviewDate: truncateToDay(time.Now()),
	}
}

func FutureCard(id int) BaseCard {
	return BaseCard{
		Front:      fmt.Sprintf("Future Front %v", id),
		Back:       fmt.Sprintf("Future Back %v", id),
		Interval:   0,
		EaseFactor: 2.5,
		Repetition: 0,
		ReviewDate: truncateToDay(time.Now().AddDate(0, 0, 1)),
	}
}

func AddCardsDueToday(db *sql.DB, quantity int) {
	deck := make([]BaseCard, 0)
	for i := 1; i < quantity+1; i++ {
		deck = append(deck, CurrentCard(i))
		stmt := "INSERT INTO Cards(DeckId, Front, Back, Interval, EaseFactor, Repetition, ReviewDate) VALUES (?, ?, ?, ?, ?, ?, ?)"
		if _, err := db.Exec(stmt, 1, deck[len(deck)-1].Front, deck[len(deck)-1].Back, 0, 2.5, 0, deck[len(deck)-1].ReviewDate); err != nil {
			log.Fatal("Failed to execute INSERT", err)
		}
	}
}

func AddCardsDueInPast(db *sql.DB, quantity int) {
	deck := make([]BaseCard, 0)
	for i := 1; i < quantity+1; i++ {
		deck = append(deck, PastCard(i))
		stmt := "INSERT INTO Cards(DeckId, Front, Back, Interval, EaseFactor, Repetition, ReviewDate) VALUES (?, ?, ?, ?, ?, ?, ?)"
		if _, err := db.Exec(stmt, 1, deck[len(deck)-1].Front, deck[len(deck)-1].Back, 0, 2.5, 0, deck[len(deck)-1].ReviewDate); err != nil {
			log.Fatal("Failed to execute INSERT", err)
		}
	}
}
func AddCardsDueInFuture(db *sql.DB, quantity int) {
	deck := make([]BaseCard, 0)
	for i := 1; i < quantity+1; i++ {
		deck = append(deck, FutureCard(i))
		stmt := "INSERT INTO Cards(DeckId, Front, Back, Interval, EaseFactor, Repetition, ReviewDate) VALUES (?, ?, ?, ?, ?, ?, ?)"
		if _, err := db.Exec(stmt, 1, deck[len(deck)-1].Front, deck[len(deck)-1].Back, 0, 2.5, 0, deck[len(deck)-1].ReviewDate); err != nil {
			log.Fatal("Failed to execute INSERT", err)
		}
	}
}

func AddDeck(db *sql.DB) {
	stmt := "INSERT INTO Decks(Name) VALUES (?)"
	if _, err := db.Exec(stmt, "Test Deck"); err != nil {
		log.Fatal("Failed to execute INSERT", err)
	}
}

func tearDown(name string) {
	err := os.Remove(name + ".db")
	if err != nil {
		log.Printf("Error deleting db, %v", err)
	}
}
