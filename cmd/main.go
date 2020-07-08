package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/drkchiloll/ciscowlc-aireos/aireos"
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
	wlc := aireos.New(host, user, pass)
	err := wlc.Client.Connect(3)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer wlc.Client.Close()

	aps := wlc.GetApDb()
	fmt.Println(len(aps))
	j, _ := json.Marshal(aps)
	fmt.Println(string(j))
	// oldApName := "AP00BE.759E.3C24"
	// config ap name <new-name> <old-name>
	// apName := "ap01.core-dev.test.tx"
	// wlc.SetApName(oldApName, apName)
	// wlc.SetApName(apName, oldApName)
	// time.Sleep(500 * time.Millisecond)
	// apGroup := "Austin-Corp-Default"
	// apGroup := "default-group"
	// wlc.SetApGroup(apGroup, oldApName)
	// config ap group-name <group-name> <ap-name>
	// fmt.Println(len(aps))
	// apCdp := wlc.GetApCdp("APB026.80DF.45E8")
	// apIntf := wlc.GetApEthStat("APB026.80DF.45E8")
	// fmt.Println(apCdp)
	// fmt.Println(apIntf)
}
