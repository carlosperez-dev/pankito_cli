package main

import (
	"log"
	"os"
	"testing"
)

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

func tearDown(name string) {
	err := os.Remove(name + ".db")
	if err != nil {
		log.Printf("Error deleting db, %v", err)
	}
}
