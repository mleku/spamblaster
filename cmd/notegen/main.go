package main

import "fmt"

func main() {
	fmt.Println("num sources", len(quotes))
	var total int
	for i := range quotes {
		n := len(quotes[i].Paragraphs)
		fmt.Println("source", quotes[i].Source, "num quotes", n)
		total += n
	}
	fmt.Println("total quotes", total)
}
