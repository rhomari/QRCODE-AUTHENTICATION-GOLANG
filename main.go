package main

import (
	"QRCODE-AUTH/serverpkg"
	"fmt"
)

func main() {
	serverpkg.StarServing(":8080")

}

func messageHandler(message []byte) {
	fmt.Println(string(message))
}
