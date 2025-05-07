# GPS Simulator

This program simulates a GPS device by emitting NMEA sentences ($GPRMC and $GPGGA) over a serial interface. It can be used to test the `gps-timesync` application without requiring a physical GPS device.

The simulator works on both Linux and Windows and includes features like position movement simulation and realistic satellite constellation simulation.

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

### Command Line Options

The simulator supports several command line options:

*   `-talker <ID>`: Sets the NMEA Talker ID (e.g., `GP`, `GN`, `GL`, `GA`). Defaults to `GP`.
*   `-sats <num>`: Sets the number of satellites reported as in use in the `$__GGA` sentence (0-12). Defaults to `7`.
*   `-move`: Enable position movement simulation. Defaults to false.
*   `-speed <value>`: Movement speed in degrees per second. Defaults to 0.0001.
*   `-bearing <degrees>`: Movement bearing in degrees from north (0-360). Defaults to 0.

### Examples

#### Basic Usage
```bash
./gps-simulator
```

#### With Movement Simulation
```bash
./gps-simulator -move -speed 0.0002 -bearing 45
```
This will simulate movement at 0.0002 degrees per second in a northeast direction (45 degrees).

#### With GNSS Constellation
```bash
./gps-simulator -talker GN -sats 10
```
This will simulate a mixed GPS/GLONASS constellation with 10 satellites in view.

### Platform-Specific Usage

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

#### Windows
1.  Run the simulator, specifying one of the COM ports from your virtual pair:
    ```bash
    ./gps-simulator.exe -com COM3
    ```
    (Replace `COM3` with the actual COM port name you intend for the simulator to use).

## Features

### Position Movement Simulation
The simulator can simulate movement by calculating new positions based on speed and bearing. This is useful for testing applications that need to track position changes.

### Realistic Satellite Simulation
* GPS PRNs (1-32) and GLONASS PRNs (65-96) are supported
* Mixed constellation simulation when using the "GN" talker ID
* Realistic satellite visibility patterns
* Configurable number of satellites in view

### Error Handling
* Automatic retry on write failures
* Graceful handling of port disconnections
* Detailed error logging

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