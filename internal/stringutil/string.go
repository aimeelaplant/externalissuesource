package stringutil

import (
	"sync"
	"math/rand"
	"time"
	"strings"
)

const randCharMap = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890"

var once sync.Once

// Generate the random seed at program start.
func init() {
	once.Do(func() {
		rand.Seed(time.Now().UnixNano())
	})
}

func TrimStrings(str string, toTrim []string) string {
	newString := strings.TrimSpace(str)
	for _, trim := range toTrim {
		newString = strings.Replace(newString, trim, "", -1)
	}
	return strings.TrimSpace(newString)
}

// Generates a random string of the length `n` with characters A-Z, a-z, and 0-9.
func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = randCharMap[rand.Intn(len(randCharMap))]
	}
	rand.Int63()
	return string(b)
}