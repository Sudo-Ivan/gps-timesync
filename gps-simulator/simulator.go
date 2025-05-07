package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"time"

	// For Linux PTY creation
	"github.com/creack/pty"
	// For Windows COM port handling (standard library)
)

var (
	talkerID        string
	numSatellites   int
	comPortName     string    // For Windows
	staticLatitude  = 51.1173 // Slightly more realistic degrees.decimal format for NMEA
	staticLongitude = -2.5166 // Example: 5107.038,N -> 51 deg, 07.038 min. -00231.000,W -> 2 deg, 31.000 min
)

// Converts degrees to NMEA DDDMM.MMMM format
func toNMEALat(deg float64) (string, string) {
	absDeg := deg
	ns := "N"
	if deg < 0 {
		absDeg = -deg
		ns = "S"
	}
	d := int(absDeg)
	m := (absDeg - float64(d)) * 60
	return fmt.Sprintf("%02d%07.4f", d, m), ns
}

func toNMEALon(deg float64) (string, string) {
	absDeg := deg
	ew := "E"
	if deg < 0 {
		absDeg = -deg
		ew = "W"
	}
	d := int(absDeg)
	m := (absDeg - float64(d)) * 60
	return fmt.Sprintf("%03d%07.4f", d, m), ew
}

// Function to write NMEA sentences periodically
func writeNMEASentences(writer io.Writer) {
	ticker := time.NewTicker(1 * time.Second) // Send data every 1 second for more responsiveness
	defer ticker.Stop()

	for {
		<-ticker.C
		timeUTC := time.Now().UTC()
		timeStrNMEA := timeUTC.Format("150405.00")
		dateStrNMEA := timeUTC.Format("020106") // DDMMYY

		latNMEA, ns := toNMEALat(staticLatitude)
		lonNMEA, ew := toNMEALon(staticLongitude)

		var sentences []string

		// GPRMC sentence
		// $--RMC,UTC,Status,Lat,N/S,Lon,E/W,Spd,Cog,Date,MV,MVE,Mode*CS
		gprmcBody := fmt.Sprintf("%sRMC,%s,A,%s,%s,%s,%s,0.1,0.0,%s,,,A", talkerID, timeStrNMEA, latNMEA, ns, lonNMEA, ew, dateStrNMEA)
		sentences = append(sentences, addChecksumAndCRLF(gprmcBody))

		// GPGGA sentence
		// $--GGA,UTC,Lat,N/S,Lon,E/W,FixQual,NumSat,HDOP,Alt,M,GeoSep,M,Age,DiffSta*CS
		hdop := 1.0 + rand.Float64()                // Random HDOP between 1.0 and 2.0
		altitude := 123.4 + (rand.Float64()-0.5)*10 // Random altitude around 123.4m
		gpggaBody := fmt.Sprintf("%sGGA,%s,%s,%s,%s,%s,1,%02d,%.1f,%.1f,M,0.0,M,,0000",
			talkerID, timeStrNMEA, latNMEA, ns, lonNMEA, ew, numSatellites, hdop, altitude)
		sentences = append(sentences, addChecksumAndCRLF(gpggaBody))

		// GPGSV sentences (Satellites in View)
		// $--GSV,NumMsg,MsgNum,NumSV,PRN,Elev,Azim,SNR,PRN,Elev,Azim,SNR,...*CS
		// Simulate up to 12 satellites in view for this example, can be expanded.
		numSVInView := numSatellites + rand.Intn(5) // Slightly more in view than locked
		if numSVInView < numSatellites {
			numSVInView = numSatellites
		}
		if numSVInView > 20 { // Cap at 20 for this simulation
			numSVInView = 20
		}
		numGSVMsgs := (numSVInView + 3) / 4 // Each GSV message can hold up to 4 SVs

		for i := 0; i < numGSVMsgs; i++ {
			msgNum := i + 1
			gsvBody := fmt.Sprintf("%sGSV,%d,%d,%02d", talkerID, numGSVMsgs, msgNum, numSVInView)
			for sv := 0; sv < 4; sv++ {
				svIndex := i*4 + sv
				if svIndex < numSVInView {
					prn := svIndex + 1         // Simple PRN assignment
					elevation := rand.Intn(90) // 0-89 degrees
					azimuth := rand.Intn(360)  // 0-359 degrees
					snr := rand.Intn(50) + 20  // Meaningful SNR 20-70
					gsvBody += fmt.Sprintf(",%02d,%02d,%03d,%02d", prn, elevation, azimuth, snr)
				} else {
					// Fill with empty fields if no more SVs for this message but message needs 4 slots
					// No, NMEA spec says to only list actual SVs, not empty trailing fields for them.
					break
				}
			}
			sentences = append(sentences, addChecksumAndCRLF(gsvBody))
		}

		for _, sentence := range sentences {
			_, err := writer.Write([]byte(sentence))
			if err != nil {
				log.Printf("Error writing NMEA sentence: %v", err)
				// Consider whether to return or just log and continue
				return
			}
			log.Printf("Sent: %s", strings.TrimSpace(sentence))
		}
	}
}

func addChecksumAndCRLF(sentenceBody string) string {
	checksum := 0
	// Checksum is XOR of all characters between $ (exclusive) and * (exclusive)
	// Our sentenceBody starts after the talkerID, which itself comes after a $ implicitly
	// So, for $GPRMC, body is GPRMC,...
	// Let's ensure the input `sentenceBody` is just the part after `$`
	// For example, if sentenceBody = "GPRMC,081836,A,3751.65,S,14507.36,E,000.0,360.0,130998,011.3,E"
	// then we feed this directly.

	// The sentenceBody starts with talkerID then message type, e.g. "GPRMC,..."
	// The checksum is calculated on this part.
	for _, char := range []byte(sentenceBody) {
		checksum ^= int(char)
	}
	return fmt.Sprintf("$%s*%02X\r\n", sentenceBody, checksum)
}

func main() {
	flag.StringVar(&talkerID, "talker", "GP", "NMEA Talker ID (e.g., GP, GN, GL, GA)")
	flag.IntVar(&numSatellites, "sats", 7, "Number of satellites to simulate in $GPGGA (fix quality)")

	if runtime.GOOS == "windows" {
		flag.StringVar(&comPortName, "com", "", "COM port to use for simulation (e.g., COM3) - this should be one end of a virtual serial port pair")
	}
	flag.Parse()

	if strings.ToUpper(talkerID) != "GP" && strings.ToUpper(talkerID) != "GN" && strings.ToUpper(talkerID) != "GL" && strings.ToUpper(talkerID) != "GA" {
		log.Fatalf("Invalid talker ID: %s. Must be GP, GN, GL, or GA.", talkerID)
	}
	talkerID = strings.ToUpper(talkerID)

	if numSatellites < 0 || numSatellites > 12 { // GPGGA practical limit for 'NumSat' is 0-12 for good quality data
		log.Printf("Warning: Number of satellites (%d) is unusual. Adjusting to a common range (0-12 for GPGGA).", numSatellites)
		if numSatellites < 0 {
			numSatellites = 0
		}
		if numSatellites > 12 {
			numSatellites = 12
		}
	}

	log.Printf("GPS Simulator starting with Talker ID: %s, Satellites (GPGGA): %d", talkerID, numSatellites)

	var err error
	var port io.ReadWriteCloser // Use ReadWriteCloser for serial/pty

	if runtime.GOOS == "linux" {
		ptmx, tty, errPty := pty.Open()
		if errPty != nil {
			log.Fatalf("Failed to open PTY: %v", errPty)
		}
		// ptmx is closed via defer in main, tty is also closed.
		// However, if writeNMEASentences returns due to an error, these defers won't run until main exits.
		// This is generally fine for a simulator like this.
		defer ptmx.Close()
		defer tty.Close()

		log.Printf("PTY created. Slave device: %s", tty.Name())
		log.Println("Run gps-timesync with: -d", tty.Name())
		port = ptmx // Write to the master side of the PTY

	} else if runtime.GOOS == "windows" {
		if comPortName == "" {
			log.Fatal("On Windows, you must specify the COM port using the -com flag (e.g., -com COM3)")
		}

		log.Printf("Attempting to open COM port: %s", comPortName)
		port, err = os.OpenFile(comPortName, os.O_RDWR, 0)
		if err != nil {
			log.Fatalf("Failed to open COM port %s: %v", comPortName, err)
		}
		defer port.Close()
		log.Printf("Successfully opened %s. Configure gps-timesync to use the other COM port of the virtual pair.", comPortName)

	} else {
		log.Fatalf("Unsupported operating system: %s", runtime.GOOS)
		return
	}

	rand.Seed(time.Now().UnixNano()) // Initialize random seed
	log.Println("Starting to write NMEA sentences...")
	writeNMEASentences(port)

	log.Println("Simulator finished or encountered an error.")
}
