package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

func TestGivenNoDbExists_ThenShouldCreateOne(t *testing.T) {
	var data = struct {
		dbName string
		err    error
	}{
		dbName: "test",
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
		reviewdate := card.ReviewDate
		currenTime := time.Now()
		diffence := currenTime.Sub(reviewdate)
		if diffence.Minutes() < 0 {
			t.Errorf("Cards in future were returned: %v; today: %v,", reviewdate, currenTime)
		}
	}
}

func TestGivenCardIsReviewed_ShouldUpdateCard(t *testing.T) {
	tests := []struct {
		name               string
		initialCard        BaseCard
		quality            float32
		expectedInterval   int
		expectedEaseFactor float32
		expectedRepetition float32
		expectedReviewDate time.Time
	}{
		{
			name:               "QualityGreaterThan3",
			initialCard:        CurrentCard(1),
			quality:            4,
			expectedInterval:   1,
			expectedEaseFactor: 1.78,
			expectedRepetition: 1,
			expectedReviewDate: time.Date(time.Now().Year(), time.Now().Month(), time.Now().AddDate(0, 0, 1).Day(), 0, 0, 0, 0, time.Local),
		},
		{
			name:               "QualityLessThanOrEqualTo3",
			initialCard:        CurrentCard(1),
			quality:            2,
			expectedInterval:   1,
			expectedEaseFactor: 1.3,
			expectedRepetition: 0,
			expectedReviewDate: CurrentCard(1).ReviewDate,
		},
		// Add more test cases here as needed.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &DB{
				db: SetUp(),
			}
			AddCardsDueToday(db.db, 1)
			card := GetCardsAdded(db.db)[0]

			result := UpdateCard(&card, tt.quality, db)

			if card.Interval != tt.expectedInterval {
				t.Errorf("Expected interval %v, got %v", tt.expectedInterval, card)
			}

			if card.EaseFactor != tt.expectedEaseFactor {
				t.Errorf("Expected ease factor %v, got %v", tt.expectedEaseFactor, card.EaseFactor)
			}

			if card.Repetition != int(tt.expectedRepetition) {
				t.Errorf("Expected repetition %v, got %v", tt.expectedRepetition, card.Repetition)
			}

			if card.ReviewDate.Local() != tt.expectedReviewDate {
				t.Errorf("Expected review date %v, got %v", tt.expectedReviewDate, card.ReviewDate)
			}
			if result != (tt.quality > 3) {
				t.Errorf("Expected result to be %v, got %v", tt.quality > 3, result)
			}
		})
	}
}

func GivenEfficiencyScoreAndQuality_ShouldCorrectlyCalculateEaseFactor(t *testing.T) {
	testCases := []struct {
		ef       float32
		quality  float32
		expected float32
	}{
		{1.0, 2.0, 1.3},  // Example 1
		{1.0, 5.0, 1.3},  // Example 2
		{5.0, 3.0, 3.42}, // Example 3
		{2.0, 5.0, 2.1},  // Example 4
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("EF=%.2f, quality=%.2f", tc.ef, tc.quality), func(t *testing.T) {
			result := CalculateEaseFactor(tc.ef, tc.quality)
			if !floatsAreEqual(result, tc.expected, 1e-5) {
				t.Errorf("Expected %f, but got %f", tc.expected, result)
			}
		})
	}
}

func GivenRepetitionPreviousIntervalAndEaseFactor_ShouldCalculateCorrrectInterval(t *testing.T) {
	testCases := []struct {
		repetition       int
		previousInterval int
		ef               float32
		expected         int
	}{
		{1, 10, 1.5, 1},
		{2, 10, 1.5, 6},
		{3, 10, 1.5, 15},
		{1, 5, 2.0, 1},
		{2, 5, 2.0, 6},
		{3, 5, 2.0, 10},
	}

	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf("repetition=%d, previousInterval=%d, ef=%.2f", tc.repetition, tc.previousInterval, tc.ef),
			func(t *testing.T) {
				result := CalculateInterval(tc.repetition, tc.previousInterval, tc.ef)

				if result != tc.expected {
					t.Errorf("Expected %d, but got %d", tc.expected, result)
				}
			},
		)
	}
}

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

func GetCardsAdded(db *sql.DB) []BaseCard {
	stmt := "SELECT * FROM Cards WHERE datetime(ReviewDate) <= datetime('now') AND DeckId = ? ORDER BY ReviewDate"
	rows, err := db.Query(stmt, 1)
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

func floatsAreEqual(a, b, tolerance float32) bool {
	return a >= b-tolerance && a <= b+tolerance
}
