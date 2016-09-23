//
// +build unit

package snapheka

import (
	"testing"
	"time"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHekaPlugin(t *testing.T) {
	Convey("Meta should return metadata for the plugin", t, func() {
		meta := Meta()
		So(meta.Name, ShouldResemble, pluginName)
		So(meta.Version, ShouldResemble, version)
		So(meta.Type, ShouldResemble, plugin.PublisherPluginType)
	})

	Convey("Create HekaPublisher", t, func() {
		op := NewHekaPublisher()
		Convey("So heka publisher should not be nil", func() {
			So(op, ShouldNotBeNil)
		})
		Convey("So heka publisher should be of heka type", func() {
			So(op, ShouldHaveSameTypeAs, &hekaPublisher{})
		})
		Convey("op.GetConfigPolicy() should return a config policy", func() {
			configPolicy, _ := op.GetConfigPolicy()
			Convey("So config policy should not be nil", func() {
				So(configPolicy, ShouldNotBeNil)
			})
			Convey("So config policy should be a cpolicy.ConfigPolicy", func() {
				So(configPolicy, ShouldHaveSameTypeAs, &cpolicy.ConfigPolicy{})
			})
			testConfig := make(map[string]ctypes.ConfigValue)
			testConfig["host"] = ctypes.ConfigValueStr{Value: "localhost"}
			testConfig["port"] = ctypes.ConfigValueInt{Value: 6565}
			cfg, errs := configPolicy.Get([]string{vendor, pluginName}).Process(testConfig)
			Convey("So config policy should process testConfig and return a config", func() {
				So(cfg, ShouldNotBeNil)
			})
			Convey("So testConfig processing should return no errors", func() {
				So(errs.HasErrors(), ShouldBeFalse)
			})
			testConfig["port"] = ctypes.ConfigValueStr{Value: "6565"}
			cfg, errs = configPolicy.Get([]string{vendor, pluginName}).Process(testConfig)
			Convey("So config policy should not return a config after processing invalid testConfig", func() {
				So(cfg, ShouldBeNil)
			})
			Convey("So testConfig processing should return errors", func() {
				So(errs.HasErrors(), ShouldBeTrue)
			})
		})
	})

	Convey("Create SnapHekaClient", t, func() {
		Convey("The SnapHekaClient should not be nil", func() {
			client, _ := NewSnapHekaClient("tcp://localhost:5600", "")
			So(client, ShouldNotBeNil)
		})
		Convey("Testing createHekaMessage", func() {
			var dynElt, staElt core.NamespaceElement
			tags := map[string]string{"tag_key": "tag_val"}
			// namespace is foo.<bar>.<name>.baz
			namespace := core.NewNamespace("foo")
			dynElt = core.NamespaceElement{
				Name:        "bar",
				Description: "",
				Value:       "bar_val"}
			namespace = append(namespace, dynElt)
			dynElt = core.NamespaceElement{
				Name:        "name",
				Description: "",
				Value:       "name_val"}
			namespace = append(namespace, dynElt)
			staElt = core.NamespaceElement{
				Name:        "",
				Description: "",
				Value:       "baz"}
			namespace = append(namespace, staElt)
			metric := *plugin.NewMetricType(namespace, time.Now(), tags, "some unit", 3.141)
			message, _ := createHekaMessage("some payload", metric, 1234, "host0")
			Convey("The Heka message should not be nil", func() {
				So(message, ShouldNotBeNil)
			})
			Convey("The Heka message should be as expected", func() {
				So(message.GetHostname(), ShouldEqual, "host0")
				So(message.GetLogger(), ShouldEqual, "snap.heka.logger")
				So(message.GetPayload(), ShouldEqual, "some payload")
				So(message.GetPid(), ShouldEqual, 1234)
				So(message.GetSeverity(), ShouldEqual, 6)
				So(message.GetType(), ShouldEqual, "snap.heka")
				fields := message.GetFields()
				So(fields, ShouldNotBeNil)
				So(fields[0].GetName(), ShouldEqual, "bar")
				So(fields[0].GetValue(), ShouldEqual, "bar_val")
				So(fields[1].GetName(), ShouldEqual, "name")
				So(fields[1].GetValue(), ShouldEqual, "name_val")
				So(fields[2].GetName(), ShouldEqual, "tag_key")
				So(fields[2].GetValue(), ShouldEqual, "tag_val")
				So(fields[3].GetName(), ShouldEqual, "dimensions")
				So(fields[3].GetValueString(), ShouldResemble, []string{"bar", "name", "tag_key"})
				So(fields[4].GetName(), ShouldEqual, "name")
				So(fields[4].GetValue(), ShouldEqual, "foo.baz")
				So(fields[5].GetName(), ShouldEqual, "value")
				So(fields[5].GetValue(), ShouldEqual, 3.141)
				So(fields[6].GetName(), ShouldEqual, "timestamp")
			})
		})
	})
}
