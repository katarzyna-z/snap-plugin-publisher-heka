package snapheka

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/ghodss/yaml"
	"github.com/mozilla-services/heka/client"
	"github.com/mozilla-services/heka/message"
	"github.com/pborman/uuid"

	"github.com/intelsdi-x/snap/control/plugin"
)

const (
	SnapDfltHekaSeverity  = 6
	SnapDfltHekaMsgType   = "snap.heka"
	SnapDfltHekaMsgLogger = "snap.heka.logger"
)

var (
	logger                              = log.WithField("_module", "_snap_heka")
	SnapHekaSeverity  int32             = SnapDfltHekaSeverity
	SnapHekaMsgType                     = SnapDfltHekaMsgType
	SnapHekaMsgLogger                   = SnapDfltHekaMsgLogger
	MetricMappings    map[string]string = make(map[string]string)
)

// SnapHekaClient defines the Heka connection scheme (e.g. tcp)
// and the connection address.
type SnapHekaClient struct {
	hekaScheme string
	hekaHost   string
}

type mappings struct {
	Severity    int32             `json:"severity"yaml:"severity"`
	MessageType string            `json:"type"yaml:"type"`
	Logger      string            `json:"logger"yaml:"logger"`
	Namespace   map[string]string `json:"namespace"yaml:"namespace"`
	Metrics     map[string]string `json:"metrics"yaml:"metrics"`
}

var (
	globalMappings = mappings{}
)

// HandleMappingsFile
// TODO protect initialization using mutex
// for potential race conditions
func HandleMappingsFile(mfile string) {
	if len(mfile) == 0 {
		return
	}
	logger.WithField("_block", "HandleMappingsFile").Debug(
		fmt.Sprintf("HandleMappingsFile checking mappings file %s",
			mfile))
	if _, err := os.Stat(mfile); err != nil {
		logger.WithField("_block", "HandleMappingsFile").Warning(
			fmt.Sprintf("HandleMappingsFile mappings file %s does not exist (ignoring)",
				mfile))
		return
	}
	ext := filepath.Ext(mfile)
	mcontent, e := ioutil.ReadFile(mfile)
	if e != nil {
		logger.WithField("_block", "HandleMappingsFile").Warning(
			fmt.Sprintf("HandleMappingsFile mappings file %s reading error %v (ignoring)",
				mfile, e))
		return
	}
	logger.WithField("_block", "HandleMappingsFile").Debug(
		fmt.Sprintf("HandleMappingsFile mappings file %s ext: %s\ncontents: %s",
			mfile, ext, mcontent))
	parsed := true
	switch ext {
	case ".yaml", ".yml":
		e = yaml.Unmarshal(mcontent, &globalMappings)
		if e != nil {
			logger.WithField("_block", "HandleMappingsFile").Warning(
				fmt.Sprintf("HandleMappingsFile error parsing YAML mappings file %s: %v",
					mfile, e))
			parsed = false
		}
	case ".json":
		e = json.Unmarshal(mcontent, &globalMappings)
		if e != nil {
			logger.WithField("_block", "HandleMappingsFile").Warning(
				fmt.Sprintf("HandleMappingsFile error parsing JSON mappings file %s: %v",
					mfile, e))
			parsed = false
		}
	default:
		logger.WithField("_block", "HandleMappingsFile").Warning(
			fmt.Sprintf("HandleMappingsFile mappings file %s extension not supported: %s (should be one of .json .yaml .yml)",
				mfile, ext))
		parsed = false
	}
	if parsed {
		logger.WithField("_block", "HandleMappingsFile").Debug(
			fmt.Sprintf("HandleMappingsFile mappings file %s\nMappings: %#+v",
				mfile, globalMappings))
		if globalMappings.Severity > 0 {
			SnapHekaSeverity = globalMappings.Severity
		}
		if len(globalMappings.MessageType) > 0 {
			SnapHekaMsgType = globalMappings.MessageType
		}
		if len(globalMappings.Logger) > 0 {
			SnapHekaMsgLogger = globalMappings.Logger
		}
	}
	logger.WithField("_block", "HandleMappingsFile").Info(
		fmt.Sprintf("Using Severity=%d MessageType=%s Logger=%s",
			SnapHekaSeverity, SnapHekaMsgType, SnapHekaMsgLogger))
}

// NewSnapHekaClient creates a new instance of Heka client
func NewSnapHekaClient(addr string, mfile string) (shc *SnapHekaClient, err error) {
	logger.WithField("_block", "NewSnapHekaClient").Debug("Enter NewSnapHekaClient")

	shc = &SnapHekaClient{}

	hekaURL, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, err
	}

	shc.hekaScheme = hekaURL.Scheme
	shc.hekaHost = hekaURL.Host
	HandleMappingsFile(mfile)
	return shc, nil
}

// sendToHeka sends array of snap metrics to Heka
func (shc *SnapHekaClient) sendToHeka(metrics []plugin.MetricType) error {
	pid := int32(os.Getpid())
	hostname, _ := os.Hostname()

	// Initializes Heka message encoder
	encoder := client.NewProtobufEncoder(nil)

	// Creates Heka message sender
	sender, err := client.NewNetworkSender(shc.hekaScheme, shc.hekaHost)
	if err != nil {
		logger.WithField("_block", "sendToHeka").Error("create NewNetworkSender error: ", err)
		return err
	}

	var buf []byte
	for _, m := range metrics {
		b, _, e := plugin.MarshalMetricTypes(plugin.SnapJSONContentType, []plugin.MetricType{m})
		if e != nil {
			logger.WithField("_block", "sendToHeka").Error("marshal metric error: ", m)
			continue
		}

		// Converts snap metric to Heka message
		msg, err := createHekaMessage(string(b), m, pid, hostname)
		if err != nil {
			logger.WithField("_block", "sendToHeka").Error("create message error: ", err)
			continue
		}
		err = encoder.EncodeMessageStream(msg, &buf)
		if err != nil {
			logger.WithField("_block", "sendToHeka").Error("encoding error: ", err)
			continue
		}

		err = sender.SendMessage(buf)
		if err != nil {
			logger.WithField("_block", "sendToHeka").Error("sending message error: ", err)
		}
	}
	sender.Close()
	return nil
}

// createHekaMessage converts a Snap metric into an Heka message
func createHekaMessage(pl string, m plugin.MetricType, pid int32, hostname string) (*message.Message, error) {
	msg := &message.Message{}
	msg.SetUuid(uuid.NewRandom())
	msg.SetTimestamp(time.Now().UnixNano())
	msg.SetType(SnapHekaMsgType)
	msg.SetLogger(SnapHekaMsgLogger)
	msg.SetSeverity(SnapHekaSeverity)
	msg.SetPayload(pl)
	msg.SetPid(pid)
	msg.SetHostname(hostname)

	err := setHekaMessageFields(m, msg)
	if err != nil {
		errStr := fmt.Sprintf("Can not extract metric name, tags or dimensions")
		log.Error(errStr)
		return nil, errors.New(errStr)
	}
	return msg, nil
}

// Function used to add a specific dynamic metric namespace element
// into the dimensions field of final Heka message structure
func addToDimensions(f *message.Field, fName string) (*message.Field, error) {
	// If the dimension field does not exists yet, create ti
	if f == nil {
		field, err := message.NewField("dimensions", fName, "")
		if err != nil {
			logger.WithField("_block", "addToDimensions").Error(err)
			return nil, err
		}
		return field, nil
	}
	// Add field name to dimension field
	f.AddValue(fName)
	return f, nil
}

// function which fills all part of Heka message
func setHekaMessageFields(m plugin.MetricType, msg *message.Message) error {
	mName := make([]string, 0, len(m.Namespace()))
	var dimField *message.Field
	var err error
	// Loop on namespace elements
	for _, elt := range m.Namespace() {
		logger.WithField("_block", "setHekaMessageFields").Debug(
			fmt.Sprintf("Namespace %#+v",
				elt))
		// Dynamic element is not inserted in metric name
		// but rather added to dimension field
		if elt.IsDynamic() {
			dimField, err = addToDimensions(dimField, elt.Name)
			if err != nil {
				logger.WithField("_block", "setHekaMessageFields").Error(err)
				return err
			}
			addField(elt.Name, elt.Value, msg)
		} else {
			// Static element is concatenated to metric name
			mName = append(mName, elt.Value)
		}
	}
	// Processing of tags
	if len(m.Tags()) > 0 {
		for tag, value := range m.Tags() {
			logger.WithField("_block", "setHekaMessageFields").Debug(
				fmt.Sprintf("Adding tag=%s value=%s",
					tag, value))
			dimField, err = addToDimensions(dimField, tag)
			if err != nil {
				logger.WithField("_block", "setHekaMessageFields").Error(err)
				return err
			}
			addField(tag, value, msg)
		}
	}
	if dimField != nil {
		msg.AddField(dimField)
	}
	// Handle metric name
	metricName := strings.Join(mName, ".")
	// TODO protect access using mutex
	// for potential race conditions
	logger.WithField("_block", "setHekaMessageFields").Debug(
		fmt.Sprintf("Checking metric=%s",
			metricName))
	// Is mapping already stored
	if val, ok := MetricMappings[metricName]; ok {
		logger.WithField("_block", "setHekaMessageFields").Debug(
			fmt.Sprintf("Metric=%s in cache %s",
				metricName, val))
		metricName = val
	} else {
		oldMetricName := metricName
		logger.WithField("_block", "setHekaMessageFields").Debug(
			fmt.Sprintf("Metric=%s not in cache",
				metricName))
		// Namespace handling
		for kmapping, vmapping := range globalMappings.Namespace {
			logger.WithField("_block", "setHekaMessageFields").Debug(
				fmt.Sprintf("Checking metric=%s against namespace %s (%s)",
					metricName, kmapping, vmapping))
			// Try to see if substitution changes something
			newMetricName := strings.Replace(metricName, kmapping, vmapping, 1)
			if strings.Compare(newMetricName, metricName) != 0 {
				MetricMappings[oldMetricName] = newMetricName
				logger.WithField("_block", "setHekaMessageFields").Debug(
					fmt.Sprintf("Changing metric=%s into %s",
						metricName, newMetricName))
				metricName = newMetricName
			}
		}
		// Metrics handling
		for kmapping, vmapping := range globalMappings.Metrics {
			logger.WithField("_block", "setHekaMessageFields").Debug(
				fmt.Sprintf("Checking metric=%s against metric %s (%s)",
					metricName, kmapping, vmapping))
			// Try to see if substitution changes something
			newMetricName := strings.Replace(metricName, kmapping, vmapping, 1)
			if strings.Compare(newMetricName, metricName) != 0 {
				MetricMappings[oldMetricName] = newMetricName
				logger.WithField("_block", "setHekaMessageFields").Debug(
					fmt.Sprintf("Changing metric=%s into %s",
						metricName, newMetricName))
				metricName = newMetricName
			}
		}
	}
	addField("name", metricName, msg)
	addField("value", getData(m.Data()), msg)
	addField("timestamp", m.Timestamp().UnixNano(), msg)
	return nil
}

// getData converts unit64 to int64 for Heka supported data type
func getData(v interface{}) interface{} {
	switch d := v.(type) {
	case uint64:
		return int64(d)
	case uint32:
		return int32(d)
	default:
		return d
	}
}

func addField(name string, value interface{}, msg *message.Message) {
	field, err := message.NewField(name, value, "")
	if err == nil {
		msg.AddField(field)
	}
}
