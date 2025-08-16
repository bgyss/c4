package db

import (
	"math/rand/v2"
	"strconv"
	"testing"
)

func TestShuffle(t *testing.T) {
	r := rand.New(rand.NewPCG(42, 0))
	for j := 1; j < 1000; j++ {
		var list []string
		for i := 0; i < j; i++ {
			list = append(list, strconv.Itoa(i))
		}
		shuffleWithRand(list, r)
		count := 0
		for i := 0; i < j; i++ {
			if list[i] == strconv.Itoa(i) {
				count++
			}
		}
		if j > 12 && (float32(count)/float32(j)) > 0.31 {
			t.Errorf("shuffle ratio for %d: %.2f", j, (float32(count) / float32(j)))
		}
		if j == 10 {
			t.Logf("shuffle: %v", list)
		}
	}

}

func shuffleWithRand(list []string, r *rand.Rand) {
	l := len(list)
	for j, i := 0, 0; i < l; i++ {
		j = r.IntN(l)
		if j != i {
			list[i], list[j] = list[j], list[i]
		}
	}
}
