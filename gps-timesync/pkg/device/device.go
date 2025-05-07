package device

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Sudo-Ivan/gps-timesync/pkg/gps"
	"github.com/Sudo-Ivan/gps-timesync/pkg/system"
)

// Error definitions for the device package.
var (
	ErrNoGPSDevices = errors.New("no GPS devices found")
)

// FindGPSDevices searches for potential GPS devices on the system.
// It scans for USB devices, ACM devices, and serial ports based on the operating system.
func FindGPSDevices(debug bool) ([]string, error) {
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
					if IsPotentialGPSDevice(device) {
						devices = append(devices, device)
					}
				}
			}
		}
	default:
		return nil, fmt.Errorf("%w: %s", system.ErrUnsupportedOS, runtime.GOOS)
	}

	if len(devices) == 0 {
		return nil, ErrNoGPSDevices
	}
	return devices, nil
}

// IsPotentialGPSDevice checks if a device might be a GPS device by checking its properties.
// On Windows, it only verifies device existence. On Unix-like systems, it checks device properties.
func IsPotentialGPSDevice(device string) bool {
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
		// #nosec G204 -- device path is from Glob or validated input
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

// MonitorDevices continuously watches for new GPS devices being plugged in.
// It polls the system periodically to detect new devices.
func MonitorDevices(interval int, debug bool) error {
	fmt.Printf("Monitoring for new GPS devices (polling every %d seconds)...\n", interval)
	fmt.Println("Press Ctrl+C to stop")

	// Keep track of previously seen devices
	seenDevices := make(map[string]bool)

	for {
		select {
		case <-time.After(time.Duration(interval) * time.Second):
			devices, err := FindGPSDevices(debug)
			if err != nil && !errors.Is(err, ErrNoGPSDevices) {
				return fmt.Errorf("error finding devices: %v", err)
			}

			// Check for new devices
			for _, device := range devices {
				if !seenDevices[device] {
					fmt.Printf("\nNew GPS device detected: %s\n", device)
					seenDevices[device] = true

					// Test if it's actually a GPS device
					gpsInstance := gps.NewGPSTimeSync(device, 9600, debug)
					isGPS, err := gpsInstance.IsGPSDevice(device)
					// Ensure context is canceled for this temporary instance if not used further
					// However, IsGPSDevice is synchronous and short-lived for this check.
					// If IsGPSDevice became asynchronous or long-running, proper context management for gpsInstance would be crucial.
					gpsInstance.Cancel() // Cancel the context for the temporary instance

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
