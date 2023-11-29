package main

import (
	"fmt"
	"math/rand"
)

func main() {
	source := rand.Intn(len(quotes))
	quote := rand.Intn(len(quotes[source].Paragraphs))
	fmt.Printf("\"%s\"\n- %s\n", quotes[source].Paragraphs[quote],
		quotes[source].Source)
}
