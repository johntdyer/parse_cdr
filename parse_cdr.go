package main

// http://stackoverflow.com/questions/24634114/slowdown-when-parsing-multiple-files-in-parallel-using-goroutines
//http://ernestmicklei.com/2013/10/10/a-case-of-sizing-and-draining-buffered-go-channels/

import (
	"encoding/xml"
	"flag"
	"fmt"
	humanize "github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"os"
	"runtime"
	"strconv"
	"sync"
)

var inputFile = flag.String("infile", "example.xml", ".")

func FloatToString(input_num float64) string {
	return strconv.FormatFloat(input_num, 'f', 2, 64)
}

type OutboundDialString struct {
	Description string `xml:"description,attr"`
	Prefix      string `xml:"prefix,attr"`
}

type PrefixList struct {
	Prefix map[string]float64
}

type Session struct {
	Duration            string             `xml:"durationMinutes,attr"`
	Transport           string             `xml:"transportCharges,attr"`
	Direction           string             `xml:"direction,attr"`
	TotalCharges        string             `xml:"totalCharges,attr"`
	PlatformRate        string             `xml:"platformRate,attr"`
	DialString          OutboundDialString `xml:"outboundDialString"`
	TransferCharges     string             `xml:"transferCharges,attr"`
	RecordingCharges    string             `xml:"recordingCharges,attr"`
	ConferencingCharges string             `xml:"conferencingCharges,attr"`
	PayphoneCharges     string             `xml:"payphoneCharges,attr"`
}

type Categories struct {
	Inbound  float64
	Outbound float64
}

// Results stores the calculation of our totals
type Results struct {
	Transport           Categories
	Duration            Categories
	TotalCharges        Categories
	PlatformRate        Categories
	TransferCharges     Categories
	RecordingCharges    Categories
	ConferencingCharges Categories
	PayphoneCharges     Categories
	Calls               Categories
	Total               int
}

// Comms is a struct for all of our inter-go routine communications
type Comms struct {
	cdrWg            sync.WaitGroup
	cdrChan          chan xml.Token
	totallerWg       sync.WaitGroup
	resultChan       chan *Results
	totallerDoneChan chan bool
}

func main() {
	// Parse the command line
	flag.Parse()

	// Ensure we are using all available cores for maximum performance
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Initialize our structs
	results := &Results{}
	comms := &Comms{
		resultChan:       make(chan *Results),
		totallerDoneChan: make(chan bool),
	}

	// Launch the go routine that handles the totalling of the CDRs
	comms.totallerWg.Add(1)
	go totaller(results, comms)

	// Open the XML file and create a new decoder stream
	xmlFile, err := os.Open(*inputFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer xmlFile.Close()
	decoder := xml.NewDecoder(xmlFile)

	// Read the XML decoder stream
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		comms.cdrWg.Add(1)
		processCDR(t, decoder, comms)
	}

	// Wait for the CDR processing routines to all complete
	comms.cdrWg.Wait()

	// Signal to the totaller we are done so that go routine exits
	comms.totallerDoneChan <- true
	comms.totallerWg.Wait()
	// Output the results to STDOUT
	outputResults(results)
}

// processCDR processes an individual CDR and passes the result to the totaller go routine
func processCDR(t xml.Token, decoder *xml.Decoder, comms *Comms) {
	// Inspect the type of the token just read.
	switch startEle := t.(type) {
	case xml.StartElement:
		if startEle.Name.Local == "session" {
			session := &Session{}
			decoder.DecodeElement(&session, &startEle)
			comms.cdrWg.Add(1)
			go work(session, comms)
		}
	}
	comms.cdrWg.Done()
}

func work(session *Session, comms *Comms) {
	result := &Results{}

	// Transport
	total, _ := strconv.ParseFloat(session.TotalCharges, 64)
	duration, _ := strconv.ParseFloat(session.Duration, 64)
	transfer, _ := strconv.ParseFloat(session.TransferCharges, 64)
	payphone, _ := strconv.ParseFloat(session.PayphoneCharges, 64)
	platform, _ := strconv.ParseFloat(session.PlatformRate, 64)
	recording, _ := strconv.ParseFloat(session.RecordingCharges, 64)
	transport, _ := strconv.ParseFloat(session.Transport, 64)
	conferencing, _ := strconv.ParseFloat(session.ConferencingCharges, 64)

	switch session.Direction {
	case "inbound":
		result.Calls.Inbound = 1
		result.Transport.Inbound = transport
		result.Duration.Inbound = duration
		result.TotalCharges.Inbound = total
		result.PlatformRate.Inbound = platform
		result.TransferCharges.Inbound = transfer
		result.RecordingCharges.Inbound = recording
		result.ConferencingCharges.Inbound = conferencing
		result.PayphoneCharges.Inbound = payphone
	case "outbound":
		//fmt.Println(session.DialString.Prefix)
		result.Calls.Outbound = 1
		result.TotalCharges.Outbound = total
		result.PlatformRate.Outbound = platform
		result.TransferCharges.Outbound = transfer
		result.RecordingCharges.Outbound = recording
		result.ConferencingCharges.Outbound = conferencing
		result.PayphoneCharges.Outbound = payphone
		result.Transport.Outbound = transport
		result.Duration.Outbound = duration
	}

	comms.resultChan <- result
	comms.cdrWg.Done()
}

// outputResults outputs the totalled results to STDOUT
func outputResults(results *Results) {

	dataTotal := [][]string{
		[]string{"CDR Counts ", humanize.Comma(int64(results.Calls.Inbound)), humanize.Comma(int64(results.Calls.Outbound)), humanize.Comma(int64(results.Total))},
		[]string{"Duration (Minutes) ", humanize.Comma(int64(results.Duration.Inbound)), humanize.Comma(int64(results.Duration.Outbound)), humanize.Comma(int64(results.Duration.Inbound + results.Duration.Outbound))},
	}

	dataItemized := [][]string{
		[]string{"Transport", FloatToString(results.Transport.Inbound), FloatToString(results.Transport.Outbound), FloatToString(results.Transport.Inbound + results.Transport.Outbound)},
		[]string{"Platform", FloatToString(results.PlatformRate.Inbound), FloatToString(results.PlatformRate.Outbound), FloatToString(results.PlatformRate.Inbound + results.PlatformRate.Outbound)},
		[]string{"Payphone", FloatToString(results.PayphoneCharges.Inbound), FloatToString(results.PayphoneCharges.Outbound), FloatToString(results.PayphoneCharges.Inbound + results.PayphoneCharges.Outbound)},
		[]string{"Transfer", FloatToString(results.TransferCharges.Inbound), FloatToString(results.TransferCharges.Outbound), FloatToString(results.TransferCharges.Inbound + results.TransferCharges.Outbound)},
		[]string{"Recording", FloatToString(results.RecordingCharges.Inbound), FloatToString(results.RecordingCharges.Outbound), FloatToString(results.RecordingCharges.Inbound + results.RecordingCharges.Outbound)},
		[]string{"Conferencing", FloatToString(results.ConferencingCharges.Inbound), FloatToString(results.ConferencingCharges.Inbound), FloatToString(results.ConferencingCharges.Inbound + results.ConferencingCharges.Inbound)},
	}
	tableTotal := tablewriter.NewWriter(os.Stdout)
	tableTotal.SetHeader([]string{"Category", "Inbound", "Outbound", "Total"})
	tableTotal.AppendBulk(dataTotal)
	tableTotal.Render()

	tableItemized := tablewriter.NewWriter(os.Stdout)
	tableItemized.SetHeader([]string{"Category", "Inbound", "Outbound", "Total"})
	tableItemized.SetFooter([]string{"Total Charges", "$" + FloatToString(results.TotalCharges.Inbound), "$" + FloatToString(results.TotalCharges.Outbound), "$" + FloatToString(results.TotalCharges.Inbound+results.TotalCharges.Outbound)}) // Add Footer
	tableItemized.AppendBulk(dataItemized)
	tableItemized.Render()

}

// totaller is a go routine that runs collects all results and totals them
func totaller(results *Results, comms *Comms) {

Loop:
	for {
		select {
		case <-comms.totallerDoneChan:
			break Loop
		case result := <-comms.resultChan:

			results.Calls.Outbound += result.Calls.Outbound
			results.Duration.Outbound += result.Duration.Outbound
			results.Transport.Outbound += result.Transport.Outbound
			results.TotalCharges.Outbound += result.TotalCharges.Outbound
			results.PlatformRate.Outbound += result.PlatformRate.Outbound
			results.TransferCharges.Outbound += result.TransferCharges.Outbound
			results.PayphoneCharges.Outbound += result.PayphoneCharges.Outbound
			results.RecordingCharges.Outbound += result.RecordingCharges.Outbound
			results.ConferencingCharges.Outbound += result.ConferencingCharges.Outbound

			results.Calls.Inbound += result.Calls.Inbound
			results.Duration.Inbound += result.Duration.Inbound
			results.Transport.Inbound += result.Transport.Inbound
			results.TotalCharges.Inbound += result.TotalCharges.Inbound
			results.PlatformRate.Inbound += result.PlatformRate.Inbound
			results.PayphoneCharges.Inbound += result.PayphoneCharges.Inbound
			results.TransferCharges.Inbound += result.TransferCharges.Inbound
			results.RecordingCharges.Inbound += result.RecordingCharges.Inbound
			results.ConferencingCharges.Inbound += result.ConferencingCharges.Inbound

			results.Total++
		}
	}
	comms.totallerWg.Done()
}
