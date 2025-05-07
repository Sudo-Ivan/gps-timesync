# GPS Time Sync

A lightweight GPS time synchronization tool for Linux, BSD, and Windows systems. This tool synchronizes your system time with GPS time using a GPS module or USB dongle.

[![DeepSource](https://app.deepsource.com/gh/Sudo-Ivan/gps-timesync.svg/?label=active+issues&show_trend=true&token=2L1lQ7ldwu37DSctPOqw-s5v)](https://app.deepsource.com/gh/Sudo-Ivan/gps-timesync/)
[![Socket](https://socket.dev/api/badge/go/package/github.com/sudo-ivan/gps-timesync?version=v1.0.0)](https://socket.dev/api/badge/go/package/github.com/sudo-ivan/gps-timesync?version=v1.0.0)

## Features

- Zero dependencies for core functionality, simulator only uses `github.com/creack/pty`
- Supports NMEA GPRMC sentences
- Automatic serial port configuration
- Interactive device detection and selection
- Real-time GPS data monitoring
- Satellite information display
- Device hot-plug monitoring
- Cross-platform support (Linux, BSD, Windows)
- Simple and efficient implementation

## Requirements

- Go 1.24+ 
- GPS module or USB dongle

## Installation

```bash
go install github.com/Sudo-Ivan/gps-timesync@latest
```

or

```bash
git clone https://github.com/Sudo-Ivan/gps-timesync
cd gps-timesync
go build
```

## Usage

### Basic Usage

1. Connect your GPS module or USB dongle
2. Run the program with sudo/root privileges:

```bash
sudo gps-timesync
```

or on Windows (PowerShell as Administrator):

```powershell
gps-timesync.exe
```

### Command Line Options

```bash
# Specify device and baud rate
sudo gps-timesync -d /dev/ttyUSB0 -b 9600

# Enable debug mode
sudo gps-timesync -db

# Monitor for new GPS devices
sudo gps-timesync -m

# Monitor with custom polling interval
sudo gps-timesync -m --interval 2

# Full options list
sudo gps-timesync --help
```

Available options:
- `-d, --device`: Specify GPS device path (e.g., /dev/ttyUSB0 or COM1)
- `-b, --baud`: Specify baud rate (default: 9600)
- `-db, --debug`: Enable debug mode
- `-m, --monitor`: Monitor for new GPS devices
- `--interval`: Polling interval in seconds for monitor mode (default: 5)
- `-nr, --no-root`: Bypass root/sudo check (use with caution, time sync will likely fail)

### GPS Simulator

A GPS simulator is included in the `gps-simulator` directory. This can be used for testing `gps-timesync` without a physical GPS device. It creates a pseudo-terminal (on Linux) or uses a specified COM port (on Windows) to emit NMEA sentences.

For detailed instructions on building and running the simulator, please see the [gps-simulator/README.md](gps-simulator/README.md).

### Interactive Mode

The program will:
1. Automatically detect potential GPS devices
2. Allow you to select the correct device
3. Verify if the selected device is a GPS device
4. Present an interactive menu with options:
   - Sync system time
   - Monitor GPS data (shows time, position, and satellite information)
   - Exit

### Monitor Mode

When running with `-m` or `--monitor`:
1. Continuously watches for new GPS devices
2. Automatically detects when devices are plugged in or removed
3. Tests new devices to confirm they are GPS devices
4. Displays real-time status updates
5. Press Ctrl+C to stop monitoring

## How it Works

The program:
1. Scans for potential GPS devices in common locations:
   - USB devices (/dev/ttyUSB*)
   - ACM devices (/dev/ttyACM*)
   - Serial ports (/dev/ttyS*)
   - COM ports (Windows)
2. Tests each device for GPS functionality
3. Opens the selected GPS device
4. Configures the serial port
5. Reads NMEA sentences
6. Parses various NMEA sentences (GPRMC, GPGGA, GPGSV) for:
   - Time and date
   - Position (latitude/longitude)
   - Satellite information
7. Sets the system time when valid data is received

## License

MIT License 