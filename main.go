package main

import (
	"bufio"
	"context"
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

	nmap "github.com/Ullaakut/nmap/v2"
)

// this needs to get to the pifoundlist.txt
// arp -a | awk '{print $2,$4}' | grep -e b8:27:eb -e dc:a6:32 -e e4:5f:01)

var matchPI = []string{"B8:27:EB", "DC:A6:32", "E4:5F:01"}
var piFoundList, aliveDeviceFoundList, ipFound []string

func find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func findString(src string, val string) bool {
	for _, item := range src {
		if string(item) == val {
			return true
		}
	}
	return false
}

func writer(coolArray []string, fileName string) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.OpenFile(fmt.Sprintf("%s/%s", dirname, fileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	datawriter := bufio.NewWriter(file)

	for _, data := range coolArray {
		_, _ = datawriter.WriteString(data + "\n")
	}

	datawriter.Flush()
	file.Close()
}

func pingMe(ipAddress string, wg *sync.WaitGroup, m *sync.Mutex) {
	pinger, err := ping.NewPinger(ipAddress)
	if err != nil {
		panic(err)
	}
	pinger.Count = 1
	pinger.Timeout = time.Second
	pinger.OnRecv = func(pkt *ping.Packet) {
		//fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n",pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
		ipFound = append(ipFound, pkt.IPAddr.String())
		// pinger.Stop()
	}
	//m.Lock()
	err = pinger.Run()
	if err != nil {
		panic(err)
	}
	//m.Unlock()

	wg.Done()
	return
}

func splitAndStore(dataFromArp []byte) {
	//scanner := bufio.NewScanner.Reader(dataFromArp)
	//scanner.read()
	var sliceOfMac = []string{}
	stringer := strings.Split(string(dataFromArp), "?")
	//fmt.Println(find())
	for _, item := range stringer {
		if strings.Contains(item, "incomplete") {
			continue
		}
		if strings.ContainsAny(item, "(") {
			ipData := strings.Split(strings.Split(item, "(")[1], ")")[0]
			macAddressData := strings.Split(strings.Split(strings.Split(strings.Split(item, "(")[1], ")")[1], "at")[1], "on")[0]
			fmt.Println(ipData, macAddressData)
			updatedString := fmt.Sprintf("ip:%v mac:%v", ipData, macAddressData)
			sliceOfMac = append(sliceOfMac, updatedString)
		} else {
			continue
		}
	}
	writer(sliceOfMac, "devicesfound.txt")
}

func runCmd() []byte {
	//cmd := "-a | awk '{print $2,$4}' | grep -e b8:27:eb -e dc:a6:32 -e e4:5f:01)"
	out, err := exec.Command("arp", "-a").Output()

	// if there is an error with our execution
	// handle it here
	if err != nil {
		fmt.Printf("%s", err)
		fmt.Println("fucked")
	}
	return out
}

func scanMe(ipAddress string, wg *sync.WaitGroup, m *sync.Mutex) {
	var piFound, aliveDeviceFound string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	scanner, err := nmap.NewScanner(
		nmap.WithTargets(ipAddress),
		nmap.WithPingScan(),
		nmap.WithContext(ctx),
		nmap.WithVerbosity(5),
	)
	if err != nil {
		log.Printf("unable to create nmap scanner: %v", err)
	}

	result, warnings, err := scanner.Run()
	fmt.Println(result.Hosts[0].Addresses)
	m.Lock()
	if len(result.Hosts[0].Addresses) > 1 {
		fmt.Printf("ALIVE! %v\n", result.Hosts[0].Addresses[1])
		aliveDeviceFound = strings.Join([]string{result.Hosts[0].Addresses[0].String(), result.Hosts[0].Addresses[1].String()}, ",")
		vendorMac := strings.Split(result.Hosts[0].Addresses[1].String(), ":")
		vendorMacString := fmt.Sprintf("%s:%s:%s", vendorMac[0], vendorMac[1], vendorMac[2])
		found := find(matchPI, vendorMacString)
		if found {
			//fmt.Printf("PI FOUND! AT %v\n", result.Hosts[0].Addresses[0])
			piFound = result.Hosts[0].Addresses[0].String()
		}
	}

	if err != nil {
		log.Printf("unable to run nmap scan: %v", err)
	}

	if warnings != nil {
		log.Printf("Warnings: \n %v", warnings)
	}
	if piFound != "" {
		piFoundList = append(piFoundList, piFound)
	}
	if aliveDeviceFound != "" {
		aliveDeviceFoundList = append(aliveDeviceFoundList, aliveDeviceFound)
	}
	m.Unlock()
	wg.Done()

}

// removes the last ip address location and adds 0/24
func splitMe(item string) string {
	last := strings.Split(item, ".")
	s := last[len(last)-1]
	// fmt.Println("Last", s)
	item = strings.Replace(item, fmt.Sprintf(".%s", s), ".0/24", -1)
	return item
}

func appendMe(item string) ([]net.IP, []string) {
	arr := []string{}
	ips := []net.IP{}
	i := 1
	item = strings.Replace(item, ".0/24", ".", -1)
	for i < 256 {
		arr = append(arr, fmt.Sprintf("%v%v", item, i))
		i++
	}
	// for _, arrItem := range arr {
	// 	ip, _, err := net.ParseCIDR(arrItem)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// 	ips = append(ips, ip)
	// }
	return ips, arr
}

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
	var w sync.WaitGroup
	var m sync.Mutex
	addrs, err := net.InterfaceAddrs()
	fmt.Println(addrs)
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

	// print all the options
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
	i1, err := strconv.Atoi(input.Text())
	if err == nil {
		fmt.Println(i1)
	}
	// ensure to signify selected ip address
	chosenIP := listOfIps[i1]
	// convert selected ip address to 0/24
	fixedip := splitMe(chosenIP)
	// explode out selection to 1 through 256
	_, stringArray := appendMe(fixedip)
	//fmt.Println(stringArray[5])
	for i := 0; i < len(stringArray); i++ {
		w.Add(1)
		go pingMe(stringArray[i], &w, &m)
	}
	w.Wait()
	data := runCmd()
	//fmt.Println(string(data))
	splitAndStore(data)
	writer(piFoundList, "pilist.txt")

}
