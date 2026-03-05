# Wake-on-LAN Controller

A web-based Wake-on-LAN (WOL) application written in Go. It provides a beautiful, modern (glassmorphism) interface to manage and remotely power on your local network devices. 

## Features

- **Web-Based Management**: Easily add, edit, View, and delete host entries using a responsive grid UI.
- **One-Click Wake**: Send Magic Packets (WOL) instantly to wake up your configured devices.
- **Live Monitoring**: Real-time visual ping status to check if a host is currently online.
- **Quick Access URLs**: Provides clickable device names that link directly to a configurable access URL (e.g., your NAS interface or home server dashboard).
- **Master Monitoring Toggle**: Conveniently enable or disable the ping monitoring status for all your devices at once.
- **Data Persistence**: Stores your host configurations persistently in a simple `hosts.json` file.
- **Docker Ready**: Fully containerized and ready to deploy via Docker and Docker Compose.

## Tech Stack

- **Backend**: Go (Golang) 1.22+
- **Frontend**: HTML5, Vanilla JavaScript, CSS (Inter font, smooth gradients, and glassmorphic elements)
- **Deployment**: Docker, Alpine Linux

## Getting Started

### Prerequisites

To run this application, you will need either:
- **Docker** and **Docker Compose**
- **Go 1.22+** (if running locally without Docker)

### Running via Docker Compose (Recommended)

This is the easiest way to deploy the application and ensures that your `hosts.json` file persists appropriately.

1. Clone the repository:
   ```bash
   git clone <repo-url>
   cd wakeonlan
   ```
2. Create a `docker-compose.yml` file with the following content:
   ```yaml
   services:
     wakeonlan:
       build: .
       container_name: wakeonlan
       # Not using host networking; instead exposing port 8080 on localhost to the host machine.
       # Note: On Docker Desktop for Mac/Windows, UDP broadcasts may still not reach the physical LAN.
       ports:
         - "127.0.0.1:8080:8080"
       environment:
         - PORT=8080
         - HOSTS_FILE=/data/hosts.json
       volumes:
         - ./hosts.json:/data/hosts.json
       restart: unless-stopped
   ```
3. Start the container:
   ```bash
   docker compose up -d
   ```
4. Access the web interface at: `http://localhost:8080`

*Note: For macOS or Windows using Docker Desktop, broadcast packets might not reach the physical network depending on network constraints. Host networking might be required on Linux. We've mapped port `8080` by default.*

### Running Locally (Development)

To run the application directly using Go without Docker:

1. Clone the repository and navigate inside:
   ```bash
   git clone <repo-url>
   cd wakeonlan
   ```
2. Run the application:
   ```bash
   go run main.go
   ```
3. Access the web interface at: `http://localhost:8080`

## Configuration

You can configure the application using environment variables:

- `PORT`: Sets the port for the web server (Default: `8080`).
- `HOSTS_FILE`: Absolute or relative path to the persistent JSON file storing the hosts (Default: `hosts.json` locally or `/data/hosts.json` inside the Docker container).

## Managing Hosts

When you click the **+ (Add Host)** button in the UI, you can provide the following information:

- **Name** (Required): Label to identify your device.
- **MAC Address** (Required): The MAC address of the device to wake up (e.g., `00:11:22:33:44:55`).
- **Broadcast IP** (Optional): Provide your specific network broadcast IP instead of the default global broadcast (`255.255.255.255`).
- **Host IP** (Optional): Highly recommended for live monitoring. Used by the ping feature to track the device's online status.
- **Access URL** (Optional): Transform your device's name into a clickable link to an admin panel or specific service URL.
- **Enable Monitoring**: Toggle ping monitoring locally for each device.

## API Endpoints

The Go backend serves a simple REST API on `/api/hosts`.

- `GET /api/hosts` - Retrieve all host records.
- `POST /api/hosts` - Create a new host record.
- `PUT /api/hosts/:id` - Edit an existing host record.
- `DELETE /api/hosts/:id` - Delete a given host record.
- `POST /api/hosts/:id/wake` - Send a Wake-on-LAN Magic Packet to the host.
- `GET /api/hosts/:id/ping` - Ping the particular host.

## License

This project is free to use and modify.
