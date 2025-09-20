# SoL - Shutdown on LAN

SoL is a service that listens for Wake-on-LAN magic packets and shuts down the system when received.

## Inspiration

This project was inspired by the article ["Выключаем компьютер через Wake-on-Lan"](https://habr.com/ru/articles/816765/) on Habr, which demonstrates how to repurpose Wake-on-LAN packets for shutdown functionality instead of wake-up.

## Description of the service

The sol service is built using [cobra](https://github.com/spf13/cobra) CLI framework and Go standard library.

### Listen
The listen command listens for Wake-on-LAN magic packets on the specified network interface and shuts down the system when a matching packet is received.

### Protocol
The service listens for Wake-on-LAN magic packets on UDP ports (commonly 7 or 9). When a magic packet containing the target MAC address is received, the system is shut down.

## Run the service

```bash
sol listen --iface eth0
```

### Options

- `--iface`: Network interface name to bind to (required)
- `--port`: UDP port to listen on (default: 9)
- `--dry-run`: Log when a matching packet is received instead of shutting down

## Installation

### Download and Install

1. **Download the archive**

   Download the latest release:

   ```bash
   wget https://github.com/bavix/sol/releases/download/v0.0.2/sol-v0.0.2-linux-amd64.tar.gz
   ```

   or using `curl`:

   ```bash
   curl -L -o sol-v0.0.2-linux-amd64.tar.gz https://github.com/bavix/sol/releases/download/v0.0.2/sol-v0.0.2-linux-amd64.tar.gz
   ```

2. **Extract the archive**

   Navigate to the download directory and extract:

   ```bash
   tar -xzvf sol-v0.0.2-linux-amd64.tar.gz
   ```

   Options:
   * `-x` — extract
   * `-z` — use gzip
   * `-v` — show progress
   * `-f` — file

3. **Navigate to the extracted directory**

   ```bash
   cd sol-v0.0.2-linux-amd64
   ```

4. **Make the binary executable**

   ```bash
   chmod +x sol
   ```

5. **Move binary to PATH directory**

   To run `sol` from anywhere, move it to a directory in PATH:

   ```bash
   sudo mv sol /usr/local/bin/sol
   ```

6. **Verify installation**

   ```bash
   sol --help
   ```

### Systemd Service Setup

To run SoL as a systemd service that starts automatically:

1. **Create systemd unit file**

   ```bash
   sudo nano /etc/systemd/system/sol.service
   ```

   Add the following content:

   ```ini
   [Unit]
   Description=SOL listener
   After=network-online.target
   Wants=network-online.target

   [Service]
   ExecStart=/usr/local/bin/sol listen --iface enp2s0 --port 9
   Restart=always
   RestartSec=5
   User=root

   [Install]
   WantedBy=multi-user.target
   ```

   **Important**: Replace `enp2s0` with your actual network interface name.

2. **Reload systemd configuration**

   ```bash
   sudo systemctl daemon-reload
   ```

3. **Enable autostart**

   ```bash
   sudo systemctl enable sol.service
   ```

4. **Start the service**

   ```bash
   sudo systemctl start sol.service
   ```

5. **Check service status**

   ```bash
   sudo systemctl status sol.service
   ```

6. **View logs**

   ```bash
   journalctl -u sol.service -f
   ```

### Service Configuration Notes

- `After=network-online.target` ensures the service starts after the network is fully online
- `Restart=always` automatically restarts the service if it crashes
- For production use, remove `--dry-run` flag from the ExecStart command
- Consider creating a dedicated user instead of running as root for better security
