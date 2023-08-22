# BRC Food Events Guide

This script produces a printable list of food events in BRC. Print pages double sided, 3-hole punch them, and put it in a binder in your camp kitchen. Great for finding out what's open when your hungry, but also excellent for planning a food-oriented city-wide exploration day.

# Usage

1. Get [Go](https://go.dev/) running
2. Get an API key for [api.burningman.org](https://api.burningman.org/)
3. Fetch the Art, Camps, and Events data from the API, and place them in [brc_api_\[YEAR\]/](/brc_api_2022/)
4. JSON unmarshalling in Go is annoying, and I was lazy so, so manually convert any integer names in the data set to strings (looking at you, Camps 3907 and 88)
5. Run `go run brc-food-guide.go`
6. Your handy dandy food guide will be waiting for you at [out/food-guide.pdf](/out/food-guide.pdf)
