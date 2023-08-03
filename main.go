package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

func main() {
	StartDeckReview(CardsToReview())
}

type PankitoBaseCard struct {
	Content    string
	Interval   int
	EaseFactor float32
	Repetition int
}

func CardsToReview() []PankitoBaseCard {
	deck := make([]PankitoBaseCard, 0)
	for i := 0; i < 10; i++ {
		deck = append(deck, PankitoBaseCard{
			Content:    fmt.Sprintf("Test %v", i),
			Interval:   0,
			EaseFactor: 2.5,
			Repetition: 0,
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
	fmt.Printf(">%s \n", card.Content)
	fmt.Println(">Type the difficulty")
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
