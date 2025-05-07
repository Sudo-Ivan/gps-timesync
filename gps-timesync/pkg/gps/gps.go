package gps

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Sudo-Ivan/gps-timesync/pkg/nmea"
	"github.com/Sudo-Ivan/gps-timesync/pkg/system"
)

// Common error definitions for the GPS package.
var (
	ErrInvalidDevice = errors.New("invalid or non-GPS device")
	ErrDeviceAccess  = errors.New("cannot access device")
	ErrNoValidData   = errors.New("no valid GPS data received")
)

// GPSTimeSync represents a GPS time synchronization instance.
// It manages the connection to a GPS device and provides methods for time synchronization
// and GPS data monitoring.
type GPSTimeSync struct {
	DevicePath string // Path to the GPS device
	BaudRate   int    // Baud rate for serial communication
	Debug      bool   // Enable debug logging
	Ctx        context.Context
	Cancel     context.CancelFunc
}

// NewGPSTimeSync creates a new GPS time synchronization instance.
// It initializes the context and sets up the device configuration.
func NewGPSTimeSync(devicePath string, baudRate int, debug bool) *GPSTimeSync {
	ctx, cancel := context.WithCancel(context.Background())
	return &GPSTimeSync{
		DevicePath: devicePath,
		BaudRate:   baudRate,
		Debug:      debug,
		Ctx:        ctx,
		Cancel:     cancel,
	}
}

// IsGPSDevice checks if a device is likely a GPS device by attempting to read NMEA sentences.
// It validates the device path and attempts to read from the device.
func (g *GPSTimeSync) IsGPSDevice(device string) (bool, error) {
	if runtime.GOOS == "windows" {
		if !strings.HasPrefix(device, "COM") {
			return false, fmt.Errorf("invalid device path: %s", device)
		}
	} else if !strings.HasPrefix(device, "/dev/") {
		return false, fmt.Errorf("invalid device path: %s", device)
	}

	// #nosec G304 - device path is validated before use
	file, err := os.OpenFile(device, os.O_RDWR, 0600)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrDeviceAccess, err)
	}
	defer file.Close()

	if err := system.ConfigureSerialPort(device, g.BaudRate); err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(file)
	timeout := time.After(2 * time.Second)

	for {
		select {
		case <-timeout:
			return false, ErrNoValidData
		case <-g.Ctx.Done():
			return false, g.Ctx.Err()
		default:
			if scanner.Scan() {
				line := scanner.Text()
				// Check for common NMEA talker IDs for GNSS data
				if strings.HasPrefix(line, "$GP") ||
					strings.HasPrefix(line, "$GN") ||
					strings.HasPrefix(line, "$GL") ||
					strings.HasPrefix(line, "$GA") {
					log.Printf("IsGPSDevice: Detected valid NMEA prefix in line: %s", line)
					return true, nil
				}
			}
			if err := scanner.Err(); err != nil {
				return false, fmt.Errorf("error reading device: %v", err)
			}
		}
	}
}

// SyncTime synchronizes system time with GPS time.
// It reads NMEA sentences from the GPS device and updates the system time
// when a valid GPRMC sentence is received.
func (g *GPSTimeSync) SyncTime() error {
	// #nosec G304 - device path is validated before use
	file, err := os.OpenFile(g.DevicePath, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeviceAccess, err)
	}
	defer file.Close()

	if err := system.ConfigureSerialPort(g.DevicePath, g.BaudRate); err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-timeout:
			return ErrNoValidData
		case <-g.Ctx.Done():
			return g.Ctx.Err()
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "$GPRMC") {
					fields := strings.Split(line, ",")
					if len(fields) < 10 || fields[2] != "A" {
						continue
					}

					gpsTime, err := nmea.ParseNMEATime(fields[1], fields[9])
					if err != nil {
						if g.Debug {
							log.Printf("Warning: Failed to parse NMEA time: %v", err)
						}
						continue
					}

					if err := system.SetSystemTime(gpsTime); err != nil {
						return err
					}

					log.Printf("Time synchronized successfully: %s", gpsTime.Format(time.RFC3339))
					return nil
				}
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading device: %v", err)
			}
		}
	}
}

// MonitorGPS continuously monitors GPS data from the device.
// It displays time, date, position, and satellite information
// from various NMEA sentences.
func (g *GPSTimeSync) MonitorGPS() error {
	// #nosec G304 - device path is validated before use
	file, err := os.OpenFile(g.DevicePath, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeviceAccess, err)
	}
	defer file.Close()

	if err := system.ConfigureSerialPort(g.DevicePath, g.BaudRate); err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	fmt.Println("Monitoring GPS data... (Press Ctrl+C to stop)")

	for {
		select {
		case <-g.Ctx.Done():
			return g.Ctx.Err()
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "$") { // Basic check for an NMEA sentence start
					fields := strings.Split(line, ",")
					if len(fields) == 0 {
						continue
					}
					sentenceType := fields[0]

					switch {
					// Check for message type directly, ignoring specific talker ID for monitoring
					case strings.HasSuffix(sentenceType, "RMC"):
						if len(fields) >= 10 && fields[2] == "A" { // Check for validity status 'A'
							fmt.Printf("Time: %s, Date: %s\n", fields[1], fields[9])
						}
					case strings.HasSuffix(sentenceType, "GGA"):
						if len(fields) >= 8 { // Basic check for enough fields
							fmt.Printf("Latitude: %s%s, Longitude: %s%s, Satellites: %s\n",
								fields[2], fields[3], fields[4], fields[5], fields[7])
						}
					case strings.HasSuffix(sentenceType, "GSV"):
						if len(fields) >= 4 { // Basic check for enough fields
							fmt.Printf("Satellites in view data: %s\n", line)
						}
					}
				}
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading device: %v", err)
			}
		}
	}
}
