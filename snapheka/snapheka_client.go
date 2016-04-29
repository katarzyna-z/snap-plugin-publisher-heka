package snapheka

import (
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
func (shc *SnapHekaClient) sendToHeka(metrics []plugin.PluginMetricType) error {
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
		b, _, e := plugin.MarshalPluginMetricTypes(plugin.SnapJSONContentType, []plugin.PluginMetricType{m})
		if e != nil {
			logger.WithField("_block", "sendToHeka").Error("marshal metric error: ", m)
			continue
		}

		// Converts snap metrics to Heka message
		payload := snapToHekaPayload(string(b), m, pid, hostname)
		err = encoder.EncodeMessageStream(payload, &buf)
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

// snapToHekaPayload converts snap metric data into Heka message
func snapToHekaPayload(pl string, m plugin.PluginMetricType, pid int32, hostname string) *message.Message {
	msg := &message.Message{}
	msg.SetUuid(uuid.NewRandom())
	msg.SetTimestamp(time.Now().UnixNano())
	msg.SetType(SnapHekaMsgType)
	msg.SetLogger(SnapHekaMsgLogger)
	msg.SetSeverity(6)
	msg.SetPayload(pl)
	msg.SetPid(pid)
	msg.SetHostname(hostname)

	addField("namespace", strings.Join(m.Namespace(), "."), msg)
	addField("data", getData(m.Data()), msg)
	addField("source", m.Source(), msg)
	addField("version", m.Version(), msg)
	addField("timestamp", m.Timestamp().UnixNano(), msg)

	return msg
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
