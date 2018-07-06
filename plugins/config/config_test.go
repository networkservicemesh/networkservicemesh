//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config_test

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/ligato/networkservicemesh/plugins/config"
	"github.com/ligato/networkservicemesh/utils/command"
	. "github.com/onsi/gomega"
)

type Config1 struct {
	One string
	Two string
}

func TestLoadConfigDefault(t *testing.T) {
	RegisterTestingT(t)
	cmd := &cobra.Command{Use: "command"}
	command.SetRootCmd(cmd)
	plugin := config.NewPlugin()
	Expect(plugin).NotTo(BeNil())
	Expect(plugin.Deps.Name).To(Equal(config.DefaultName))
	Expect(plugin.Deps.DefaultConfig).ToNot(BeNil())
	Expect(plugin.Deps.Cmd).To(Equal(cmd))
	err := plugin.Init()
	Expect(err).To(BeNil())
	config := plugin.LoadConfig()
	Expect(config).NotTo(BeNil())
	err = plugin.Close()
	Expect(err).To(BeNil())
}

func TestLoadConfigNonDefault(t *testing.T) {
	RegisterTestingT(t)
	cmd := &cobra.Command{Use: "command"}
	command.SetRootCmd(cmd)
	cmd2 := &cobra.Command{Use: "command2"}
	dcfg := &Config1{}
	name := "config2"
	plugin := config.NewPlugin(config.UseDeps(&config.Deps{
		Name:          name,
		Cmd:           cmd2,
		DefaultConfig: dcfg,
	}))
	Expect(plugin).NotTo(BeNil())
	Expect(plugin.Deps.Name).To(Equal(name))
	Expect(plugin.Deps.DefaultConfig).To(Equal(dcfg))
	Expect(plugin.Deps.Cmd).To(Equal(cmd2))
	err := plugin.Init()
	Expect(err).To(BeNil())
	config := plugin.LoadConfig()
	Expect(config).To(Equal(dcfg))
	err = plugin.Close()
	Expect(err).To(BeNil())
}

func TestSharedPlugin(t *testing.T) {
	RegisterTestingT(t)
	cmd := &cobra.Command{Use: "command"}
	command.SetRootCmd(cmd)
	cmd2 := &cobra.Command{Use: "command2"}
	dcfg := &Config1{}
	name := "config3"
	deps := &config.Deps{
		Name:          name,
		Cmd:           cmd2,
		DefaultConfig: dcfg,
	}
	option := config.UseDeps(deps)
	plugin := config.SharedPlugin(option)
	Expect(plugin).NotTo(BeNil())
	Expect(plugin.Deps.Name).To(Equal(name))
	Expect(plugin.Deps.DefaultConfig).To(Equal(dcfg))
	Expect(plugin.Deps.Cmd).To(Equal(cmd2))
	err := plugin.Init()
	Expect(err).To(BeNil())
	plugin2 := config.SharedPlugin(option)
	Expect(plugin2).To(Equal(plugin))
	err = plugin2.Init()
	Expect(err).To(BeNil())
	err = plugin.Close()
	Expect(err).To(BeNil())
	Expect(plugin.IsClosed()).To(BeFalse())
	err = plugin2.Close()
	Expect(err).To(BeNil())
	Expect(plugin2.IsClosed()).To(BeTrue())

	// Make sure the plugin isn't still lingering
	// After its final close
	plugin3 := config.SharedPlugin(option)
	Expect(plugin3).NotTo(BeNil())
	Expect(plugin3.Deps.Name).To(Equal(name))
	Expect(plugin3.Deps.DefaultConfig).To(Equal(dcfg))
	Expect(plugin3).NotTo(Equal(plugin))
	Expect(plugin3.Deps.Cmd).To(Equal(cmd2))
	err = plugin3.Init()
	Expect(err).To(BeNil())
}

func TestLoadConfigFile(t *testing.T) {
	RegisterTestingT(t)
	cmd := &cobra.Command{Use: "command3"}
	command.SetRootCmd(cmd)
	dcfg := &Config1{One: "NotOneValue", Two: "NotTwoValue"}
	name := "config4"
	plugin := config.NewPlugin(config.UseDeps(&config.Deps{
		Name:          name,
		DefaultConfig: dcfg,
	}))
	Expect(plugin).NotTo(BeNil())
	Expect(plugin.Deps.Name).To(Equal(name))
	Expect(plugin.Deps.DefaultConfig).To(Equal(dcfg))
	Expect(plugin.Deps.Cmd).To(Equal(cmd))
	err := plugin.Init()
	Expect(err).To(BeNil())
	config := plugin.LoadConfig().(*Config1)
	Expect(config.One).To(Equal("OneValue"))
	Expect(config.Two).To(Equal("TwoValue"))

}

func TestLoadConfigFileMissingKey(t *testing.T) {
	RegisterTestingT(t)
	cmd := &cobra.Command{Use: "command3"}
	command.SetRootCmd(cmd)
	dcfg := &Config1{One: "NotOneValue", Two: "NotTwoValue"}
	name := "config5"
	plugin := config.NewPlugin(config.UseDeps(&config.Deps{
		Name:          name,
		DefaultConfig: dcfg,
	}))
	Expect(plugin).NotTo(BeNil())
	Expect(plugin.Deps.Name).To(Equal(name))
	Expect(plugin.Deps.DefaultConfig).To(Equal(dcfg))
	Expect(plugin.Deps.Cmd).To(Equal(cmd))
	err := plugin.Init()
	Expect(err).To(BeNil())
	config := plugin.LoadConfig().(*Config1)
	Expect(config.One).To(Equal(dcfg.One))
	Expect(config.Two).To(Equal(dcfg.Two))

}
