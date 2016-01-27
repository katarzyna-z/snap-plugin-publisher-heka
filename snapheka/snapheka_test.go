//
// +build unit

package snapheka

import (
	"testing"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core/ctypes"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHekaPlugin(t *testing.T) {
	Convey("Meta should return metadata for the plugin", t, func() {
		meta := Meta()
		So(meta.Name, ShouldResemble, name)
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
			cfg, errs := configPolicy.Get([]string{""}).Process(testConfig)
			Convey("So config policy should process testConfig and return a config", func() {
				So(cfg, ShouldNotBeNil)
			})
			Convey("So testConfig processing should return no errors", func() {
				So(errs.HasErrors(), ShouldBeFalse)
			})
			testConfig["port"] = ctypes.ConfigValueStr{Value: "6565"}
			cfg, errs = configPolicy.Get([]string{""}).Process(testConfig)
			Convey("So config policy should not return a config after processing invalid testConfig", func() {
				So(cfg, ShouldBeNil)
			})
			Convey("So testConfig processing should return errors", func() {
				So(errs.HasErrors(), ShouldBeTrue)
			})
		})
	})
}
