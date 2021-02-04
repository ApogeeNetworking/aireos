package main

import (
	"log"
	"os"

	"github.com/ApogeeNetworking/aireos"
	"github.com/subosito/gotenv"
)

var host, user, pass string

func init() {
	gotenv.Load()
	host = os.Getenv("SSH_HOST")
	user = os.Getenv("SSH_USER")
	pass = os.Getenv("SSH_PW")
}

func main() {
	wlc := aireos.New(host, user, pass, "old")
	err := wlc.Client.Connect(3)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer wlc.Logout()
}
