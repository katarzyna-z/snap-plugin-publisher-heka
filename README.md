<!--
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2016 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# snap heka publisher plugin

This plugin publishes snap metric data into heka via TCP.

It's used in the [snap framework](http://github.com/intelsdi-x/snap).

1. [Getting Started](#getting-started)
  * [System Requirements](#system-requirements)
  * [Building from source](#building-from-source)
  * [Configuration and Usage](configuration-and-usage)
2. [Documentation](#documentation)
  * [Collected Metrics](#collected-metrics)
  * [Examples](#examples)
  * [Roadmap](#roadmap)
3. [Community Support](#community-support)
4. [Contributing](#contributing)
5. [License](#license-and-authors)
6. [Acknowledgements](#acknowledgements)

## Getting Started
### System Requirements
* [snap] (https://github.com/intelsdi-x/snap)
* [heka](https://github.com/mozilla-services/heka/)
* [golang 1.5+](https://golang.org/dl/)

### Operating systems
All OSs currently supported by snap:
* Linux/amd64
* Darwin/amd64

### Building from source
* Get the package: 
```go get github.com/intelsdi-x/snap-plugin-publisher-heka```
* Build the snap-plugin-publisher-heka plugin
 *  From the root of the snap-plugin-publisher-heka path type ```make all```.
   * This builds the plugin in `/build/rootfs/`.

### Configuration and Usage
* Set up the [snap framework](https://github.com/intelsdi-x/snap/blob/master/README.md#getting-started)
* Ensure `$SNAP_PATH` is exported
`export SNAP_PATH=$GOPATH/src/github.com/intelsdi-x/snap/build`
* Make sure the Docker is ready on your machine before you run integration tests.
 * cd snap-plugin-publisher-heka
  * make test-unit 
  * make test-integration

## Documentation

### Suitable Metrics
All metrics that is complaint with snap metric type definition.


### Examples
Assuming that, you have a heka instance running with the appropriate configuration. For example:
``` 
$ sudo hekad -config=tcp-input-multioutputs.toml
2016/01/13 14:09:06 Pre-loading: [RstEncoder]
2016/01/13 14:09:06 Pre-loading: [InfluxdbLineEncoder]
2016/01/13 14:09:06 Pre-loading: [ESJsonEncoder]
2016/01/13 14:09:06 Pre-loading: [ElasticSearchOutput]
2016/01/13 14:09:06 Pre-loading: [tcp_heka_output_encoder]
2016/01/13 14:09:06 Pre-loading: [tcp_in:3242]
2016/01/13 14:09:06 Pre-loading: [tcp_heka_output_log]
2016/01/13 14:09:06 Pre-loading: [InfluxdbOutput]
2016/01/13 14:09:06 Pre-loading: [ProtobufDecoder]
2016/01/13 14:09:06 Loading: [ProtobufDecoder]
2016/01/13 14:09:06 Pre-loading: [ProtobufEncoder]
2016/01/13 14:09:06 Loading: [ProtobufEncoder]
2016/01/13 14:09:06 Pre-loading: [TokenSplitter]
2016/01/13 14:09:06 Loading: [TokenSplitter]
2016/01/13 14:09:06 Pre-loading: [HekaFramingSplitter]
2016/01/13 14:09:06 Loading: [HekaFramingSplitter]
2016/01/13 14:09:06 Pre-loading: [NullSplitter]
2016/01/13 14:09:06 Loading: [NullSplitter]
2016/01/13 14:09:06 Loading: [RstEncoder]
2016/01/13 14:09:06 Loading: [InfluxdbLineEncoder]
2016/01/13 14:09:06 Loading: [ESJsonEncoder]
2016/01/13 14:09:06 Loading: [tcp_heka_output_encoder]
2016/01/13 14:09:06 Loading: [tcp_in:3242]
2016/01/13 14:09:06 Loading: [ElasticSearchOutput]
2016/01/13 14:09:06 Loading: [tcp_heka_output_log]
2016/01/13 14:09:06 Loading: [InfluxdbOutput]
2016/01/13 14:09:06 Starting hekad...
2016/01/13 14:09:06 Output started: ElasticSearchOutput
2016/01/13 14:09:06 Output started: tcp_heka_output_log
2016/01/13 14:09:06 Output started: InfluxdbOutput
2016/01/13 14:09:06 MessageRouter started.
2016/01/13 14:09:06 Input started: tcp_in:3242
```

To run Heka inside a Docker container
```
$ docker run --name heka -it -p 4352:4352 -p 3242:3242 -v <path to heka-tcp-config.toml file>:/etc/heka/config.toml mozilla/heka -config /etc/heka/config.toml
```
Where port 4352 is the Heka dashboard port and 3242 is a sample TCP input port.


In one terminal window, start the snap daemon (in this case with logging set to 1 and trust disabled):
```
$ $SNAP_PATH/bin/snapd -l 1 -t 0
```

In another terminal window:
Load snap collector plugins. For example:
```
$ $SNAP_PATH/bin/snapctl plugin load $SNAP_PSUTIL_PLUGIN/build/rootfs/snap-plugin-collector-psutil
```
Load snap heka publisher plugin
```
$ $SNAP_PATH/bin/snapctl plugin load <path to built snap-plugin-publisher-heka plugin>
```
Create tasks. For example:
```
$ $SNAP_PATH/bin/snapctl task create -t /tmp/heka-task.json
Using task manifest to create task
Task created
ID: 69f71b95-7d32-418d-b81a-60fcc9ef2798
Name: Task-69f71b95-7d32-418d-b81a-60fcc9ef2798
State: Running

```
The example task manifest file, heka-task.json:
```json
{
    "version": 1,
    "schedule": {
        "type": "simple",
        "interval": "1s"
    },
    "workflow": {
        "collect": {
            "metrics": {
                "/intel/psutil/load/load1": {},
                "/intel/psutil/load/load5": {},
                "/intel/psutil/load/load15": {},
                "/intel/psutil/vm/available": {},
                "/intel/psutil/vm/free": {},
                "/intel/psutil/vm/used": {}
            },
            "config": {
                "/intel/mock": {
                    "password": "secret",
                    "user": "root"
                }
            },
            "process": null,
            "publish": [
                {
                    "plugin_name": "heka",                            
                    "config": {
                        "host": "HEKA_HOST_IP",
                        "port": "TCP_PORT"
                    }
                }
            ]                                            
        }
    }
}
```

Sample snap heka file output message:
```
:Timestamp: 2016-01-13 22:10:41.441442012 +0000 UTC
:Type: snap.heka
:Hostname: egu-mac01.lan
:Pid: 90945
:Uuid: f9fc6e86-b5d8-4807-829a-681bf13c4184
:Logger: snap.heka.logger
:Payload: [{"namespace":["intel","psutil","vm","free"],"last_advertised_time":"0001-01-01T00:00:00Z","version":0,"config":null,"data":2680213504,"labels":null,"tags":null,"source":"egu-mac01.lan","timestamp":"2016-01-13T14:10:41.439225319-08:00"}]
:EnvVersion: 
:Severity: 6
:Fields:
    | name:"namespace" type:string value:"intel.psutil.vm.free"
    | name:"source" type:string value:"egu-mac01.lan"
    | name:"version" type:integer value:0
    | name:"timestamp" type:integer value:1452723041439225319
```

Sample snap heka elasticsearch message:
```json
{
"_index": "intel-snap-2016.01.13",
"_type": "snap.heka",
"_id": "AVI8zvzEYXJQpvQObIO2",
"_version": 1,
"_score": 1,
"_source": {
	"Uuid": "9f243b35-c32c-41f1-bffd-be8c87f59e0f",
	"@timestamp": "2016-01-13T21:05:43",
	"Type": "snap.heka",
	"Logger": "snap.heka.logger",
	"level": 6,
	"Payload": "[{"namespace":["intel","psutil","load","load15"],"last_advertised_time":"0001-01-01T00:00:00Z","version":0,"config":null,"data":4.51,"labels":null,"tags":null,"source":"egu-mac01.lan","timestamp":"2016-01-13T13:05:43.806600468-08:00"}]",
	"EnvVersion": "",
	"Pid": 76280,
	"Hostname": "egu-mac01.lan",
	"namespace": "intel.psutil.load.load15",
	"data": 4.51,
	"source": "egu-mac01.lan",
	"version": 0,
	"timestamp": 1452719143806600400
}
}
```

Sample snap heka influxdb data:
```
> select * from namespavce
1452720018000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	intel.psutil.load.load1
1452720019000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	intel.psutil.load.load1
1452720020000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	intel.psutil.load.load1
1452720021000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	intel.psutil.load.load1
1452720022000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	intel.psutil.load.load1
1452720023000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	intel.psutil.load.load1
1452720433000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	intel.psutil.vm.available
1452720434000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	intel.psutil.vm.available

> select * from data
1452720612000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	3
1452720613000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	3
1452720614000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	3
1452723019000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.69
1452723020000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.69
1452723021000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.69
1452723022000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.69
1452723023000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.69
1452723024000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.67
1452723025000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.67
1452723026000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.67
1452723027000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.67
1452723028000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.67
1452723029000000000	egu-mac01.lan	snap.heka.logger	6		snap.heka	2.65

```

### Roadmap
There isn't a current roadmap for this plugin, but it is in active development. As we launch this plugin, we do not have any outstanding requirements for the next release. If you have a feature request, please add it as an [issue](https://github.com/intelsdi-x/snap-plugin-collector-etcd/issues/new) and/or submit a [pull request](https://github.com/intelsdi-x/snap-plugin-collector-etcd/pulls).

## Community Support
This repository is one of **many** plugins in **snap**, a powerful telemetry framework. See the full project at http://github.com/intelsdi-x/snap.

To reach out to other users, head to the [main framework](https://github.com/intelsdi-x/snap#community-support).

## Contributing
We love contributions!

There's more than one way to give back, from examples to blogs to code updates. See our recommended process in [CONTRIBUTING.md](CONTRIBUTING.md).

## License
[snap](http://github.com:intelsdi-x/snap), along with this plugin, is an Open Source software released under the Apache 2.0 [License](LICENSE).

## Acknowledgements
* Author: [@candysmurf](https://github.com/candysmurf)

And **thank you!** Your contribution, through code and participation, is incredibly important to us.

