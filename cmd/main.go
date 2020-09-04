package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ApogeeNetworking/ciscowlc-aireos/aireos"
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
	fmt.Println(aps)

	// apCdp := wlc.GetApCdp("APB026.80DF.45E8")
	// fmt.Println(apCdp)
	// apIntf := wlc.GetApEthStat("ap01.mnsu.craw.dining.mn")
	// 	apIntf := wlc.GetApEthStat("302686-CrawA-403")
	// 	fmt.Println(apIntf)
}
