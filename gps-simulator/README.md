# GPS Simulator

This program simulates a GPS device by emitting NMEA sentences ($GPRMC and $GPGGA) over a serial interface. It can be used to test the `gps-timesync` application without requiring a physical GPS device.

The simulator works on both Linux and Windows.

## Prerequisites

### Windows
You will need to create a virtual serial port pair. Tools like `com0com` (freely available) can be used for this.
1.  Install `com0com` or a similar tool.
2.  Create a virtual pair, for example, `COM3` <-> `COM4`. The simulator will write to one of these (e.g., `COM3`), and the `gps-timesync` application will read from the other (e.g., `COM4`).

## Building the Simulator

1.  Navigate to the simulator's directory:
    ```bash
    cd gps-simulator
    ```
2.  Build the simulator:
    ```bash
    go build
    ```
    This will create an executable named `gps-simulator` (or `gps-simulator.exe` on Windows).

## Running the Simulator

### Option 1: Using the built executable

When running the simulator, you can specify the NMEA talker ID and the number of satellites for GPGGA sentences:
*   `-talker <ID>`: Sets the NMEA Talker ID (e.g., `GP`, `GN`, `GL`, `GA`). Defaults to `GP`.
*   `-sats <num>`: Sets the number of satellites reported as in use in the `$__GGA` sentence (0-12). Defaults to `7`.

#### Linux
1.  Run the simulator:
    ```bash
    ./gps-simulator
    ```
    The simulator will output the name of the pseudo-terminal (PTY) it has created, for example:
    ```
    INFO simulator.go:XX: PTY created. Slave device: /dev/pts/X
    INFO simulator.go:XX: Run gps-timesync with: -d /dev/pts/X
    ```
    Take note of this device path (e.g., `/dev/pts/X`).

Example with flags:
    ```bash
    ./gps-simulator -talker GN -sats 10
    ```

#### Windows
1.  Run the simulator, specifying one of the COM ports from your virtual pair, and optionally the talker ID and satellite count:
    ```bash
    ./gps-simulator.exe -com COM3 -talker GN -sats 10
    ```
    (Replace `COM3` with the actual COM port name you intend for the simulator to use).
    The simulator will log that it's using this port.

Example with flags:
    ```bash
    go run simulator.go -talker GN -sats 10
    ```

#### Windows
    ```bash
    go run simulator.go -com COM3 -talker GN -sats 10
    ```
    (Replace `COM3` with your chosen COM port).

### Option 2: Using `go run` (without building first)

You can also run the simulator directly using `go run`:

1.  Navigate to the simulator's directory:
    ```bash
    cd gps-simulator
    ```
#### Linux
    ```bash
    go run simulator.go
    ```
    Note the PTY device path as mentioned above.
#### Windows
    ```bash
    go run simulator.go -com COM3
    ```
    (Replace `COM3` with your chosen COM port).


## Connecting `gps-timesync` to the Simulator

Once the simulator is running:

1.  Navigate to the directory of the main `gps-timesync` application (usually the project root).
    ```bash
    cd .. 
    ```
    (If you are in `gps-simulator`, `cd ..` goes to the `gps-timesync` root).

2.  Run `gps-timesync`, pointing it to the correct device:

    #### Linux
    Use the PTY device path noted from the simulator's output:
    ```bash
    sudo ./gps-timesync -d /dev/pts/X 
    ```
    (Replace `/dev/pts/X` with the actual path).

    #### Windows
    Use the *other* COM port from your virtual pair. For example, if the simulator is using `COM3`, and `COM4` is the other end of the pair, run:
    ```bash
    ./gps-timesync.exe -d COM4
    ```
    *Note: The `gps-timesync` application requires administrative privileges to set the system time. On Linux, this is typically `sudo`. On Windows, you may need to run it from an Administrator terminal.*

The `gps-timesync` application should now detect the simulator as a GPS device and you can use its features (e.g., sync time, monitor data). 