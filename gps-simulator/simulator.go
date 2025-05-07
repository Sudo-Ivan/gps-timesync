package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
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
	talkerID      string
	numSatellites int
	comPortName   string // For Windows
	// Initial position with slight random variation
	baseLatitude  = 51.1173
	baseLongitude = -2.5166
	// Movement parameters
	movementEnabled bool
	movementSpeed   float64 // degrees per second
	movementBearing float64 // degrees from north
	// Satellite configuration
	gpsPRNs     = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	glonassPRNs = []int{65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96}
)

// Position represents a geographic position
type Position struct {
	Latitude  float64
	Longitude float64
}

// Calculate new position based on movement
func calculateNewPosition(current Position, speed float64, bearing float64, elapsed time.Duration) Position {
	// Convert speed from degrees/second to degrees
	distance := speed * elapsed.Seconds()

	// Convert bearing to radians
	bearingRad := bearing * (math.Pi / 180.0)

	// Calculate new position using great circle formula
	lat1 := current.Latitude * (math.Pi / 180.0)
	lon1 := current.Longitude * (math.Pi / 180.0)

	angularDistance := distance * (math.Pi / 180.0)

	lat2 := math.Asin(math.Sin(lat1)*math.Cos(angularDistance) +
		math.Cos(lat1)*math.Sin(angularDistance)*math.Cos(bearingRad))

	lon2 := lon1 + math.Atan2(math.Sin(bearingRad)*math.Sin(angularDistance)*math.Cos(lat1),
		math.Cos(angularDistance)-math.Sin(lat1)*math.Sin(lat2))

	return Position{
		Latitude:  lat2 * (180.0 / math.Pi),
		Longitude: lon2 * (180.0 / math.Pi),
	}
}

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
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	currentPos := Position{
		Latitude:  baseLatitude,
		Longitude: baseLongitude,
	}
	lastUpdate := time.Now()

	// Retry configuration
	maxRetries := 3
	retryDelay := time.Second * 2

	for {
		<-ticker.C
		timeUTC := time.Now().UTC()
		timeStrNMEA := timeUTC.Format("150405.00")
		dateStrNMEA := timeUTC.Format("020106")

		// Update position if movement is enabled
		if movementEnabled {
			elapsed := time.Since(lastUpdate)
			currentPos = calculateNewPosition(currentPos, movementSpeed, movementBearing, elapsed)
			lastUpdate = time.Now()
		}

		latNMEA, ns := toNMEALat(currentPos.Latitude)
		lonNMEA, ew := toNMEALon(currentPos.Longitude)

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

		// Enhanced GPGSV sentences with realistic PRNs
		numSVInView := numSatellites + rand.Intn(5)
		if numSVInView < numSatellites {
			numSVInView = numSatellites
		}
		if numSVInView > 20 {
			numSVInView = 20
		}

		// Select appropriate PRNs based on talker ID
		var availablePRNs []int
		switch strings.ToUpper(talkerID) {
		case "GN":
			// Mix GPS and GLONASS PRNs
			availablePRNs = append(gpsPRNs, glonassPRNs...)
		case "GP":
			availablePRNs = gpsPRNs
		case "GL":
			availablePRNs = glonassPRNs
		default:
			availablePRNs = gpsPRNs
		}

		// Shuffle and select PRNs
		rand.Shuffle(len(availablePRNs), func(i, j int) {
			availablePRNs[i], availablePRNs[j] = availablePRNs[j], availablePRNs[i]
		})
		selectedPRNs := availablePRNs[:numSVInView]

		numGSVMsgs := (numSVInView + 3) / 4
		for i := 0; i < numGSVMsgs; i++ {
			msgNum := i + 1
			gsvBody := fmt.Sprintf("%sGSV,%d,%d,%02d", talkerID, numGSVMsgs, msgNum, numSVInView)

			for sv := 0; sv < 4; sv++ {
				svIndex := i*4 + sv
				if svIndex < numSVInView {
					prn := selectedPRNs[svIndex]
					elevation := rand.Intn(90)
					azimuth := rand.Intn(360)
					snr := rand.Intn(50) + 20
					gsvBody += fmt.Sprintf(",%02d,%02d,%03d,%02d", prn, elevation, azimuth, snr)
				} else {
					break
				}
			}
			sentences = append(sentences, addChecksumAndCRLF(gsvBody))
		}

		// Write sentences with retry logic
		for _, sentence := range sentences {
			var err error
			for retry := 0; retry < maxRetries; retry++ {
				_, err = writer.Write([]byte(sentence))
				if err == nil {
					break
				}
				log.Printf("Error writing NMEA sentence (attempt %d/%d): %v", retry+1, maxRetries, err)
				time.Sleep(retryDelay)
			}
			if err != nil {
				log.Printf("Failed to write NMEA sentence after %d attempts: %v", maxRetries, err)
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
	flag.BoolVar(&movementEnabled, "move", false, "Enable position movement simulation")
	flag.Float64Var(&movementSpeed, "speed", 0.0001, "Movement speed in degrees per second")
	flag.Float64Var(&movementBearing, "bearing", 0, "Movement bearing in degrees from north (0-360)")

	if runtime.GOOS == "windows" {
		flag.StringVar(&comPortName, "com", "", "COM port to use for simulation (e.g., COM3)")
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
		if !strings.HasPrefix(strings.ToUpper(comPortName), "COM") {
			log.Fatal("Invalid COM port name. Must start with 'COM' (e.g., COM3)")
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
