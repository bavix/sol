# sol

sol is a service that listens for Wake-on-LAN magic packets and shuts down the system when received.

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
