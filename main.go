package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := newDb()
	if err != nil {
		log.Fatal("Error when starting db", err)
	}
	// AddCardsToReview(db)
	// GetCardsToReview(db)
	StartDeckReview(GetCardsToReview(db), db)
}

type DB struct {
	db *sql.DB
}

func newDb() (*DB, error) {
	file := "pankito.db"
	create := "CREATE TABLE IF NOT EXISTS [Cards] ( Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, Front TEXT NOT NULL, Back TEXT NOT NULL, Interval INTEGER NOT NULL, EaseFactor DECIMAL(10,8) NOT NULL, Repetition INTEGER NOT NULL, ReviewDate DATETIME NOT NULL);"

	db, err := sql.Open("sqlite3", file)
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

func AddNewCard(db *DB, card PankitoBaseCard) {
	stmt := "INSERT INTO Cards(Front, Back, Interval, EaseFactor, Repetition, ReviewDate) VALUES (?, ?, ?, ?, ?, ?)"
	if _, err := db.db.Exec(stmt, card.Front, card.Back, 0, 2.5, 0, time.Now()); err != nil {
		log.Fatal("Failed to execute INSERT", err)
	}
	log.Printf("Card added")
}

type PankitoBaseCard struct {
	Id         int
	Front      string
	Back       string
	Interval   int
	EaseFactor float32
	Repetition int
	ReviewDate time.Time
}

func AddCardsToReview(db *DB) []PankitoBaseCard {
	deck := make([]PankitoBaseCard, 0)
	for i := 0; i < 10; i++ {
		deck = append(deck, PankitoBaseCard{
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

func GetCardsToReview(db *DB) []PankitoBaseCard {
	stmt := "SELECT * FROM Cards WHERE datetime(ReviewDate) <= datetime('now') ORDER BY ReviewDate"
	rows, err := db.db.Query(stmt)
	if err != nil {
		log.Fatal("Error querying for cards", err)
	}

	defer rows.Close()

	data := []PankitoBaseCard{}
	for rows.Next() {
		i := PankitoBaseCard{}
		err = rows.Scan(&i.Id, &i.Front, &i.Back, &i.Interval, &i.EaseFactor, &i.Repetition, &i.ReviewDate)
		if err != nil {
			log.Fatal("Error occurred whilst mapping cards Id: %v", &i.Id, err)
		}
		data = append(data, i)
		fmt.Println(i)
	}
	return data

}

func StartDeckReview(deck []PankitoBaseCard, db *DB) []PankitoBaseCard {
	if len(deck) == 0 {
		return deck
	} else {
		updatedDeck := ReviewCard(deck, db)
		return StartDeckReview(updatedDeck, db)
	}
}

func UpdateReviewDeck(reviewDeck []PankitoBaseCard, pop bool) []PankitoBaseCard {
	updatedReviewDeck := make([]PankitoBaseCard, 0)
	if pop {
		return append(updatedReviewDeck, reviewDeck[1:]...)
	} else {
		updatedReviewDeck = append(updatedReviewDeck, reviewDeck[1:]...)
		return append(updatedReviewDeck, reviewDeck[0])
	}
}

func ReviewCard(reviewDeck []PankitoBaseCard, db *DB) []PankitoBaseCard {
	card := &reviewDeck[0]
	var pop bool

	//get quality
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n > %s \n\n", card.Front)

	fmt.Println("--> Press 'Enter' to show answer")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	fmt.Printf("> %s \n", card.Back)
	fmt.Println("\n --> Quality of answer (0 - 5)?")

	input, err := reader.ReadString('\n')

	if err != nil {
		log.Fatal(err)
	}
	input = strings.TrimSpace(input)
	quality64, err := strconv.ParseFloat(input, 32)
	if err != nil {
		log.Fatal(err)
	}
	quality := float32(quality64)
	if quality <= 3 {
		card.Repetition = 0
		card.EaseFactor = CalculateEaseFactor(card.EaseFactor, quality)
		card.Interval = CalculateInterval(card.Repetition, card.Interval, card.EaseFactor)

		//persist card changes
		_, err := db.db.Exec("UPDATE Cards SET Repetition = ?, EaseFactor = ?, Interval = ? WHERE Id = ?;", card.Repetition, card.EaseFactor, card.Interval, card.Id)
		if err != nil {
			fmt.Printf("Failed to update card Id: %v with error: %v", card.Id, err)
		}

		pop = false
		return UpdateReviewDeck(reviewDeck, pop)
	} else {
		card.Repetition = card.Repetition + 1
		card.EaseFactor = CalculateEaseFactor(card.EaseFactor, quality)
		card.Interval = CalculateInterval(card.Repetition, card.Interval, card.EaseFactor)
		card.ReviewDate = truncateToDay(card.ReviewDate.AddDate(0, 0, card.Interval))

		//persist card changes
		_, err := db.db.Exec("UPDATE Cards SET Repetition = ?, EaseFactor = ?, Interval = ?, ReviewDate = ? WHERE Id = ?;", card.Repetition, card.EaseFactor, card.Interval, card.ReviewDate, card.Id)
		if err != nil {
			fmt.Printf("Failed to update card Id: %v with error: %v", card.Id, err)
		}

		pop = true
		return UpdateReviewDeck(reviewDeck, pop)
	}
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
