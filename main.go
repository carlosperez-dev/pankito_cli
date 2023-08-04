package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	StartDeckReview(CardsToReview())
}

type PankitoBaseCard struct {
	Front      string
	Back       string
	Interval   int
	EaseFactor float32
	Repetition int
	ReviewDate time.Time
}

func CardsToReview() []PankitoBaseCard {
	deck := make([]PankitoBaseCard, 0)
	for i := 0; i < 10; i++ {
		deck = append(deck, PankitoBaseCard{
			Front:      fmt.Sprintf("Test %v?", i),
			Back:       fmt.Sprintf("Answer %v", i),
			Interval:   6,
			EaseFactor: 2.5,
			Repetition: 2,
			ReviewDate: time.Now(),
		})
	}
	return deck
}

func StartDeckReview(params ...[]PankitoBaseCard) []PankitoBaseCard {
	//get reviewdeck
	var deck []PankitoBaseCard
	if len(params) == 0 {
		deck = CardsToReview()
	} else {
		deck = params[0]
	}

	if len(deck) == 0 {
		return deck
	} else {
		updatedDeck := ReviewNextCard(deck)
		return StartDeckReview(updatedDeck)
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

func ReviewNextCard(reviewDeck []PankitoBaseCard) []PankitoBaseCard {
	card := reviewDeck[0]
	var pop bool

	//get quality
	reader := bufio.NewReader(os.Stdin)
	year, month, day := card.ReviewDate.Date()
	fmt.Printf("> %s \n> Current Date: %v/%v/%v \n", card.Front, day, int(month), year)

	fmt.Println("> Press 'Enter' to show answer...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	fmt.Printf("> %s \n", card.Back)
	fmt.Println("> Quality of answer (0 - 5)")

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
		pop = false
		return UpdateReviewDeck(reviewDeck, pop)
	} else {
		card.Repetition = card.Repetition + 1
		card.EaseFactor = CalculateEaseFactor(card.EaseFactor, quality)
		card.Interval = CalculateInterval(card.Repetition, card.Interval, card.EaseFactor)
		card.ReviewDate = card.ReviewDate.AddDate(0, 0, card.Interval)
		year, month, day := card.ReviewDate.Date()
		fmt.Printf("> Next review in %v days \n> New Revivew date: %v/%v/%v \n \n", card.Interval, day, int(month), year)
		//persist card changes
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
