package main

import (
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/hash"
)

func main() {
	h, err := hash.HashPassword("History2@@5")
	if err != nil {
		panic(err)
	}
	fmt.Println(h)
}
