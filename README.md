# flowercare-json-exporter

A server that reads data from Xiaomi MiFlora / HHCC Flower Care devices using Bluetooth and exposes it as JSON.
It was originally written by xperimental at [github.com/xperimental/flowercare-exporter](https://github.com/xperimental/flowercare-exporter), but then ultimately changed here to return JSON instead of Prometheus metrics.
Thanks a lot of the original project.

## Installation

First clone the repository, then run the following command to get a binary for your current operating system / architecture. This assumes a working Go installation with modules support (Go >= 1.11.0):

```bash
git clone https://github.com/marcelb/flowercare-json-exporter.git
cd flowercare-json-exporter
go build .
```

## Usage

```plain
$ flowercare-json-exporter -h
Usage of ./flowercare-json-exporter:
  -i, --adapter string                Bluetooth device to use for communication. (default "hci0")
  -a, --addr string                   Address to listen on for connections. (default ":9294")
      --log-level level               Minimum log level to show. (default info)
  -r, --refresh-duration duration     Interval used for refreshing data from bluetooth devices. (default 2m0s)
      --refresh-timeout duration      Timeout for reading data from a sensor. (default 1m0s)
      --retry-factor float            Factor used to multiply wait time for subsequent retries. (default 2)
      --retry-max-duration duration   Maximum wait time between retries on error. (default 30m0s)
      --retry-min-duration duration   Minimum wait time between retries on error. (default 30s)
  -s, --sensor address                MAC-address of sensor to collect data from. Can be specified multiple times.
      --stale-duration duration       Duration after which data is considered stale and is not used for metrics anymore. (default 5m0s)
```

After starting, the server will offer the sensor data as JSON on the `/sensors` endpoint.

The exporter uses an internal cache, so that each scrape of the exporter does not try to read data from the sensors to avoid unnecessary drain of the battery.

All sensors can optionally have a "name" assigned to them, so they are more easily identifiable in the JSON output. This is possible by prefixing the MAC-address with `name=`, for example:

```bash
./flowercare-json-exporter -s tomatoes=AA:BB:CC:DD:EE:FF
```
### Example docker compose file
```yaml
services:
  flowercare-json-exporter:
    image: flowercare-json-exporter:dev
    container_name: flowercare-json-exporter
    restart: unless-stopped
    privileged: true
    command: -r 2m -i hci0 -s plant1=00:11:22:33:44:55
    devices:
      - /dev/hci0
    network_mode: host
    environment:
      - TZ=Europe/Berlin
```