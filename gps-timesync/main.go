// Package main implements a GPS time synchronization tool for Linux, BSD, and Windows systems.
// It provides functionality to synchronize system time with GPS time and monitor GPS data.
//
// The package supports:
//   - Automatic GPS device detection
//   - NMEA sentence parsing
//   - System time synchronization
//   - Real-time GPS data monitoring
//   - Cross-platform support (Linux, BSD, Windows)
//
// Example usage:
//
//	sudo ./gps-timesync -d /dev/ttyUSB0 -b 9600
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Sudo-Ivan/gps-timesync/pkg/device"
	"github.com/Sudo-Ivan/gps-timesync/pkg/gps"
)

// Common error definitions for the package.
var (
	ErrNoGPSDevices     = errors.New("no GPS devices found")
	ErrInvalidDevice    = errors.New("invalid or non-GPS device")
	ErrDeviceAccess     = errors.New("cannot access device")
	ErrNoValidData      = errors.New("no valid GPS data received")
	ErrInvalidNMEAData  = errors.New("invalid NMEA data")
	ErrSystemTimeUpdate = errors.New("failed to update system time")
	ErrUnsupportedOS    = errors.New("unsupported operating system")
)

// GPSTimeSync represents a GPS time synchronization instance.
// It manages the connection to a GPS device and provides methods for time synchronization
// and GPS data monitoring.
type GPSTimeSync struct {
	devicePath string // Path to the GPS device
	baudRate   int    // Baud rate for serial communication
	debug      bool   // Enable debug logging
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewGPSTimeSync creates a new GPS time synchronization instance.
// It initializes the context and sets up the device configuration.
//
// Parameters:
//   - devicePath: Path to the GPS device (e.g., "/dev/ttyUSB0" or "COM1")
//   - baudRate: Baud rate for serial communication (default: 9600)
//   - debug: Enable debug logging
//
// Returns a new GPSTimeSync instance.
func NewGPSTimeSync(devicePath string, baudRate int, debug bool) *GPSTimeSync {
	ctx, cancel := context.WithCancel(context.Background())
	return &GPSTimeSync{
		devicePath: devicePath,
		baudRate:   baudRate,
		debug:      debug,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// findGPSDevices searches for potential GPS devices on the system.
// It scans for USB devices, ACM devices, and serial ports based on the operating system.
//
// Parameters:
//   - debug: Enable debug logging
//
// Returns a list of potential GPS device paths and any error encountered.
func findGPSDevices(debug bool) ([]string, error) {
	var devices []string

	switch runtime.GOOS {
	case "windows":
		// On Windows, look for COM ports
		for i := 1; i <= 16; i++ {
			device := fmt.Sprintf("COM%d", i)
			if _, err := os.Stat(device); err == nil {
				devices = append(devices, device)
			}
		}
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		// First try USB devices
		usbDevices, err := filepath.Glob("/dev/ttyUSB*")
		if err == nil {
			devices = append(devices, usbDevices...)
		}

		// Then try ACM devices (common for USB GPS dongles)
		acmDevices, err := filepath.Glob("/dev/ttyACM*")
		if err == nil {
			devices = append(devices, acmDevices...)
		}

		// If no USB devices found, try serial ports
		if len(devices) == 0 {
			serialDevices, err := filepath.Glob("/dev/ttyS*")
			if err == nil {
				// Filter out non-GPS serial ports
				for _, device := range serialDevices {
					// Check if device is a potential GPS device
					if isPotentialGPSDevice(device) {
						devices = append(devices, device)
					}
				}
			}
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
	}

	if len(devices) == 0 {
		return nil, ErrNoGPSDevices
	}
	return devices, nil
}

// isPotentialGPSDevice checks if a device might be a GPS device by checking its properties.
// On Windows, it only verifies device existence. On Unix-like systems, it checks device properties.
//
// Parameters:
//   - device: Path to the device to check
//
// Returns true if the device might be a GPS device, false otherwise.
func isPotentialGPSDevice(device string) bool {
	// Check if device exists and is readable
	if _, err := os.Stat(device); err != nil {
		return false
	}

	switch runtime.GOOS {
	case "windows":
		// On Windows, we can't easily check device properties
		// Just verify the device exists and is accessible
		return true
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		// Try to get device information
		cmd := exec.Command("udevadm", "info", "--query=property", device)
		output, err := cmd.Output()
		if err != nil {
			return false
		}

		// Check for GPS-related properties
		properties := string(output)
		return strings.Contains(properties, "ID_MODEL=gps") ||
			strings.Contains(properties, "ID_MODEL=GPS") ||
			strings.Contains(properties, "ID_VENDOR=gps") ||
			strings.Contains(properties, "ID_VENDOR=GPS")
	default:
		return false
	}
}

// setSystemTime sets the system time based on the operating system.
// On Windows, it uses the 'time' and 'date' commands.
// On Unix-like systems, it uses the 'date' command.
//
// Parameters:
//   - t: The time to set
//
// Returns any error encountered during the operation.
func setSystemTime(t time.Time) error {
	switch runtime.GOOS {
	case "windows":
		// On Windows, use the 'time' command
		timeStr := t.Format("15:04:05")
		// #nosec G204 - timeStr is generated from time.Time, not user input
		cmd := exec.Command("time", timeStr)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%w: %v", ErrSystemTimeUpdate, err)
		}

		// Set date using 'date' command
		dateStr := t.Format("01/02/2006")
		// #nosec G204 - dateStr is generated from time.Time, not user input
		cmd = exec.Command("date", dateStr)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%w: %v", ErrSystemTimeUpdate, err)
		}
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		// On Unix-like systems, use the 'date' command
		timeStr := t.Format("2006-01-02 15:04:05")
		// #nosec G204 - timeStr is generated from time.Time, not user input
		cmd := exec.Command("date", "-s", timeStr)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%w: %v", ErrSystemTimeUpdate, err)
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
	}
	return nil
}

// configureSerialPort configures the serial port based on the operating system.
// On Windows, it relies on device driver settings.
// On Unix-like systems, it uses the 'stty' command.
//
// Parameters:
//   - device: Path to the serial device
//   - baudRate: Baud rate for serial communication
//
// Returns any error encountered during the operation.
func configureSerialPort(device string, baudRate int) error {
	switch runtime.GOOS {
	case "windows":
		// On Windows, we can't easily configure the serial port
		// The port settings are typically managed by the device driver
		return nil
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		// #nosec G204 - device and baudRate are validated before use
		cmd := exec.Command("stty", "-F", device, fmt.Sprintf("%d", baudRate), "raw", "-echo")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to configure serial port: %v", err)
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
	}
	return nil
}

// isGPSDevice checks if a device is likely a GPS device by attempting to read NMEA sentences.
// It validates the device path and attempts to read from the device.
//
// Parameters:
//   - device: Path to the device to check
//
// Returns true if the device appears to be a GPS device, and any error encountered.
func (g *GPSTimeSync) isGPSDevice(device string) (bool, error) {
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

	if err := configureSerialPort(device, g.baudRate); err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(file)
	timeout := time.After(2 * time.Second)

	for {
		select {
		case <-timeout:
			return false, ErrNoValidData
		case <-g.ctx.Done():
			return false, g.ctx.Err()
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "$GP") {
					return true, nil
				}
			}
			if err := scanner.Err(); err != nil {
				return false, fmt.Errorf("error reading device: %v", err)
			}
		}
	}
}

// parseNMEATime parses time and date from NMEA sentence.
// It expects time in HHMMSS format and date in DDMMYY format.
//
// Parameters:
//   - timeStr: Time string in HHMMSS format
//   - dateStr: Date string in DDMMYY format
//
// Returns the parsed time and any error encountered.
func parseNMEATime(timeStr, dateStr string) (time.Time, error) {
	if len(timeStr) < 6 || len(dateStr) != 6 {
		return time.Time{}, ErrInvalidNMEAData
	}

	hour := timeStr[0:2]
	min := timeStr[2:4]
	sec := timeStr[4:6]
	day := dateStr[0:2]
	month := dateStr[2:4]
	year := "20" + dateStr[4:6]

	timeStr = fmt.Sprintf("%s-%s-%s %s:%s:%s", year, month, day, hour, min, sec)
	return time.Parse("2006-01-02 15:04:05", timeStr)
}

// syncTime synchronizes system time with GPS time.
// It reads NMEA sentences from the GPS device and updates the system time
// when a valid GPRMC sentence is received.
//
// Returns any error encountered during the operation.
func (g *GPSTimeSync) syncTime() error {
	// #nosec G304 - device path is validated before use
	file, err := os.OpenFile(g.devicePath, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeviceAccess, err)
	}
	defer file.Close()

	if err := configureSerialPort(g.devicePath, g.baudRate); err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-timeout:
			return ErrNoValidData
		case <-g.ctx.Done():
			return g.ctx.Err()
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "$GPRMC") {
					fields := strings.Split(line, ",")
					if len(fields) < 10 || fields[2] != "A" {
						continue
					}

					gpsTime, err := parseNMEATime(fields[1], fields[9])
					if err != nil {
						if g.debug {
							log.Printf("Warning: Failed to parse NMEA time: %v", err)
						}
						continue
					}

					if err := setSystemTime(gpsTime); err != nil {
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

// monitorGPS continuously monitors GPS data from the device.
// It displays time, date, position, and satellite information
// from various NMEA sentences.
//
// Returns any error encountered during the operation.
func (g *GPSTimeSync) monitorGPS() error {
	// #nosec G304 - device path is validated before use
	file, err := os.OpenFile(g.devicePath, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeviceAccess, err)
	}
	defer file.Close()

	if err := configureSerialPort(g.devicePath, g.baudRate); err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	fmt.Println("Monitoring GPS data... (Press Ctrl+C to stop)")

	for {
		select {
		case <-g.ctx.Done():
			return g.ctx.Err()
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "$GP") {
					fields := strings.Split(line, ",")

					switch {
					case strings.HasPrefix(line, "$GPRMC"):
						if len(fields) >= 10 && fields[2] == "A" {
							fmt.Printf("Time: %s, Date: %s\n", fields[1], fields[9])
						}
					case strings.HasPrefix(line, "$GPGGA"):
						if len(fields) >= 15 {
							fmt.Printf("Latitude: %s%s, Longitude: %s%s, Satellites: %s\n",
								fields[2], fields[3], fields[4], fields[5], fields[7])
						}
					case strings.HasPrefix(line, "$GPGSV"):
						if len(fields) >= 4 {
							fmt.Printf("Satellites in view: %s\n", fields[3])
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

// monitorDevices continuously watches for new GPS devices being plugged in.
// It polls the system periodically to detect new devices.
//
// Parameters:
//   - interval: Polling interval in seconds
//   - debug: Enable debug logging
//
// Returns any error encountered during the operation.
func monitorDevices(interval int, debug bool) error {
	fmt.Printf("Monitoring for new GPS devices (polling every %d seconds)...\n", interval)
	fmt.Println("Press Ctrl+C to stop")

	// Keep track of previously seen devices
	seenDevices := make(map[string]bool)

	for {
		select {
		case <-time.After(time.Duration(interval) * time.Second):
			devices, err := findGPSDevices(debug)
			if err != nil && !errors.Is(err, ErrNoGPSDevices) {
				return fmt.Errorf("error finding devices: %v", err)
			}

			// Check for new devices
			for _, device := range devices {
				if !seenDevices[device] {
					fmt.Printf("\nNew GPS device detected: %s\n", device)
					seenDevices[device] = true

					// Test if it's actually a GPS device
					gps := NewGPSTimeSync(device, 9600, debug)
					isGPS, err := gps.isGPSDevice(device)
					if err != nil {
						if debug {
							fmt.Printf("Error testing device %s: %v\n", device, err)
						}
						continue
					}
					if isGPS {
						fmt.Printf("Confirmed %s is a GPS device\n", device)
					} else {
						fmt.Printf("%s does not appear to be a GPS device\n", device)
					}
				}
			}

			// Check for removed devices
			for device := range seenDevices {
				if _, err := os.Stat(device); os.IsNotExist(err) {
					fmt.Printf("\nDevice removed: %s\n", device)
					delete(seenDevices, device)
				}
			}
		}
	}
}

// main is the entry point of the program.
// It handles command-line arguments, device selection, and the main menu.
func main() {
	// Parse command line flags
	deviceFlag := flag.String("device", "", "Specify GPS device path (e.g., /dev/ttyUSB0 or COM1)")
	baudFlag := flag.Int("baud", 9600, "Specify baud rate (default: 9600)")
	debugFlag := flag.Bool("debug", false, "Enable debug mode")
	monitorFlag := flag.Bool("monitor", false, "Monitor for new GPS devices")
	monitorShortFlag := flag.Bool("m", false, "Short flag for -monitor")
	intervalFlag := flag.Int("interval", 5, "Polling interval in seconds for monitor mode (default: 5)")
	noRootFlag := flag.Bool("no-root", false, "Bypass root/sudo check (use with caution)")

	// Add short flags
	flag.StringVar(deviceFlag, "d", "", "Short flag for -device")
	flag.IntVar(baudFlag, "b", 9600, "Short flag for -baud")
	flag.BoolVar(debugFlag, "db", false, "Short flag for -debug")
	flag.BoolVar(noRootFlag, "nr", false, "Short flag for --no-root")

	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Check for root privileges on Unix systems
	if runtime.GOOS != "windows" {
		if os.Geteuid() != 0 { // Not running as root
			if *noRootFlag {
				log.Println("Warning: Running without root privileges due to --no-root flag.")
				log.Println("Time synchronization and some device configurations may fail.")
				log.Println("Ensure the user has necessary permissions for the specified device and time setting if not running as root.")
			} else {
				log.Fatal("This program must be run as root/sudo to ensure full functionality. Use --no-root or -nr to bypass this check if you understand the implications (e.g., for monitoring only, or if permissions are already set for your user).")
			}
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Handle monitor mode
	if *monitorFlag || *monitorShortFlag {
		if err := device.MonitorDevices(*intervalFlag, *debugFlag); err != nil {
			log.Fatalf("Error in monitor mode: %v", err)
		}
		return
	}

	var selectedDevice string

	// If device is specified via command line, use it
	if *deviceFlag != "" {
		selectedDevice = *deviceFlag
		gpsInstance := gps.NewGPSTimeSync(selectedDevice, *baudFlag, *debugFlag)
		defer gpsInstance.Cancel() // Defer cancel after gpsInstance is created
		isGPS, err := gpsInstance.IsGPSDevice(selectedDevice)
		if err != nil {
			log.Fatalf("Error testing device: %v", err)
		}
		if !isGPS {
			log.Fatalf("Specified device %s does not appear to be a GPS device", selectedDevice)
		}
		fmt.Printf("Using specified device: %s\n", selectedDevice)
	} else {
		// Otherwise, search for devices
		devices, err := device.FindGPSDevices(*debugFlag)
		if err != nil {
			log.Fatalf("Error finding GPS devices: %v", err)
		}

		fmt.Println("Found potential GPS devices:")
		for i, d := range devices {
			fmt.Printf("%d. %s\n", i+1, d)
		}

		for {
			fmt.Print("Select a device number (or 'q' to quit): ")
			var input string
			if _, err := fmt.Scanln(&input); err != nil {
				fmt.Println("Error reading input:", err)
				continue
			}

			if input == "q" {
				os.Exit(0)
			}

			var index int
			if _, err := fmt.Sscanf(input, "%d", &index); err != nil || index < 1 || index > len(devices) {
				fmt.Println("Invalid selection. Please try again.")
				continue
			}

			selectedDevice = devices[index-1]
			fmt.Printf("Testing device %s...\n", selectedDevice)

			gpsInstance := gps.NewGPSTimeSync(selectedDevice, *baudFlag, *debugFlag)
			defer gpsInstance.Cancel() // Defer cancel for this scope as well
			isGPS, err := gpsInstance.IsGPSDevice(selectedDevice)
			if err != nil {
				fmt.Printf("Error testing device: %v\n", err)
				// gpsInstance.Cancel() // Not strictly needed here due to defer, but good for clarity if loop continues differently
				continue
			}
			if isGPS {
				fmt.Printf("Confirmed %s is a GPS device.\n", selectedDevice)
				// gpsInstance.Cancel() // Not strictly needed here due to defer
				break
			} else {
				fmt.Printf("%s does not appear to be a GPS device. Please select another device.\n", selectedDevice)
				// gpsInstance.Cancel() // Not strictly needed here due to defer
			}
		}
	}

	// Create the main gpsInstance only after a device has been successfully selected and confirmed.
	// The defer gps.Cancel() from the previous block might be on a different gpsInstance if we re-scoped.
	// It's safer to create the one true gpsInstance here that the rest of the app uses.
	// However, the current logic reuses selectedDevice and re-initializes gpsInstance outside the loop, which is fine.

	gpsInstance := gps.NewGPSTimeSync(selectedDevice, *baudFlag, *debugFlag)
	defer gpsInstance.Cancel() // This is the main cancel for the application's gpsInstance

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal. Shutting down...")
		gpsInstance.Cancel()
	}()

	for {
		fmt.Println("\nGPS Time Sync Menu:")
		fmt.Println("1. Sync system time")
		fmt.Println("2. Monitor GPS data")
		fmt.Println("3. Exit")
		fmt.Print("Select an option: ")

		var choice int
		if _, err := fmt.Scanln(&choice); err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		switch choice {
		case 1:
			if err := gpsInstance.SyncTime(); err != nil {
				log.Printf("Failed to sync time: %v", err)
			}
		case 2:
			if err := gpsInstance.MonitorGPS(); err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Printf("Failed to monitor GPS: %v", err)
				}
			}
		case 3:
			return
		default:
			fmt.Println("Invalid option. Please try again.")
		}
	}
}
