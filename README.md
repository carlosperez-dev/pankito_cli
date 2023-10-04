# Pankit cli  
I have been a user of [Anki](https://github.com/ankitects/anki) for the last 3 years and became curious about the inner workings of the SuperMemo algorithm it uses under the hood as a means to reset the [forgetting curve](https://www.growthengineering.co.uk/what-is-the-forgetting-curve/). Thus I decided to create a CLI application that implements said algorithm and persists the data to an SQLite database. 

## Why Go?
After watching multiple videos that speak highly of the language for its readability and usability, I decided to give it a Go (pun intended) and a CLI application is always a great first project to get comfortable with syntax and the standard libraries. Moreover, I wanted to have a bit of fun with a language that's different from the one I use at work (C#).

## ⚠️ Tests (under construction)
To run tests execute `go test` or `go test -v` for a verbose output.  

To get the test coverage report file run `go test -coverprofile=coverage.out`.  

To open the test coverage report in the browser run `go tool cover -html=coverage.out`. This will highlight the parts of the code that are covered by the test suite. 
