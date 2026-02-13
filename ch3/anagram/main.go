package main

import "fmt"

func main() {
	var word1, word2 string
	fmt.Print("Enter the first word: ")
	_, _ = fmt.Scan(&word1)
	fmt.Print("Enter the second word: ")
	_, _ = fmt.Scan(&word2)

	if isAnagram(word1, word2) {
		fmt.Println("The words are anagrams.")
	} else {
		fmt.Println("The words are not anagrams.")
	}
}

func isAnagram(s, t string) bool {
	if len(s) != len(t) {
		return false
	}

	var sCount [26]int
	var tCount [26]int

	for i := 0; i < len(s); i++ {
		sCount[s[i]-'a']++
		tCount[t[i]-'a']++
	}

	return sCount == tCount
}
