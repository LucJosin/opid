// Command basic demonstrates UUIDv7 encoding and decoding with opid.
package main

import (
	"fmt"
	"log"
	"uuid"

	"github.com/LucJosin/opid"
)

func main() {
	codec, err := opid.NewCodec("usr")
	if err != nil {
		log.Fatal(err)
	}

	id := uuid.NewV7()
	publicID, err := codec.Encode(id)
	if err != nil {
		log.Fatal(err)
	}

	decoded, err := codec.Decode(publicID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("UUID:", id)
	fmt.Println("Public ID:", publicID.String())
	fmt.Println("Same UUID:", decoded == id)
}
