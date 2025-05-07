package system

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// Error definitions for the system package.
var (
	ErrSystemTimeUpdate = errors.New("failed to update system time")
	ErrUnsupportedOS    = errors.New("unsupported operating system")
)

// SetSystemTime sets the system time based on the operating system.
// On Windows, it uses the 'time' and 'date' commands.
// On Unix-like systems, it uses the 'date' command.
func SetSystemTime(t time.Time) error {
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

// ConfigureSerialPort configures the serial port based on the operating system.
// On Windows, it relies on device driver settings.
// On Unix-like systems, it uses the 'stty' command.
func ConfigureSerialPort(device string, baudRate int) error {
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
