package main

import (
	"fmt"
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
	wlc := aireos.New(host, user, pass)
	err := wlc.Client.Connect(3)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer wlc.Logout()
	ap, _ := wlc.GetAp("f0:b2:e5:c2:39:98")
	fmt.Println(ap)
	// apCdp := wlc.GetApCdp("ap01.it.corp-austin.tx")
	// fmt.Println(apCdp)
	// var wg1 sync.WaitGroup
	// wg1.Add(1)
	// go func() {
	// 	aps, _ := wlc.GetApDb()
	// 	for _, ap := range aps {
	// 		fmt.Println(ap)
	// 	}
	// 	wg1.Done()
	// }()
	// wg1.Wait()
}
