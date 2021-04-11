package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	nmap "github.com/Ullaakut/nmap/v2"
)

var matchPI = []string{"B8:27:EB", "DC:A6:32", "E4:5F:01"}

func find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func writer(coolArray []string, fileName string) {
	file, err := os.OpenFile(fmt.Sprintf("~/%s", fileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

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

func scanMe(ipAddress string, workerNum int) (string, string) {
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
		log.Printf("unable to create nmap scanner: %v %v", err, workerNum)
	}
	result, warnings, err := scanner.Run()
	fmt.Println(result.Hosts[0].Addresses)
	if len(result.Hosts[0].Addresses) > 1 {
		fmt.Printf("ALIVE! %v\n", result.Hosts[0].Addresses[1])
		aliveDeviceFound = strings.Join([]string{result.Hosts[0].Addresses[0].String(), result.Hosts[0].Addresses[1].String()}, ",")
		vendorMac := strings.Split(result.Hosts[0].Addresses[1].String(), ":")
		vendorMacString := fmt.Sprintf("%s:%s:%s", vendorMac[0], vendorMac[1], vendorMac[2])
		found := find(matchPI, vendorMacString)
		if found {
			fmt.Printf("PI FOUND! AT %v\n", result.Hosts[0].Addresses[0])
			piFound = result.Hosts[0].Addresses[0].String()
		}
	}
	if err != nil {
		log.Printf("unable to run nmap scan: %v %v", err, workerNum)
	}

	if warnings != nil {
		log.Printf("Warnings: \n %v", warnings)
	}
	return piFound, aliveDeviceFound
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
	for _, arrItem := range arr {
		ip, _, err := net.ParseCIDR(arrItem)
		if err != nil {
			fmt.Println(err)
		}
		ips = append(ips, ip)
	}
	return ips, arr
}

func main() {
	var piFoundList, aliveDeviceFoundList []string
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
	finalArray, stringArray := appendMe(fixedip)
	fmt.Println(finalArray[5], stringArray[5])
	// Equivalent to `/usr/local/bin/nmap -p 80,443,843 google.com facebook.com youtube.com`,
	// with a 5 minute timeout.
	i := 0
	for i < len(stringArray) {
		piFound, deviceFound := scanMe(stringArray[i], i)
		if piFound != "" {
			piFoundList = append(piFoundList, piFound)
		}
		if deviceFound != "" {
			aliveDeviceFoundList = append(aliveDeviceFoundList, deviceFound)
		}
		i++
	}
	writer(piFoundList, "pilist.txt")
	writer(aliveDeviceFoundList, "devicesfound.txt")
	// scanMe("10.10.200.*", 1)
	// fmt.Printf("Nmap done: %d hosts up scanned in %3f seconds\n", len(result.Hosts), result.Stats.Finished.Elapsed)
}
