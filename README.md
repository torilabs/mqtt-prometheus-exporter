# MQTT Prometheus Exporter

_The MQTT Prometheus exporter subscribes to the MQTT broker and publish the received messages as prometheus metrics._

Collected metrics (together with application metrics) are exposed on `/metrics` endpoint. Prometheus target is then configured with this endpoint and port e.g. `http://localhost:8079/metrics`.

Collected metric contains exact time of message read. This helps prometheus and other tools like Grafana to interpret the values correctly on time axis. The value and time are updated when new message is processed from MQTT broker and topic and all the labels match.

**Raw or JSON message**

MQTT Prometheus exporter consumes messages containing raw numeric value e.g. `12.5` or the value encoded in JSON e.g. `{"temperature":12.5}`. 
To enable JSON message format and consume values use `json_field` in metrics configuration.

**Example of metric**
```
# HELP temperature temperature measured on home sensors
# TYPE temperature gauge
temperature{mylabel="label value",topic="/home/kitchen/temperature"} 21.568 1601809393358
temperature{mylabel="label value",topic="/home/bedroom/temperature"} 20.155 1601809389335
```

## Configuration

MQTT Prometheus exporter requires yaml configuration file to be provided.

### Config file

If the default value match with your choice you can omit it.

```yaml
# Logger configuration
logging:
  # logging level - default: INFO
  level: DEBUG
  # development mode of logging - default: false
  developmentMode: false

# HTTP server configuration
server:
  # server port - default: 8079
  port: 8080

# MQTT client configuration
mqtt:
  # MQTT broker to connect to - default is: tcp://127.0.0.1
  # The format should be "scheme://host", where "scheme"
  # is one of "tcp", "ssl", or "ws", "host" is the ip-address (or hostname).
  # Default values for hostname is "127.0.0.1", for schema is "tcp://".
  # IPv6 is supported. Use host in format of "[fd12:3456:789a::1]".
  host: "ws://10.0.0.15"
  # MQTT broker port - default: 9641
  port: 9001
  # username for connection to MQTT broker
  username: ""
  # password for connection to MQTT broker
  password: ""
  #connection timeout - default: 3s
  timeout: 3s

# internal cache holding collected metrics configuration
cache:
  # expiration duration of collected entries - default: 60s
  # expiration <= 0 means no expiration
  expiration: 60s

# list of metrics to be exported
metrics:
    # name of the MQTT topic
  - mqtt_topic: "/home/+/temperature"
    # name of the exported metric in prometheus
    prom_name: "temperature"
    # type of prometheus metric, valid values are: "gauge" and "counter"
    type: "gauge"
    # prometheus help text of the metric
    help: "temperature measured on home sensors"
    # list of constant labels with values added to metric
    const_labels:
      - mylabel: "label value"
    # list of labels with topic name index
    # index represents part number of topic name split by "/"
    # example below add {device: "sensorX", negative_idx: "sensorX", out_of_range: ""} labels
    #   when message received from topic "/home/sensorX/temperature"
    topic_labels:
      - device: 2
      - negative_idx: -2
      - out_of_range: 5
  - mqtt_topic: "/home/rpi/memory"
    prom_name: "rpi_memory"
    type: "gauge"
    help: "free memory of the Raspberry Pi"
  - mqtt_topic: "/home/overview"
    prom_name: "sensor_count"
    type: "gauge"
    # using json_field you can consume message in a valid JSON format
    # value is then parsed from JSON tree by the given path/field
    json_field: "total.count"
```

Minimal config file can contain only `metrics` definition. Default values will be used for logging level (`INFO`), HTTP server port (`8079`) and MQTT broker URI (`:9641`).


## Build & Run
To build the binary run:
```bash
make build
```

Run the binary with optional `config` parameter provided:
```bash
./mqtt-prometheus-exporter [--config=<path to yaml config file>]
```
If you don't provide `config` parameter, application will search on default path: `./config.yaml`.

## Healthcheck
Remember to use `/healthcheck` endpoint to verify exporter works correctly and is connected to MQTT broker. Healthcheck returns `HTTP 200` when healthy. This is standard in cloud computing.

In case app is unhealthy (returning non HTTP 200 code) n times in a row, try to restart the container.

## Docker image
Public docker image is available for multiple platforms: https://hub.docker.com/r/torilabs/mqtt-prometheus-exporter
```
docker run -it -p 8079:8079 -v $(pwd)/my-config.yaml:/config.yaml --rm torilabs/mqtt-prometheus-exporter:latest
```
