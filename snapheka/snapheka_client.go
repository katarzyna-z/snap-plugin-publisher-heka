package snapheka

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/mozilla-services/heka/client"
	"github.com/mozilla-services/heka/message"
	"github.com/pborman/uuid"

	"github.com/intelsdi-x/snap/control/plugin"
)

const (
	SnapHekaMsgType   = "snap.heka"
	SnapHekaMsgLogger = "snap.heka.logger"
)

var (
	logger = log.WithField("_module", "_snap_heka")
)

// SnapHekaClient defines the Heka connection scheme (e.g. tcp)
// and the connection address.
type SnapHekaClient struct {
	hekaScheme string
	hekaHost   string
}

// NewSnapHekaClient creates a new instance of
func NewSnapHekaClient(addr string) (shc *SnapHekaClient, err error) {
	logger.WithField("_block", "NewSnapHekaClient").Info("Enter NewSnapHekaClient")

	shc = &SnapHekaClient{}

	hekaURL, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, err
	}

	shc.hekaScheme = hekaURL.Scheme
	shc.hekaHost = hekaURL.Host
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
			logger.WithField("_block", "sendToHeka").Info("sending message error: ", err)
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
	msg.SetSeverity(6)
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
		// Dynamic element is not inserted in metric name
		// but rather added to dimension field
		if elt.IsDynamic() {
			dimField, err = addToDimensions(dimField, elt.Name)
			if err != nil {
				logger.WithField("_block", "fillMetricNameTagsDimensions").Error(err)
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
			dimField, err = addToDimensions(dimField, tag)
			if err != nil {
				logger.WithField("_block", "fillMetricNameTagsDimensions").Error(err)
				return err
			}
			addField(tag, value, msg)
		}
	}
	if dimField != nil {
		msg.AddField(dimField)
	}
	addField("name", strings.Join(mName, "."), msg)
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
