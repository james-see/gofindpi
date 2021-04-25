package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"github.com/jaypipes/ghw"
)

// notes
// this needs to print any raspberry pi's found as one liner if not using this
// package after pinging all ip addresses on the network
// arp -a | awk '{print $2,$4}' | grep -e b8:27:eb -e dc:a6:32 -e e4:5f:01)

// can use any arbitrary trigram for any physical address identifier for devices
var matchPI = []string{"b8:27:eb", "dc:a6:32", "e4:5f:01"}
var piFoundList, aliveDeviceFoundList, ipFound []string

// finds out if an item in slice matches
func find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// handles writing data to a filename at user home folder
func writer(coolArray []string, fileName string) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.OpenFile(fmt.Sprintf("%s/%s", dirname, fileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	datawriter := bufio.NewWriter(file)

	for _, data := range coolArray {
		_, _ = datawriter.WriteString(data + "\n")
	}

	datawriter.Flush()
}

// ping each ip address
func pingMe(ipAddress string, wg *sync.WaitGroup) {
	pinger, err := ping.NewPinger(ipAddress)
	if err != nil {
		panic(err)
	}
	defer pinger.Stop()
	pinger.Count = 1
	pinger.Timeout = time.Millisecond * 800
	pinger.OnRecv = func(pkt *ping.Packet) {
		ipFound = append(ipFound, pkt.IPAddr.String())
	}
	err = pinger.Run()
	if err != nil {
		panic(err)
	}

	wg.Done()
	return
}

// split out the arp results into slices for device list and pi list
func splitAndStore(dataFromArp []byte) {
	var sliceOfMac = []string{}
	var piSlice = []string{}
	// specific formatting shit from the arp command itself
	stringer := strings.Split(string(dataFromArp), "?")
	for _, item := range stringer {
		// skip wasted strings
		if strings.Contains(item, "incomplete") {
			continue
		}
		// iterate and parse out just the macaddress and ip address from the arp string slice
		if strings.ContainsAny(item, "(") {
			ipData := strings.Split(strings.Split(item, "(")[1], ")")[0]
			macAddressData := strings.Split(strings.Split(strings.Split(strings.Split(item, "(")[1], ")")[1], "at")[1], "on")[0]
			macAddressData = strings.TrimSpace(macAddressData)
			vendorMac := strings.Split(macAddressData, ":")
			vendorMacString := fmt.Sprintf("%s:%s:%s", vendorMac[0], vendorMac[1], vendorMac[2])
			// check to see if mac address trigram matches any of the pi trigrams
			found := find(matchPI, vendorMacString)
			updatedString := fmt.Sprintf("ip:%v mac:%v", ipData, macAddressData)
			sliceOfMac = append(sliceOfMac, updatedString)
			if found {
				piSlice = append(piSlice, updatedString)
			}
		} else {
			continue
		}
	}
	writer(sliceOfMac, "devicesfound.txt")
	writer(piSlice, "pilist.txt")
	fmt.Printf("Found %v devices including %v raspberry pis on the network in\n", len(sliceOfMac), len(piSlice))
}

// runs the arp command example output below
// $ arp -a
//  (192.168.1.1) at cc:40:d0:54:4e:f4 on en0 ifscope [ethernet]
//  (192.168.1.3) at d4:ab:cd:7:4:13 on en0 ifscope [ethernet]
//  (192.168.1.7) at c8:69:cd:4a:c1:27 on en0 ifscope [ethernet]
//  (192.168.1.13) at 68:d9:3c:8a:ad:5c on en0 ifscope [ethernet]
func runCmd() []byte {
	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		fmt.Printf("%s", err)
		fmt.Println("fucked")
	}
	return out
}

// removes the last ip address trigram
func splitMe(item string) string {
	last := strings.Split(item, ".")
	s := last[len(last)-1]
	item = strings.Replace(item, fmt.Sprintf(".%s", s), ".0/24", -1)
	return item
}

// appends all the ip addresses as strings
func appendMe(item string) []string {
	arr := []string{}
	i := 1
	item = strings.Replace(item, ".0/24", ".", -1)
	for i < 256 {
		arr = append(arr, fmt.Sprintf("%v%v", item, i))
		i++
	}
	return arr
}

// gets the core count for the cpu info
func getCores() uint32 {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "machdep.cpu.thread_count").Output()
		if err != nil {
			fmt.Println(err)
		}
		var totalCores uint32
		if _, err := fmt.Sscanf(string(out), "machdep.cpu.thread_count: %2d", &totalCores); err == nil {
			return totalCores
		}

	} else {
		cpuData, err := ghw.CPU()
		if err != nil {
			fmt.Println(err)
		}
		var totalCores uint32
		totalCores = cpuData.TotalCores
		return totalCores
	}
	return 0
}

func main() {
	// used for goroutine to avoid memory errors
	var w sync.WaitGroup
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
	}
	var currentIP string
	var listOfIps []string
	for _, address := range addrs {

		// check the address type and if it is not a loopback the display it
		// = GET LOCAL IP ADDRESS

		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				// fmt.Println("Current IP address : ", ipnet.IP.String())
				currentIP = ipnet.IP.String()
				listOfIps = append(listOfIps, currentIP)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}

	// print all of the network interface options
	for i, item := range listOfIps {
		last := strings.Split(item, ".")
		s := last[len(last)-1]
		item = strings.Replace(item, fmt.Sprintf(".%s", s), ".0/24", -1)
		fmt.Println("Option:", i, item)
	}

	// get user to select option number and press enter
	fmt.Printf("You have %v cores available for processing.\n", getCores())
	fmt.Print("select option for finding pi on what network: ")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	if len(input.Text()) > 1 {
		panic("input is wrong, must be a single number")
	}
	startTime := time.Now()
	i1, err := strconv.Atoi(input.Text())
	if err != nil {
		fmt.Println(i1)
	}
	// ensure to signify selected ip address
	chosenIP := listOfIps[i1]
	// convert selected ip address to 0/24
	fixedip := splitMe(chosenIP)
	// explode out selection to 1 through 256
	stringArray := appendMe(fixedip)
	// loop through ip range and ping each in parallel
	for i := 0; i < len(stringArray); i++ {
		w.Add(1)
		go pingMe(stringArray[i], &w)
	}
	w.Wait()
	// run arp command to get all pinged devices found
	data := runCmd()
	// iterate through found devices and save pilist and devicelist
	splitAndStore(data)
	// get time it took to run after the input
	endTime := time.Now()
	diff := endTime.Sub(startTime)
	fmt.Printf("%f seconds\n", diff.Seconds())
	fmt.Println("devicesfound.txt and pilist.txt saved to user's home folder")
}
