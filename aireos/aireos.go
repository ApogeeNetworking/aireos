package aireos

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ApogeeNetworking/gonet"
)

// Wlc ...
type Wlc struct {
	Client *gonet.Gonet
}

// New ...
func New(host, user, pass string) *Wlc {
	return &Wlc{
		Client: &gonet.Gonet{
			IP:       host,
			Username: user,
			Password: pass,
			Vendor:   "Cisco",
			Model:    "aireos",
		},
	}
}

// CiscoAP ...
type CiscoAP struct {
	Name    string
	MacAddr string
	Model   string
	Serial  string
	Group   string
}

// ApCdp ...
type ApCdp struct {
	LocalIntf      string
	RemoteSw       string
	RemoteIntf     string
	RemoteSwIPAddr string
}

// ApIntf ...
type ApIntf struct {
	Name   string
	Status string
	Speed  string
	TxRcv  string
	Drops  string
}

// Logout ...
func (w *Wlc) Logout() {
	w.Client.SendConfig("logout")
	w.Client.Close()
}

// GetApDb ...
func (w *Wlc) GetApDb() []CiscoAP {
	// First Retreive Inventory for All AP's
	out, _ := w.Client.SendCmd("show ap inventory all")
	apNameRe := regexp.MustCompile(`Inven\w+\sfor\s(\S+)`)
	var apNames []string
	lines := strings.Split(out, "\n")
	// Parse AP Name's from Inventory
	for _, line := range lines {
		line = trimWS(line)
		if apNameRe.MatchString(line) {
			m := apNameRe.FindStringSubmatch(line)
			apNames = append(apNames, m[1])
		}
	}
	var aps []CiscoAP
	// Close the Current Client's SSH Connect
	// So we can Spawn Multiple Concurrent Connections
	fmt.Println("Logout Session to collect AP DB")
	w.Logout()
	conns := w.spawnConnPool()
	// Make a Semaphore
	sem := make(chan struct{}, 2)
	// CiscoAP Channel to Receive AP Config Data
	ap := make(chan CiscoAP, len(apNames))
	var wg sync.WaitGroup
	for _, apName := range apNames {
		wg.Add(1)
		sem <- struct{}{}
		go func(apName string) {
			dworker := dbWorkerArgs{
				ConnPool: conns,
				ApName:   apName,
				Sem:      sem,
				ApCh:     ap,
			}
			go w.dbWorker(&dworker)
			cAp := <-ap
			aps = append(aps, cAp)
			wg.Done()
		}(apName)
	}
	wg.Wait()
	defer w.closeConnPool(conns)
	err := w.Client.Connect(3)
	if err != nil {
		fmt.Println(err)
	}
	return aps
}

type dbWorkerArgs struct {
	ConnPool [2]*connPool
	ApName   string
	Sem      chan struct{}
	ApCh     chan CiscoAP
}

func (w *Wlc) dbWorker(worker *dbWorkerArgs) {
	cmd := fmt.Sprintf("show ap config general %s", worker.ApName)
	for {
		for i, conn := range worker.ConnPool {
			if !conn.InUse {
				worker.ConnPool[i].InUse = true
				out, _ := conn.SSH.SendCmd(cmd)
				macRe := regexp.MustCompile(`MAC\sAddress[\\.]+\s(\S+)`)
				apGrpRe := regexp.MustCompile(`AP\sGroup\sName[\\.]+\s(\S+)`)
				apSnRe := regexp.MustCompile(`Serial\sNumber[\\.]+\s(\S+)`)
				apModelRe := regexp.MustCompile(`AP\sModel[\\.]+\s(\S+)`)
				var macAddr, sn, apGrp, apModel string
				if macRe.MatchString(out) {
					m := macRe.FindStringSubmatch(out)
					macAddr = m[1]
				}
				if apGrpRe.MatchString(out) {
					m := apGrpRe.FindStringSubmatch(out)
					apGrp = m[1]
				}
				if apSnRe.MatchString(out) {
					m := apSnRe.FindStringSubmatch(out)
					sn = m[1]
				}
				if apModelRe.MatchString(out) {
					m := apModelRe.FindStringSubmatch(out)
					apModel = m[1]
				}
				ap := CiscoAP{
					Name:    worker.ApName,
					MacAddr: macAddr,
					Serial:  sn,
					Group:   apGrp,
					Model:   apModel,
				}
				fmt.Println(ap)
				worker.ApCh <- ap
				worker.ConnPool[i].InUse = false
				<-worker.Sem
				return
			}
		}
	}
}

type connPool struct {
	InUse bool
	SSH   *gonet.Gonet
}

func (w *Wlc) spawnConnPool() [2]*connPool {
	var conns [2]*connPool
	for i := 0; i < 2; i++ {
		client := &gonet.Gonet{
			IP:       w.Client.IP,
			Username: w.Client.Username,
			Password: w.Client.Password,
			Vendor:   w.Client.Vendor,
			Model:    w.Client.Model,
		}
		err := client.Connect(3)
		if err != nil {
			fmt.Println(err)
		}
		conns[i] = &connPool{SSH: client}
	}
	return conns
}

func (w *Wlc) closeConnPool(conns [2]*connPool) {
	for _, conn := range conns {
		conn.SSH.SendConfig("logout")
		conn.SSH.Close()
	}
}

func trimWS(text string) string {
	tsRe := regexp.MustCompile(`\s+`)
	return tsRe.ReplaceAllString(text, " ")
}

// GetApCdp ...
func (w *Wlc) GetApCdp(apName string) ApCdp {
	cmd := fmt.Sprintf("show ap cdp neighbor detail %s", apName)
	out, _ := w.Client.SendCmd(cmd)
	return w.parseApCdp(out)
}

func (w *Wlc) parseApCdp(out string) ApCdp {
	var apCdp ApCdp
	apNameRe := regexp.MustCompile(`AP\sName:(\S+)`)
	apIntfRe := regexp.MustCompile(`Interface:\s(\w+)`)
	remoteSwRe := regexp.MustCompile(`Device\sID:\s(\S+)`)
	remoteSwIntfRe := regexp.MustCompile(`outgoing\sport\):\s(\S+)`)
	// If No Neighbor Exist Return Empty Cdp Object
	if !apNameRe.MatchString(out) {
		return apCdp
	}
	if apNameRe.MatchString(out) {
		// m := apNameRe.FindStringSubmatch(out)
		// fmt.Println(m[1])
	}
	if apIntfRe.MatchString(out) {
		m := apIntfRe.FindStringSubmatch(out)
		apCdp.LocalIntf = m[1]
	}
	if remoteSwRe.MatchString(out) {
		m := remoteSwRe.FindStringSubmatch(out)
		apCdp.RemoteSw = m[1]
	}
	if remoteSwIntfRe.MatchString(out) {
		m := remoteSwIntfRe.FindStringSubmatch(out)
		apCdp.RemoteIntf = m[1]
	}
	return apCdp
}

// GetApEthStat ...
func (w *Wlc) GetApEthStat(apName string) ApIntf {
	cmd := fmt.Sprintf("show ap stats ethernet %s", apName)
	out, _ := w.Client.SendCmd(cmd)
	return w.parseApIntfStat(out)
}

func (w *Wlc) parseApIntfStat(out string) ApIntf {
	var apIntf ApIntf
	intfNameRe := regexp.MustCompile(`Interface\sname[\\.]+\s(\S+)`)
	statusRe := regexp.MustCompile(`Status[\\.]+\s(\S+)`)
	speedRe := regexp.MustCompile(`Speed[\\.]+\s(\S+)`)
	duplexRe := regexp.MustCompile(`Duplex[\\.]+\s(\S+)`)
	txRe := regexp.MustCompile(`Tx\sBytes[\\.]+\s(\S+)`)
	rcvRe := regexp.MustCompile(`Rx\sBytes[\\.]+\s(\S+)`)
	var speed, duplex string
	if !intfNameRe.MatchString(out) {
		// No Dice on Eth Status for this AP Name
		return apIntf
	}
	if intfNameRe.MatchString(out) {
		m := intfNameRe.FindStringSubmatch(out)
		apIntf.Name = m[1]
	}
	if statusRe.MatchString(out) {
		m := statusRe.FindStringSubmatch(out)
		apIntf.Status = m[1]
	}
	if speedRe.MatchString(out) {
		m := speedRe.FindStringSubmatch(out)
		speed = m[1]
	}
	if duplexRe.MatchString(out) {
		m := duplexRe.FindStringSubmatch(out)
		duplex = m[1]
	}
	if txRe.MatchString(out) && rcvRe.MatchString(out) {
		m1 := txRe.FindStringSubmatch(out)
		m2 := rcvRe.FindStringSubmatch(out)
		apIntf.TxRcv = m1[1] + "/" + m2[1]
	}
	apIntf.Speed = speed + " " + duplex
	return apIntf
}

// SetApName modifies AP Name using ARG (Old APName|MAC Addr|Serial Number)
func (w *Wlc) SetApName(newApName, arg string) {
	// arg can either be Old AP Name, MacAddr, OR Serial Number
	cmd := fmt.Sprintf("config ap name %s %s", newApName, arg)
	w.Client.SendConfig(cmd)
	time.Sleep(250 * time.Millisecond)
}

// SetApGroup ...
func (w *Wlc) SetApGroup(groupName, apName string) {
	cmd := fmt.Sprintf("config ap group-name %s %s", groupName, apName)
	w.Client.SendConfig(cmd)
	// Confirm that YES We Do Want to Change the AP Group
	w.Client.SendConfig("y")
	time.Sleep(250 * time.Millisecond)
}

// SaveConfig ...
func (w *Wlc) SaveConfig() {
	w.Client.SendConfig("save config")
	w.Client.SendConfig("y")
	time.Sleep(250 * time.Millisecond)
}
