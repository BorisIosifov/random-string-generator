package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"time"
)

const (
	// All possible characters which could match '.' character or inverted character class: [^...]
	possibleCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890',./|?><`~!#$%^&*()-_=+{}[]:;\"\\"
	// Max number of characters (or strings generated for subpatterns) for quantifiers * and +
	maxCharactersInStarOrPlus = 10
)

var (
	regexp = flag.String("re", "", "regular expression")
	number = flag.Int("n", 10, "numer of generated strings")
	// Index of current subpattern
	currentIndex = 0
	// Matching indexes and generated strings for subpatterns
	subpatterns map[int]string
)

// All possible parts of regexp
type regExp interface {
	// The method generating a string for a part
	generate() string
	// The setter for min and max numbers
	setMinMaxNumber(int, int)
}

// Embeded struct with min and max numbers
// Using pointers to be able to modify numbers on the spot without assignment of object
type minMaxNumbers struct {
	minNumber *int
	maxNumber *int
}

// Constructor
func newMinMaxNumbers() minMaxNumbers {
	var (
		min = 1
		max = 1
	)
	return minMaxNumbers{&min, &max}
}

// Setter for min and max numbers
func (mmn minMaxNumbers) setMinMaxNumber(min int, max int) {
	*mmn.minNumber = min
	*mmn.maxNumber = max
}

// List of regexp parts
type regExpList struct {
	minMaxNumbers
	// List of parts
	list []regExp
	// True if this part is one of alternative branches of a bigger part
	isItAPartOfAlternative bool
	// Pointer to a part which contains current part as alternative branch
	parentAlternative *regExpAlternative
}

func (rel regExpList) generate() string {
	var index int
	if !rel.isItAPartOfAlternative {
		index = currentIndex
		currentIndex += 1
	}
	size := *rel.minNumber + rand.Intn(*rel.maxNumber-*rel.minNumber+1)
	var result string
	for i := 0; i < size; i++ {
		for _, re := range rel.list {
			result += re.generate()
		}
	}
	if !rel.isItAPartOfAlternative {
		subpatterns[index] = result
	}
	return result
}

// A part with alternative branches
type regExpAlternative struct {
	minMaxNumbers
	// List of alternative branches
	list []regExp
}

func (rea regExpAlternative) generate() string {
	index := currentIndex
	currentIndex += 1
	size := *rea.minNumber + rand.Intn(*rea.maxNumber-*rea.minNumber+1)
	var result string
	for i := 0; i < size; i++ {
		result += rea.list[rand.Intn(len(rea.list))].generate()
	}
	subpatterns[index] = result
	return result
}

// Characters class
type regExpClass struct {
	minMaxNumbers
	// List of possible characters
	characters []rune
}

func (rec regExpClass) generate() string {
	size := *rec.minNumber + rand.Intn(*rec.maxNumber-*rec.minNumber+1)
	var result string
	for i := 0; i < size; i++ {
		result += string(rec.characters[rand.Intn(len(rec.characters))])
	}
	return result
}

// Back reference
type regExpBackReference struct {
	minMaxNumbers
	// Index of subpattern
	index int
}

func (reb regExpBackReference) generate() string {
	str, ok := subpatterns[reb.index]
	if ok {
		size := *reb.minNumber + rand.Intn(*reb.maxNumber-*reb.minNumber+1)
		var result string
		for i := 0; i < size; i++ {
			result += str
		}
		return result
	} else {
		return ""
	}
}

// Stack of subpatterns. It needed during the analysis of a regexp
type Stack struct {
	arr []regExpList
}

func (stack Stack) len() int {
	return len(stack.arr)
}

func (stack *Stack) push(rel regExpList) {
	stack.arr = append(stack.arr, rel)
}

func (stack *Stack) pop() regExpList {
	pop := stack.arr[stack.len()-1]
	stack.arr = stack.arr[:(stack.len() - 1)]
	return pop
}

func main() {
	flag.Parse()

	// Regexp in runes (utf8 support)
	r := []rune(*regexp)
	// All possible characters in runes
	possibleCharactersRunes := []rune(possibleCharacters)
	sort.Slice(possibleCharactersRunes, func(i, j int) bool {
		return possibleCharactersRunes[i] < possibleCharactersRunes[j]
	})
	// Current regExpList (for the first time it is the top level of regexp parts)
	rel := regExpList{
		minMaxNumbers:          newMinMaxNumbers(),
		isItAPartOfAlternative: false,
	}
	// Stack of subpatterns
	stack := Stack{}
	rand.Seed(time.Now().Unix())

	for i := 0; i < len(r); i++ {
		switch r[i] {
		case '(':
			// Starting a new subpatern, pushing the old one to the stack
			stack.push(rel)
			rel = regExpList{
				minMaxNumbers:          newMinMaxNumbers(),
				isItAPartOfAlternative: false,
			}

		case ')':
			// Closing a subpattern. Appending it to the parts list of an above level.
			if stack.len() == 0 {
				log.Fatalf("Found ')' without '('")
			}

			relFinished := rel
			rel = stack.pop()
			if relFinished.isItAPartOfAlternative {
				relFinished.parentAlternative.list = append(relFinished.parentAlternative.list, relFinished)
				rel.list = append(rel.list, relFinished.parentAlternative)
			} else {
				rel.list = append(rel.list, relFinished)
			}

		case '|':
			// Starting an alternative branch
			// If it is a second one, we didn't know about the first one that it should be a branch. Making it as a branch.
			if !rel.isItAPartOfAlternative {
				rea := regExpAlternative{minMaxNumbers: newMinMaxNumbers()}
				rel.isItAPartOfAlternative = true
				rel.parentAlternative = &rea
			}

			rel.parentAlternative.list = append(rel.parentAlternative.list, rel)
			rel = regExpList{
				minMaxNumbers:          newMinMaxNumbers(),
				isItAPartOfAlternative: true,
				parentAlternative:      rel.parentAlternative,
			}

		case '[':
			// Starting a class of characters
			if i+1 == len(r) {
				log.Fatalf("missing ']' character")
			}

			// If the first character is ^, we will invert the class (do a set subtraction)
			needInvert := false
			if r[i+1] == '^' {
				i++
				needInvert = true
			}

			var chars []rune
			for i = i + 1; i < len(r) && r[i] != ']'; i++ {
				// Processing a range
				if i+2 < len(r) && r[i+1] == '-' && r[i+2] != ']' {
					for char := r[i]; char <= r[i+2]; char++ {
						chars = append(chars, char)
					}
					i += 2
				} else {
					chars = append(chars, r[i])
				}
			}
			if i >= len(r) {
				log.Fatalf("missing ']' character")
			}
			if len(chars) == 0 {
				log.Fatalf("empty sequence between [ and ]")
			}

			var rec regExpClass
			if needInvert {
				// Invert the class (doing a set subtraction)
				var invertChars []rune
				sort.Slice(chars, func(i, j int) bool { return chars[i] < chars[j] })
				var k = 0
				for j := 0; j < len(possibleCharactersRunes); j++ {
					if k < len(chars) && possibleCharactersRunes[j] >= chars[k] {
						k++
					} else {
						invertChars = append(invertChars, possibleCharactersRunes[j])
					}
				}
				rec = regExpClass{
					minMaxNumbers: newMinMaxNumbers(),
					characters:    invertChars,
				}
			} else {
				rec = regExpClass{
					minMaxNumbers: newMinMaxNumbers(),
					characters:    chars,
				}
			}
			rel.list = append(rel.list, rec)

		case '{':
			// Starting min/max quantifier
			var minmax = [2]int{0, 0}
			var index = 0
			for i = i + 1; i < len(r) && r[i] != '}'; i++ {
				if r[i] == ',' {
					index++
					if index > 1 {
						log.Fatalf("error in character %d (%s), extra comma", i, string(r[i]))
					}
					continue
				}
				digit, err := strconv.Atoi(string(r[i]))
				if err != nil {
					log.Fatalf("error in character %d (%s) a digit was expected", i, string(r[i]))
				}
				minmax[index] = 10*minmax[index] + digit
			}

			if index == 0 {
				minmax[1] = minmax[0]
			} else if minmax[1] == 0 {
				minmax[1] = minmax[0] + maxCharactersInStarOrPlus
			}
			if minmax[0] > minmax[1] {
				log.Fatalf("min quantifier bigger than max")
			}
			var ind = len(rel.list) - 1
			rel.list[ind].setMinMaxNumber(minmax[0], minmax[1])

		case '?':
			var ind = len(rel.list) - 1
			rel.list[ind].setMinMaxNumber(0, 1)

		case '*':
			var ind = len(rel.list) - 1
			rel.list[ind].setMinMaxNumber(0, maxCharactersInStarOrPlus)

		case '+':
			var ind = len(rel.list) - 1
			rel.list[ind].setMinMaxNumber(1, maxCharactersInStarOrPlus)

		case '.':
			chars := possibleCharactersRunes
			rec := regExpClass{
				minMaxNumbers: newMinMaxNumbers(),
				characters:    chars,
			}
			rel.list = append(rel.list, rec)

		case '\\':
			// Backslash escaping of a character
			// It can be back reference or just a special character which should not be recognized as a special
			i++
			digit, err := strconv.Atoi(string(r[i]))
			if err == nil {
				reb := regExpBackReference{
					minMaxNumbers: newMinMaxNumbers(),
					index:         digit,
				}
				rel.list = append(rel.list, reb)
			} else {
				var chars []rune
				chars = append(chars, r[i])
				rec := regExpClass{
					minMaxNumbers: newMinMaxNumbers(),
					characters:    chars,
				}
				rel.list = append(rel.list, rec)
			}

		default:
			var chars []rune
			chars = append(chars, r[i])
			rec := regExpClass{
				minMaxNumbers: newMinMaxNumbers(),
				characters:    chars,
			}
			rel.list = append(rel.list, rec)
		}
	}

	if rel.isItAPartOfAlternative {
		relFinished := rel
		relFinished.parentAlternative.list = append(relFinished.parentAlternative.list, relFinished)
		rel = regExpList{
			minMaxNumbers:          newMinMaxNumbers(),
			isItAPartOfAlternative: false,
		}
		rel.list = append(rel.list, relFinished.parentAlternative)
	}

	if stack.len() > 0 {
		log.Fatalf("missing ')' character")
	}

	for i := 0; i < *number; i++ {
		currentIndex = 0
		subpatterns = make(map[int]string)
		fmt.Println(rel.generate())
	}
}
