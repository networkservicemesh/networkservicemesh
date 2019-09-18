package nseregistry

import (
	"bufio"
	"encoding/base64"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

const (
	ClientRegistered = "CLE"
	NSERegistered    = "NSE"
)

type NSERegistry struct {
	lock sync.Mutex
	file string
}

type NSEEntry struct {
	Workspace string
	NseReg    *registry.NSERegistration
}

func NewNSERegistry(file string) *NSERegistry {
	return &NSERegistry{file: file}
}

func store(values ...string) string {
	res := ""
	for _, s := range values {
		if len(res) > 0 {
			res += "\t"
		}
		ss := strings.Replace(s, "\t", "\\t,", -1)
		ss = strings.Replace(ss, "\n", "\\n,", -1) // Just in case
		ss = strings.Replace(ss, "\r", "\\r,", -1) // Just in case
		res += ss
	}
	return res
}
func unescape(s string) string {
	s = strings.Replace(s, "\\t", "\t", -1)
	s = strings.Replace(s, "\\n", "\n", -1)
	s = strings.Replace(s, "\\r", "\r", -1)
	return s
}
func restore(inputS string) []string {
	ts := strings.TrimSpace(inputS)
	segments := strings.SplitN(ts, "\t", -1)
	for idx, sm := range segments {
		segments[idx] = unescape(sm)
	}
	return segments
}

/**
We adding a registration for client.
*/
func (reg *NSERegistry) writeLine(op ...string) error {
	reg.lock.Lock()
	defer reg.lock.Unlock()
	f, err := os.OpenFile(reg.file, os.O_APPEND|os.O_WRONLY|os.O_SYNC|os.O_CREATE, 0600)
	if err != nil {
		logrus.Errorf("Failed to store Client information")
		return err
	}

	defer f.Close()

	if _, err = f.WriteString(store(op...) + "\n"); err != nil {
		return err
	}
	_ = f.Sync()
	return nil
}
func (reg *NSERegistry) AppendClientRequest(workspace string) error {
	return reg.writeLine(ClientRegistered, workspace)
}

func (reg *NSERegistry) AppendNSERegRequest(workspace string, nseReg *registry.NSERegistration) error {
	return reg.writeLine(NSERegistered, nseReg.GetNetworkServiceEndpoint().GetName(), workspace, reg.storeNSEBase64(nseReg)) // Few workspaces could contain few NSEs
}

func (reg *NSERegistry) storeNSEBase64(nseReg *registry.NSERegistration) string {
	bytes, err := proto.Marshal(nseReg)
	if err != nil {
		logrus.Errorf("Failed to serialize NSE passed %v", err)
	}
	strData := base64.StdEncoding.EncodeToString(bytes)
	return strData
}

func (reg *NSERegistry) DeleteNSE(endpointid string) error {
	reg.lock.Lock()
	defer reg.lock.Unlock()
	clients, nses, err := reg.loadRegistry()
	if err != nil {
		return err
	}
	delete(nses, endpointid)

	return reg.Save(clients, nses)
}

/**
Delete client workspace and all NSEs registered.
*/
func (reg *NSERegistry) DeleteClient(workspace string) error {
	reg.lock.Lock()
	defer reg.lock.Unlock()
	clients, nses, err := reg.loadRegistry()
	if err != nil {
		return err
	}
	for idx, ws := range clients {
		if ws == workspace {
			clients = append(clients[:idx], clients[idx+1:]...)
		}
	}

	for endpointId, entry := range nses {
		if entry.Workspace == workspace {
			delete(nses, endpointId)
		}
	}

	return reg.Save(clients, nses)
}

func (reg *NSERegistry) LoadRegistry() (clients []string, nses map[string]NSEEntry, err error) {
	reg.lock.Lock()
	defer reg.lock.Unlock()
	return reg.loadRegistry()
}

func (reg *NSERegistry) loadRegistry() (clients []string, nses map[string]NSEEntry, res_err error) {
	nses = map[string]NSEEntry{}

	f, err := os.OpenFile(reg.file, os.O_RDONLY, 0600)
	if err != nil {
		logrus.Infof("No stored registry file exists")
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		r, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				res_err = err
				return
			}
			// End of file
			break
		}
		values := restore(r)
		logrus.Printf("Values: %v", values)
		if len(values) > 0 {
			if values[0] == ClientRegistered && len(values) == 2 {
				clients = append(clients, values[1])
			} else if values[0] == NSERegistered && len(values) == 4 {
				entry := NSEEntry{}
				entry.Workspace = values[2]
				strData := values[3]
				bytes, err := base64.StdEncoding.DecodeString(strData)
				if err != nil {
					logrus.Errorf("Failed to decode NSE registration %v", err)
				}
				nseReg := &registry.NSERegistration{}
				err = proto.Unmarshal(bytes, nseReg)
				if err != nil {
					logrus.Errorf("Failed to decode nse registration message %v", err)
				}
				entry.NseReg = nseReg
				nses[values[1]] = entry
			} else {
				logrus.Errorf("Unknown registry file line: %v", r)
			}
		}
	}
	logrus.Infof("Clients: %v", clients)
	logrus.Infof("NSEs: %v", nses)

	return
}

/**
Saves memory model info file
*/
func (reg *NSERegistry) Save(clients []string, nses map[string]NSEEntry) error {
	tmpFile := reg.file + "_tmp"
	f, err := os.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		logrus.Errorf("Failed to store Client information")
		return err
	}

	for _, workspace := range clients {
		if _, err = f.WriteString(store(ClientRegistered, workspace) + "\n"); err != nil {
			return err
		}
	}

	for endpointId, entry := range nses {
		if _, err = f.WriteString(store(NSERegistered, endpointId, entry.Workspace, reg.storeNSEBase64(entry.NseReg)) + "\n"); err != nil {
			return err
		}
	}
	if err = f.Close(); err != nil {
		return err
	}
	// Now we need to replace existing file with new one.
	return os.Rename(tmpFile, reg.file)
}

func (reg *NSERegistry) Delete() {
	_ = os.Remove(reg.file)
}
