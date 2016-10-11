/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2015 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package snapheka

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/url"

	log "github.com/Sirupsen/logrus"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core/ctypes"
)

const (
	//vendor namespace part
	vendor     = "intel"
	pluginName = "heka"
	version    = 3
	pluginType = plugin.PublisherPluginType
)

// Meta returns a plugin meta data
func Meta() *plugin.PluginMeta {
	return plugin.NewPluginMeta(
		pluginName,
		version,
		pluginType,
		[]string{plugin.SnapGOBContentType},
		[]string{plugin.SnapGOBContentType},
		plugin.RoutingStrategy(plugin.StickyRouting),
		plugin.ConcurrencyCount(1),
	)
}

//NewHekaPublisher returns an instance of the Heka publisher
func NewHekaPublisher() *hekaPublisher {
	return &hekaPublisher{}
}

type hekaPublisher struct {
}

// GetConfigPolicy returns the config of the Heka plugin
func (p *hekaPublisher) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	cp := cpolicy.New()
	config := cpolicy.NewPolicyNode()

	r1, err := cpolicy.NewStringRule("host", true)
	handleErr(err)
	r1.Description = "Heka host"
	config.Add(r1)

	r2, err := cpolicy.NewIntegerRule("port", true)
	handleErr(err)
	r2.Description = "Heka port"
	config.Add(r2)

	r3, err := cpolicy.NewStringRule("mappings-file", false)
	handleErr(err)
	r3.Description = "Heka plugin mappings JSON/XML file"
	config.Add(r3)

	cp.Add([]string{vendor, pluginName}, config)
	return cp, nil
}

// Publish publishes metric data to heka.
func (p *hekaPublisher) Publish(contentType string, content []byte, config map[string]ctypes.ConfigValue) error {
	logger := log.New()
	var metrics []plugin.MetricType

	switch contentType {
	case plugin.SnapGOBContentType:
		dec := gob.NewDecoder(bytes.NewBuffer(content))
		if err := dec.Decode(&metrics); err != nil {
			logger.Printf("Error decoding GOB: error=%v content=%v", err, content)
			return err
		}
	case plugin.SnapJSONContentType:
		err := json.Unmarshal(content, &metrics)
		if err != nil {
			logger.Printf("Error decoding JSON: error=%v content=%v", err, content)
			return err
		}
	default:
		logger.Printf("Error unknown content type '%v'", contentType)
		return fmt.Errorf("Unknown content type '%s'", contentType)
	}

	u, err := url.Parse(fmt.Sprintf("%s:%d", config["host"].(ctypes.ConfigValueStr).Value, config["port"].(ctypes.ConfigValueInt).Value))
	handleErr(err)
	mappingsFile := ""
	if mFile, ok := config["mappings-file"]; ok {
		mappingsFile = mFile.(ctypes.ConfigValueStr).Value
	}

	// Publish metric data to Heka through TCP
	shc, _ := NewSnapHekaClient(fmt.Sprintf("tcp://%s", u), mappingsFile)
	err = shc.sendToHeka(metrics)
	handleErr(err)

	return nil
}

func handleErr(e error) {
	if e != nil {
		panic(e)
	}
}
