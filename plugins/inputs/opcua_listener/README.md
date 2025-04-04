# OPC UA Client Listener Input Plugin

⭐ Telegraf v1.25.0
🏷️ network
💻 linux, windows

The `opcua_listener` plugin subscribes to data & events from OPC UA Server
devices.

## Service Input <!-- @/docs/includes/service_input.md -->

This plugin is a service input. Normal plugins gather metrics determined by the
interval setting. Service plugins start a service to listens and waits for
metrics or events to occur. Service plugins have two key differences from
normal plugins:

1. The global or plugin specific `interval` setting may not apply
2. The CLI options of `--test`, `--test-wait`, and `--once` may not produce
   output for this plugin

## Global configuration options <!-- @/docs/includes/plugin_config.md -->

In addition to the plugin-specific configuration settings, plugins support
additional global and plugin configuration settings. These settings are used to
modify metrics, tags, and field or create aliases and configure ordering, etc.
See the [CONFIGURATION.md][CONFIGURATION.md] for more details.

[CONFIGURATION.md]: ../../../docs/CONFIGURATION.md#plugins

## Secret-store support

This plugin supports secrets from secret-stores for the `username` and
`password` option.
See the [secret-store documentation][SECRETSTORE] for more details on how
to use them.

[SECRETSTORE]: ../../../docs/CONFIGURATION.md#secret-store-secrets

## Configuration

```toml @sample.conf
# Retrieve data from OPCUA devices
[[inputs.opcua_listener]]
  ## Metric name
  # name = "opcua_listener"
  #
  ## OPC UA Endpoint URL
  # endpoint = "opc.tcp://localhost:4840"
  #
  ## Maximum time allowed to establish a connect to the endpoint.
  # connect_timeout = "10s"
  #
  ## Behavior when we fail to connect to the endpoint on initialization. Valid options are:
  ##     "error": throw an error and exits Telegraf
  ##     "ignore": ignore this plugin if errors are encountered
  #      "retry": retry connecting at each interval
  # connect_fail_behavior = "error"
  #
  ## Maximum time allowed for a request over the established connection.
  # request_timeout = "5s"
  #
  # Maximum time that a session shall remain open without activity.
  # session_timeout = "20m"
  #
  ## The interval at which the server should at least update its monitored items.
  ## Please note that the OPC UA server might reject the specified interval if it cannot meet the required update rate.
  ## Therefore, always refer to the hardware/software documentation of your server to ensure the specified interval is supported.
  # subscription_interval = "100ms"
  #
  ## Security policy, one of "None", "Basic128Rsa15", "Basic256",
  ## "Basic256Sha256", or "auto"
  # security_policy = "auto"
  #
  ## Security mode, one of "None", "Sign", "SignAndEncrypt", or "auto"
  # security_mode = "auto"
  #
  ## Path to cert.pem. Required when security mode or policy isn't "None".
  ## If cert path is not supplied, self-signed cert and key will be generated.
  # certificate = "/etc/telegraf/cert.pem"
  #
  ## Path to private key.pem. Required when security mode or policy isn't "None".
  ## If key path is not supplied, self-signed cert and key will be generated.
  # private_key = "/etc/telegraf/key.pem"
  #
  ## Authentication Method, one of "Certificate", "UserName", or "Anonymous".  To
  ## authenticate using a specific ID, select 'Certificate' or 'UserName'
  # auth_method = "Anonymous"
  #
  ## Username. Required for auth_method = "UserName"
  # username = ""
  #
  ## Password. Required for auth_method = "UserName"
  # password = ""
  #
  ## Option to select the metric timestamp to use. Valid options are:
  ##     "gather" -- uses the time of receiving the data in telegraf
  ##     "server" -- uses the timestamp provided by the server
  ##     "source" -- uses the timestamp provided by the source
  # timestamp = "gather"
  #
  ## The default timetsamp format is RFC3339Nano
  # Other timestamp layouts can be configured using the Go language time
  # layout specification from https://golang.org/pkg/time/#Time.Format
  # e.g.: json_timestamp_format = "2006-01-02T15:04:05Z07:00"
  #timestamp_format = ""
  #
  #
  ## Client trace messages
  ## When set to true, and debug mode enabled in the agent settings, the OPCUA
  ## client's messages are included in telegraf logs. These messages are very
  ## noisey, but essential for debugging issues.
  # client_trace = false
  #
  ## Include additional Fields in each metric
  ## Available options are:
  ##   DataType -- OPC-UA Data Type (string)
  # optional_fields = []
  #
  ## Node ID configuration
  ## name              - field name to use in the output
  ## namespace         - OPC UA namespace of the node (integer value 0 thru 3)
  ## identifier_type   - OPC UA ID type (s=string, i=numeric, g=guid, b=opaque)
  ## identifier        - OPC UA ID (tag as shown in opcua browser)
  ## default_tags      - extra tags to be added to the output metric (optional)
  ## monitoring_params - additional settings for the monitored node (optional)
  ##
  ## Monitoring parameters
  ## sampling_interval  - interval at which the server should check for data
  ##                      changes (default: 0s)
  ## queue_size         - size of the notification queue (default: 10)
  ## discard_oldest     - how notifications should be handled in case of full
  ##                      notification queues, possible values:
  ##                      true: oldest value added to queue gets replaced with new
  ##                            (default)
  ##                      false: last value added to queue gets replaced with new
  ## data_change_filter - defines the condition under which a notification should
  ##                      be reported
  ##
  ## Data change filter
  ## trigger        - specify the conditions under which a data change notification
  ##                  should be reported, possible values:
  ##                  "Status": only report notifications if the status changes
  ##                            (default if parameter is omitted)
  ##                  "StatusValue": report notifications if either status or value
  ##                                 changes
  ##                  "StatusValueTimestamp": report notifications if either status,
  ##                                          value or timestamp changes
  ## deadband_type  - type of the deadband filter to be applied, possible values:
  ##                  "Absolute": absolute change in a data value to report a notification
  ##                  "Percent": works only with nodes that have an EURange property set
  ##                             and is defined as: send notification if
  ##                             (last value - current value) >
  ##                             (deadband_value/100.0) * ((high–low) of EURange)
  ## deadband_value - value to deadband_type, must be a float value, no filter is set
  ##                  for negative values
  ##
  ## Use either the inline notation or the bracketed notation, not both.
  #
  ## Inline notation (default_tags and monitoring_params not supported yet)
  # nodes = [
  #   {name="node1", namespace="", identifier_type="", identifier=""},
  #   {name="node2", namespace="", identifier_type="", identifier=""}
  # ]
  #
  ## Bracketed notation
  # [[inputs.opcua_listener.nodes]]
  #   name = "node1"
  #   namespace = ""
  #   identifier_type = ""
  #   identifier = ""
  #   default_tags = { tag1 = "value1", tag2 = "value2" }
  #
  # [[inputs.opcua_listener.nodes]]
  #   name = "node2"
  #   namespace = ""
  #   identifier_type = ""
  #   identifier = ""
  #
  #   [inputs.opcua_listener.nodes.monitoring_params]
  #     sampling_interval = "0s"
  #     queue_size = 10
  #     discard_oldest = true
  #
  #     [inputs.opcua_listener.nodes.monitoring_params.data_change_filter]
  #       trigger = "Status"
  #       deadband_type = "Absolute"
  #       deadband_value = 0.0
  #
  ## Node Group
  ## Sets defaults so they aren't required in every node.
  ## Default values can be set for:
  ## * Metric name
  ## * OPC UA namespace
  ## * Identifier
  ## * Default tags
  ## * Sampling interval
  ##
  ## Multiple node groups are allowed
  #[[inputs.opcua_listener.group]]
  ## Group Metric name. Overrides the top level name.  If unset, the
  ## top level name is used.
  # name =
  #
  ## Group default namespace. If a node in the group doesn't set its
  ## namespace, this is used.
  # namespace =
  #
  ## Group default identifier type. If a node in the group doesn't set its
  ## namespace, this is used.
  # identifier_type =
  #
  ## Default tags that are applied to every node in this group. Can be
  ## overwritten in a node by setting a different value for the tag name.
  ##   example: default_tags = { tag1 = "value1" }
  # default_tags = {}
  #
  ## Group default sampling interval. If a node in the group doesn't set its
  ## sampling interval, this is used.
  # sampling_interval = "0s"
  #
  ## Node ID Configuration.  Array of nodes with the same settings as above.
  ## Use either the inline notation or the bracketed notation, not both.
  #
  ## Inline notation (default_tags and monitoring_params not supported yet)
  # nodes = [
  #  {name="node1", namespace="", identifier_type="", identifier=""},
  #  {name="node2", namespace="", identifier_type="", identifier=""}
  #]
  #
  ## Bracketed notation
  # [[inputs.opcua_listener.group.nodes]]
  #   name = "node1"
  #   namespace = ""
  #   identifier_type = ""
  #   identifier = ""
  #   default_tags = { tag1 = "override1", tag2 = "value2" }
  #
  # [[inputs.opcua_listener.group.nodes]]
  #   name = "node2"
  #   namespace = ""
  #   identifier_type = ""
  #   identifier = ""
  #
  #   [inputs.opcua_listener.group.nodes.monitoring_params]
  #     sampling_interval = "0s"
  #     queue_size = 10
  #     discard_oldest = true
  #
  #     [inputs.opcua_listener.group.nodes.monitoring_params.data_change_filter]
  #       trigger = "Status"
  #       deadband_type = "Absolute"
  #       deadband_value = 0.0
  #

  ## Enable workarounds required by some devices to work correctly
  # [inputs.opcua_listener.workarounds]
    ## Set additional valid status codes, StatusOK (0x0) is always considered valid
    # additional_valid_status_codes = ["0xC0"]

  # [inputs.opcua_listener.request_workarounds]
    ## Use unregistered reads instead of registered reads
    # use_unregistered_reads = false


  ## Multiple event groups are allowed.
  # [[inputs.opcua_listener.eventgroup]]
  #   sampling_interval = "10s"
  #   queue_size = 10
  #   namespace = ""
  #   identifier_type = ""
  #   source_names = ["SourceName1", "SourceName2"]
  #   fields = ["Severity", "Message", "Time"]
  #
  #   [inputs.opcua_listener.eventgroup.event_type_node]
  #     namespace = ""
  #     identifier_type = ""
  #     identifier = ""
  #
  #   [[inputs.opcua_listener.eventgroup.node_ids]]
  #     namespace = ""
  #     identifier_type = ""
  #     identifier = ""

```

## Node Configuration

An OPC UA node ID may resemble: "ns=3;s=Temperature". In this example:

- ns=3 is indicating the `namespace` is 3
- s=Temperature is indicting that the `identifier_type` is a string and `identifier` value is 'Temperature'
- This example temperature node has a value of 79.0
To gather data from this node enter the following line into the 'nodes' property above:

```text
{name="temp", namespace="3", identifier_type="s", identifier="Temperature"},
```

This node configuration produces a metric like this:

```text
opcua,id=ns\=3;s\=Temperature temp=79.0,Quality="OK (0x0)" 1597820490000000000
```

With 'DataType' entered in Additional Metrics, this node configuration
produces a metric like this:

```text
opcua,id=ns\=3;s\=Temperature temp=79.0,Quality="OK (0x0)",DataType="Float" 1597820490000000000
```

## Group Configuration

Groups can set default values for the namespace, identifier type, tags
settings and sampling interval.  The default values apply to all the
nodes in the group.  If a default is set, a node may omit the setting
altogether. This simplifies node configuration, especially when many
nodes share the same namespace or identifier type.

The output metric will include tags set in the group and the node.  If
a tag with the same name is set in both places, the tag value from the
node is used.

This example group configuration has three groups with two nodes each:

```toml
  # Group 1
  [[inputs.opcua_listener.group]]
    name = "group1_metric_name"
    namespace = "3"
    identifier_type = "i"
    default_tags = { group1_tag = "val1" }
    [[inputs.opcua.group.nodes]]
      name = "name"
      identifier = "1001"
      default_tags = { node1_tag = "val2" }
    [[inputs.opcua.group.nodes]]
      name = "name"
      identifier = "1002"
      default_tags = {node1_tag = "val3"}

  # Group 2
  [[inputs.opcua_listener.group]]
    name = "group2_metric_name"
    namespace = "3"
    identifier_type = "i"
    default_tags = { group2_tag = "val3" }
    [[inputs.opcua.group.nodes]]
      name = "saw"
      identifier = "1003"
      default_tags = { node2_tag = "val4" }
    [[inputs.opcua.group.nodes]]
      name = "sin"
      identifier = "1004"

  # Group 3
  [[inputs.opcua_listener.group]]
    name = "group3_metric_name"
    namespace = "3"
    identifier_type = "i"
    default_tags = { group3_tag = "val5" }
    nodes = [
      {name="name", identifier="1001"},
      {name="name", identifier="1002"},
    ]
```

## Metrics

The metrics collected by this input plugin will depend on the
configured `nodes` and `group`.

## Example Output

```text
group1_metric_name,group1_tag=val1,id=ns\=3;i\=1001,node1_tag=val2 name=0,Quality="OK (0x0)" 1606893246000000000
group1_metric_name,group1_tag=val1,id=ns\=3;i\=1002,node1_tag=val3 name=-1.389117,Quality="OK (0x0)" 1606893246000000000
group2_metric_name,group2_tag=val3,id=ns\=3;i\=1003,node2_tag=val4 Quality="OK (0x0)",saw=-1.6 1606893246000000000
group2_metric_name,group2_tag=val3,id=ns\=3;i\=1004 sin=1.902113,Quality="OK (0x0)" 1606893246000000000
```

## Connection Service

This plugin subscribes to the specified nodes to receive data from
the OPC server. The updates are received at most as fast as the
`subscription_interval`.
Events are received within intervalls defined in `sampling_interval`

## Event Streaming Configuration (in general)

This additional group enables monitoring of
OPC UA events by subscribing to specific node IDs and filtering events based on
event_type and source_name.
Once configured, Telegraf subscribes to the specified event_type’s Node-ID,
and collects events that meet the defined criteria.
The `node_ids` parameter specifies the nodes to monitor for
events (monitored items).
However, the actual subscription is based on the `event_type_node`,
which determines the events that are capture.

## Event Streaming Configuration Parameters (definitions)

- `sampling_interval` Polling interval for data collection (default: 10s).
- `queue_size`Size of the notification queue (default: 10).
- `node_ids` A list of OPC UA node identifiers (NodeIds) specifying the nodes to monitor for event notifications, which are associated with the defined event type.
- `event_type_node` Defines the type or level of events to capture from the monitored nodes.
- `fields` Specifies the fields to capture from event notifications.
- `source_names` Specifies OPCUA Event source_names to filter on (optional).

## Basic Configurations

```toml
# Event streaming is similar to value queries and can work alone or 
# in combination with Nodes/GroupNodes.  The following configuration 
# parameters also apply to event streaming:
[[inputs.opcua_listener]]
  # name = "opcua_listener"
  #
  # endpoint = "opc.tcp://localhost:4840"
  #
  # connect_timeout = "10s"
  #
  # connect_fail_behavior = "error"
  #
  # request_timeout = "5s"
  #
  # session_timeout = "20m"
  #
  # subscription_interval = "100ms"
  #
  # security_policy = "auto"
  #
  # security_mode = "auto"
  #
  # certificate = "/etc/telegraf/cert.pem"
  #
  # private_key = "/etc/telegraf/key.pem"
  #
  # auth_method = "Anonymous"
  #
  # username = ""
  #
  # password = ""
  #
  # client_trace = false
  #
  [[inputs.opcua_listener.eventgroup]]
```

## Event Group Configuration

You can define multiple groups for the event streaming
to subscribe to different event_types. Each group has default values
for namespace and identifier_type.
Defined namespace and identifier_type within the node_ids will
override this default values.
Event_type_node is not affected by default values and all 3
parameters must be set within this stanza.

This example group configuration shows how to use group settings:

```toml
# Group 1
[[inputs.opcua_listener.eventgroup]]
   sampling_interval = "10s"
   queue_size = "100"
   source_names = ["SourceName1", "SourceName2"]
   fields = ["Severity", "Message", "Time"]

   [inputs.opcua_listener.eventgroup.event_type_node]
     namespace = "1"
     identifier_type = "i"
     identifier = "1234"

   [[inputs.opcua_listener.eventgroup.node_ids]]
     namespace = "2"
     identifier_type = "i"
     identifier = "2345"

# Group 2
[[inputs.opcua_listener.eventgroup]]
   sampling_interval = "10s"
   queue_size = "100"
   namespace = "3"
   identifier_type = "s"
   source_names = ["SourceName1", "SourceName2"]
   fields = ["Severity", "Message", "Time"]

   [inputs.opcua_listener.eventgroup.event_type_node]
     namespace = "1"
     identifier_type = "i"
     identifier = "5678"

    node_ids = [
      {identifier="Sensor1"}, // default values will be used for namespace and identifier_type 
      {namespace="2", identifier="TemperatureSensor"}, // default values will be used for identifier_type
      {namespace="5", identifier_type="i", identifier="2002"} // no default values will be used
    ]
```

## Event Streaming Metrics

Measurement names are based on the OPC UA fields selected in the
telegraf config.
All the fields are added to the Output `fields`.
All metrics receive the node_id & source `tags` indicating
the related NodeID and OPCUA Server where the event is coming from.

## Event Streaming Output

```text
{
    "fields": {
        "EventType": "i=10751",
        "Message": "The alarm severity has increased.",
        "SourceName": "SouthMotor",
        "Time": "2024-12-09 07:46:48.8492578 +0000 UTC"
    },
    "name": "opcua_event",
    "tags": {
        "host": "myHost",
        "node_id": "ns=2;s=0:East/Blue",
        "opcua_host": "opc.tcp://opcua.demo-this.com:62544/Quickstarts/AlarmConditionServer"
    },
    "timestamp": 1733730411
}
```
