package aireos

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ApogeeNetworking/aireoshttp"
	"github.com/ApogeeNetworking/gonetssh"
	"github.com/ApogeeNetworking/gonetssh/universal"
)

// Service ...
type Service struct {
	HTTPClient *aireoshttp.Client
	Client     universal.Device
}

// New ...
func New(host, user, pass string) *Service {
	c, _ := gonetssh.NewDevice(
		host, user, pass, "", gonetssh.DType.CiscoAireos,
	)
	httpClient := aireoshttp.New(host, user, pass, true)
	httpClient.Login()
	return &Service{
		Client:     c,
		HTTPClient: httpClient,
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
func (w *Service) Logout() {
	w.Client.Disconnect()
}

//GetApDetail ...
func (w *Service) GetApDetail(macAddr string) (aireoshttp.ApDetail, error) {
	return w.HTTPClient.GetApDetails(macAddr)
}

// GetAp ...
func (w *Service) GetAp(macAddr string) (CiscoAP, error) {
	var ap CiscoAP
	apDetails, err := w.HTTPClient.GetApDetails(macAddr)
	if err != nil {
		return ap, err
	}
	ap = CiscoAP{
		Name:    apDetails.Name,
		Group:   apDetails.Group,
		Serial:  apDetails.Serial,
		Model:   apDetails.Model,
		MacAddr: apDetails.MacAddr,
	}
	return ap, nil
}

// GetApDb ...
func (w *Service) GetApDb() ([]CiscoAP, error) {
	var aps []CiscoAP
	httpAps, _ := w.HTTPClient.GetAps()
	var wg sync.WaitGroup
	var mut sync.Mutex
	sem := make(chan struct{}, 4)
	for _, httpAp := range httpAps {
		wg.Add(1)
		sem <- struct{}{}
		go func(httpAp aireoshttp.AP) {
			defer func() {
				wg.Done()
				<-sem
			}()
			apdetail, err := w.HTTPClient.GetApDetails(httpAp.MacAddr)
			if err != nil {
				fmt.Printf("error occurred during REQ: %v\n", err)
			}
			mut.Lock()
			aps = append(aps, CiscoAP{
				Name:    apdetail.Name,
				MacAddr: apdetail.MacAddr,
				Group:   apdetail.Group,
				Serial:  apdetail.Serial,
				Model:   apdetail.Model,
			})
			mut.Unlock()
		}(httpAp)
	}
	wg.Wait()
	return aps, nil
}

func trimWS(text string) string {
	tsRe := regexp.MustCompile(`\s+`)
	return tsRe.ReplaceAllString(text, " ")
}

// GetApCdpCli ...
func (w *Service) GetApCdpCli(apName string) ApCdp {
	cmd := fmt.Sprintf("show ap cdp neighbor detail %s", apName)
	out, _ := w.Client.SendCmd(cmd)
	return w.parseApCdp(out)
}

func (w *Service) parseApCdp(out string) ApCdp {
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

// GetApEthStatCli ...
func (w *Service) GetApEthStatCli(apName string) ApIntf {
	cmd := fmt.Sprintf("show ap stats ethernet %s", apName)
	out, _ := w.Client.SendCmd(cmd)
	return w.parseApIntfStat(out)
}

func (w *Service) parseApIntfStat(out string) ApIntf {
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
func (w *Service) SetApName(newApName, arg string) {
	// arg can either be Old AP Name, MacAddr, OR Serial Number
	cmd := fmt.Sprintf("config ap name %s %s", newApName, arg)
	w.Client.SendCmd(cmd)
	time.Sleep(250 * time.Millisecond)
}

// SetApGroup ...
func (w *Service) SetApGroup(groupName, apName string) {
	cmd := fmt.Sprintf("config ap group-name %s %s", groupName, apName)
	w.Client.SendConfig([]string{cmd})
	time.Sleep(250 * time.Millisecond)
}

// RebootAp ...
func (w *Service) RebootAp(apName string) string {
	cmd := fmt.Sprintf("config ap reset %s", apName)
	output, _ := w.Client.SendConfig([]string{cmd})
	time.Sleep(250 * time.Millisecond)
	return output
}

// FactoryResetAp ...
func (w *Service) FactoryResetAp(apName string) (string, error) {
	cmd := fmt.Sprintf("clear ap config %s", apName)
	out, err := w.Client.SendConfig([]string{cmd})
	if err != nil {
		return "", fmt.Errorf("%v", err)
	}
	var result string
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		re := regexp.MustCompile(`All\sAP\sconfiguration(.*)`)
		if re.MatchString(line) {
			result = re.FindString(line)
			break
		}
	}
	return result, nil
}

// SaveConfig ...
func (w *Service) SaveConfig() {
	w.Client.SendConfig([]string{"save config"})
	time.Sleep(250 * time.Millisecond)
}

// LanPortState string enable|disable
type LanPortState string

// LPState {Enable: LanSportState, Disable...}
type LPState struct {
	Enable  LanPortState
	Disable LanPortState
}

// ApLanPortState enum ...
var ApLanPortState = LPState{
	Enable:  "enable",
	Disable: "disable",
}

// ApLanPort ...
type ApLanPort struct {
	// LAN Port-Id
	ID int
	// Enable|Disable
	State LanPortState
	// Operational VLAN
	VlanID int
}

// GetApLanPorts ...
func (w *Service) GetApLanPorts(apName string) ([]ApLanPort, error) {
	cmd := fmt.Sprintf("show ap lan port-summary %s", apName)
	out, _ := w.Client.SendCmd(cmd)
	lines := strings.Split(out, "\n")
	lanRe := regexp.MustCompile(`lan(\d)`)
	statRe := regexp.MustCompile(`enabled|disabled`)
	vlRe := regexp.MustCompile(`\d+`)
	var apLanPorts []ApLanPort
	for _, line := range lines {
		line = strings.ToLower(trimWS(line))
		if lanRe.MatchString(line) {
			// Parse out the Port-Id
			pidMatch := lanRe.FindStringSubmatch(line)
			id, _ := strconv.Atoi(pidMatch[1])
			// Parse out the LanPortState
			status := statRe.FindString(line)
			var state LanPortState
			switch status {
			case "enabled":
				state = LanPortState("enable")
			case "disabled":
				state = LanPortState("disable")
			}
			// Parse out the VlanID
			vlanID, _ := strconv.Atoi(vlRe.FindString(line[10:]))
			apLanPorts = append(apLanPorts, ApLanPort{
				ID:     id,
				State:  state,
				VlanID: vlanID,
			})
		}
	}
	return apLanPorts, nil
}

// SetApLanPortState ...
func (w *Service) SetApLanPortState(apName string, portID int, state LanPortState) {
	cmd := fmt.Sprintf("config ap lan port-id %d %s %s", portID, string(state), apName)
	w.Client.SendCmd(cmd)
}

// SetApLanPortVlanID ...
func (w *Service) SetApLanPortVlanID(apName string, portID, vlanID int) {
	cmd := fmt.Sprintf("config ap lan enable access vlan %d %d %s", vlanID, portID, apName)
	w.Client.SendCmd(cmd)
}
