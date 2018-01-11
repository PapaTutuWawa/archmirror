package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type IPVersion uint8
type ProtocolType uint8

const (
	// Other constants
	ArchLinuxUrl string = "https://www.archlinux.org/mirrorlist/"

	// IPv4 or IPv6
	IPVersion4 IPVersion = iota
	IPVersion6

	// HTTP or HTTPS
	ProtocolTypeHTTP ProtocolType = iota
	ProtocolTypeHTTPS
)

// The configuration of the mirrorlist
type MirrorListConfig struct {
	Protocols  []ProtocolType
	IPVersions []IPVersion
	Country    string
}

// Convert the protocol to an URL parameter
func (t *ProtocolType) ToParameter() string {
	ret := "protocol="

	switch *t {
	case ProtocolTypeHTTP:
		ret += "http"
	case ProtocolTypeHTTPS:
		ret += "https"
	}

	return ret
}

// Convert the IP version to an URL parameter
func (t *IPVersion) ToParameter() string {
	ret := "ip_version="

	switch *t {
	case IPVersion4:
		ret += "4"
	case IPVersion6:
		ret += "6"
	}

	return ret
}

func RequestMirrorList(c *MirrorListConfig) (*[]string, error) {
	parameters := make([]string, 0)
	// Build the Parameters
	// Protocols
	for _, v := range c.Protocols {
		parameters = append(parameters, v.ToParameter())
	}

	// IP versions
	for _, v := range c.IPVersions {
		parameters = append(parameters, v.ToParameter())
	}

	// Country
	parameters = append(parameters, "country="+c.Country)

	// Build the URL and try to send the request
	urlParameters := "?" + strings.Join(parameters, "&")
	resp, err := http.Get(ArchLinuxUrl + urlParameters)
	if err != nil {
		return &[]string{}, err
	}

	// If we don't receive plaintext content: Bail out!
	if resp.Header.Get("Content-Type") != "text/plain" {
		return &[]string{}, errors.New("Expected plaintext, got something else")
	}

	// Read the data that is sent in the body
	strBuf := make([]string, 0)
	reader := bufio.NewReader(resp.Body)
	for {
		str, err := reader.ReadString('\n')

		if strings.Contains(str, "<!DOCTYPE html>") {
			fmt.Println("Found an HTML tag. Perhaps got HTML?")
			fmt.Println("Mirrorlist may not work!")
		}

		// Already activate the mirrors"
		str = strings.Replace(str, "#Server", "Server", -1)
		strBuf = append(strBuf, str)

		// We will read the response stream until an error occurs, which
		// should be when the EOF is reached
		if err != nil {
			break
		}
	}

	return &strBuf, nil
}

// Set up the flags
var (
	// Options affecting the mirrorlist
	IPv4        = flag.Bool("4", true, "Include IPv4 mirrors")
	IPv6        = flag.Bool("6", false, "Include IPv6 mirrors")
	useHTTP     = flag.Bool("http", false, "Include HTTP mirrors")
	useHTTPS    = flag.Bool("https", true, "Include HTTPS mirrors")
	CountryCode = flag.String("country", "", "Mirror location")

	// Everything else
	outputFile = flag.String("out", "mirrorlist", "Output file")
)

func main() {
	// Prepare the MirrorListConfig
	r := &MirrorListConfig{
		Protocols:  []ProtocolType{},
		IPVersions: []IPVersion{},
		Country:    "",
	}

	flag.Parse()

	// IP Version
	if *IPv4 {
		r.IPVersions = append(r.IPVersions, IPVersion4)
	}
	if *IPv6 {
		r.IPVersions = append(r.IPVersions, IPVersion6)
	}

	// Protocols
	if *useHTTP {
		r.Protocols = append(r.Protocols, ProtocolTypeHTTP)
	}
	if *useHTTPS {
		r.Protocols = append(r.Protocols, ProtocolTypeHTTPS)
	}

	// The CountryCode
	r.Country = *CountryCode

	// Check if we have all we need
	if len(r.Protocols) == 0 {
		fmt.Println("No protocol(s) specified!")
		os.Exit(1)
	}
	if len(r.IPVersions) == 0 {
		fmt.Println("No IP version(s) specified!")
		os.Exit(1)
	}
	if r.Country == "" {
		fmt.Println("No county specified!")
		os.Exit(1)
	}
	if *outputFile == "" {
		fmt.Println("No output file specified!")
		os.Exit(1)
	}

	// Fetch the Mirrorlist
	ret, err := RequestMirrorList(r)
	if err != nil {
		fmt.Printf("Failed requesting the mirrorlist: %v\n", err)
		os.Exit(1)
	}

	// Open the file
	// - O_APPEND: We write the lines one after another
	// - O_CREATE: If the file does not exist, we want to create it
	// - O_EXCL: We don't want the file to already exist
	// - O_WRONLY: We only want to write to the file
	file, err := os.OpenFile(*outputFile, os.O_APPEND|os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		os.Exit(1)
	}

	// In case we fail we still want the file to be closed
	defer file.Close()

	// Append all lines
	for _, line := range *ret {
		_, err := file.WriteString(line)
		if err != nil {
			fmt.Printf("Failed writing mirrorlist: %v\n", err)
			os.Exit(1)
		}
	}
}
