package supervisor

import (
	"errors"
	"fmt"
	"github.com/kolo/xmlrpc"
	"net"
	"net/http"
	"strings"
)

const (
	apiVersion string = "3.0"
)

func makeParams(params ...interface{}) []interface{} {
	var ret []interface{}
	for _, param := range params {
		ret = append(ret, param)
	}
	return ret
}

type SupervisorState struct {
	StateCode int64
	StateName string
}

func newSupervisorState(result map[string]interface{}) *SupervisorState {
	state := new(SupervisorState)
	state.StateCode = result["statecode"].(int64)
	state.StateName = result["statename"].(string)
	return state
}

func (state SupervisorState) String() string {
	return fmt.Sprintf(`SupervisorState{%d, "%s"}`, state.StateCode, state.StateName)
}

type ProcessInfo struct {
	Name          string
	Description   string
	Group         string
	Start         int64
	Stop          int64
	Now           int64
	State         int64
	StateName     string
	SpawnErr      string
	ExitStatus    int64
	Logfile       string
	StdoutLogfile string
	StderrLogfile string
	PID           int64
}

func newProcessInfo(result map[string]interface{}) ProcessInfo {
	var processInfo ProcessInfo
	for key, value := range result {
		if value == nil {
			continue
		}
		switch key {
		case "name":
			processInfo.Name = value.(string)
		case "description":
			processInfo.Description = value.(string)
		case "group":
			processInfo.Group = value.(string)
		case "start":
			processInfo.Start = value.(int64)
		case "stop":
			processInfo.Stop = value.(int64)
		case "now":
			processInfo.Now = value.(int64)
		case "state":
			processInfo.State = value.(int64)
		case "statename":
			processInfo.StateName = value.(string)
		case "spawnerr":
			processInfo.SpawnErr = value.(string)
		case "exitstatus":
			processInfo.ExitStatus = value.(int64)
		case "logfile":
			processInfo.Logfile = value.(string)
		case "stdout_logfile":
			processInfo.StdoutLogfile = value.(string)
		case "stderr_logfile":
			processInfo.StderrLogfile = value.(string)
		case "pid":
			processInfo.PID = value.(int64)
		}
	}
	return processInfo
}

func (info ProcessInfo) String() string {
	return fmt.Sprintf(`ProcessInfo{"%s", %d, "%s"}`, info.Name, info.PID, info.StateName)
}

type ProcessStatus struct {
	Name        string
	Description string
	Group       string
	Status      int64
}

func newProcessStatus(result map[string]interface{}) ProcessStatus {
	return ProcessStatus{
		Name:        result["name"].(string),
		Description: result["description"].(string),
		Group:       result["group"].(string),
		Status:      result["status"].(int64),
	}
}

func (status ProcessStatus) String() string {
	return fmt.Sprintf(`ProcessStatus("%s", %d)`, status.Name, status.Status)
}

type ProcessTail struct {
	Log      string
	Offset   int64
	Overflow bool
}

func newProcessTail(result []interface{}) *ProcessTail {
	tail := new(ProcessTail)
	tail.Log = result[0].(string)
	tail.Offset = result[1].(int64)
	tail.Overflow = result[2].(bool)
	return tail
}

func (tail ProcessTail) String() string {
	return tail.Log
}

type ReloadInfo struct {
	Added, Changed, Removed []string
}

func newReloadInfo(results []interface{}) ReloadInfo {
	var info ReloadInfo

	for _, add := range results[0].([]interface{}) {
		info.Added = append(info.Added, add.(string))
	}
	for _, chg := range results[1].([]interface{}) {
		info.Changed = append(info.Changed, chg.(string))
	}
	for _, del := range results[2].([]interface{}) {
		info.Removed = append(info.Removed, del.(string))
	}

	return info
}

func (info ReloadInfo) String() string {
	return fmt.Sprintf("added: %d, changed: %d, removed: %d", len(info.Added), len(info.Changed), len(info.Removed))
}

type Client struct {
	RpcClient  *xmlrpc.Client
	ApiVersion string
}

func dialer(sock string) func(proto, addr string) (net.Conn, error) {
	return func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", sock)
	}
}

// NewClient creates a new supervisor RPC client.
func NewClient(url string) (client Client, err error) {
	var rpc *xmlrpc.Client

	var transport http.RoundTripper

	if strings.HasPrefix(url, "unix://") {
		var sock = strings.TrimPrefix(url, "unix://")
		if index := strings.Index(sock, ".sock"); index > 0 {
			url = "http://localhost:80" + sock[index+5:] //fake
			sock = sock[:index+5]
		}
		transport = &http.Transport{Dial: dialer(sock)}
	}

	if rpc, err = xmlrpc.NewClient(url, transport); err != nil {
		return
	}

	version := ""
	if err = rpc.Call("supervisor.getAPIVersion", nil, &version); err != nil {
		return
	}
	if version != apiVersion {
		err = errors.New(fmt.Sprintf("want Supervisor API version %s, got %s instead", apiVersion, version))
		return
	}
	client = Client{rpc, version}
	return
}

// Close the client.
func (client Client) Close() error {
	return client.RpcClient.Close()
}

// GetSupervisorVersion returns the Supervisor version we connect to.
func (client Client) GetSupervisorVersion() (version string, err error) {
	err = client.RpcClient.Call("supervisor.getSupervisorVersion", nil, &version)
	return
}

// GetIdentification returns the Supervisor ID string.
func (client Client) GetIdentification() (id string, err error) {
	err = client.RpcClient.Call("supervisor.getIdentification", nil, &id)
	return
}

// GetState returns the Supervisor process state.
func (client Client) GetState() (state *SupervisorState, err error) {
	result := make(map[string]interface{})
	if err = client.RpcClient.Call("supervisor.getState", nil, &result); err == nil {
		state = newSupervisorState(result)
	}
	return
}

// GetPID returns the Supervisor process PID.
func (client Client) GetPID() (pid int64, err error) {
	err = client.RpcClient.Call("supervisor.getPID", nil, &pid)
	return
}

// ClearLog clears the Supervisor process log.
func (client Client) ClearLog() (result bool, err error) {
	err = client.RpcClient.Call("supervisor.clearLog", nil, &result)
	return
}

// Shutdown shuts down the Supervisor process.
func (client Client) Shutdown() (result bool, err error) {
	err = client.RpcClient.Call("supervisor.shutdown", nil, &result)
	return
}

// Restart restarts the Supervisor process.
func (client Client) Restart() (result bool, err error) {
	err = client.RpcClient.Call("supervisor.restart", nil, &result)
	return
}

func (client Client) ReloadConfig() (info ReloadInfo, err error) {
	var results []interface{}
	if err = client.RpcClient.Call("supervisor.reloadConfig", nil, &results); err == nil {
		info = newReloadInfo(results[0].([]interface{}))
	}
	return
}

// GetProcessInfo retrieves information for a particular Supervisor process.
func (client Client) GetProcessInfo(name string) (info ProcessInfo, err error) {
	result := make(map[string]interface{})
	if err = client.RpcClient.Call("supervisor.getProcessInfo", name, &result); err == nil {
		info = newProcessInfo(result)
	}
	return
}

// GetAllProcessInfo retrieves information for all Supervisor processes.
func (client Client) GetAllProcessInfo() (info []ProcessInfo, err error) {
	var results []interface{}
	if err = client.RpcClient.Call("supervisor.getAllProcessInfo", nil, &results); err == nil {
		info = make([]ProcessInfo, len(results))
		for i, result := range results {
			info[i] = newProcessInfo(result.(map[string]interface{}))
		}
	}
	return
}

// StartProcess tells Supervisor to start the named process.
func (client Client) StartProcess(name string, wait bool) (result bool, err error) {
	params := makeParams(name, wait)
	err = client.RpcClient.Call("supervisor.startProcess", params, &result)
	return
}

// StopProcess tells Supervisor to stop the named process.
func (client Client) StopProcess(name string, wait bool) (result bool, err error) {
	params := makeParams(name, wait)
	err = client.RpcClient.Call("supervisor.stopProcess", params, &result)
	return
}

// StartAllProcesses tells Supervisor to start all stopped processes.
func (client Client) StartAllProcesses(wait bool) (info []ProcessStatus, err error) {
	var results []interface{}
	if err = client.RpcClient.Call("supervisor.startAllProcesses", wait, &results); err == nil {
		info = make([]ProcessStatus, len(results))
		for i, result := range results {
			info[i] = newProcessStatus(result.(map[string]interface{}))
		}
	}
	return
}

// StopAllProcesses teslls Supervisor to stop all running processes.
func (client Client) StopAllProcesses(wait bool) (info []ProcessStatus, err error) {
	var results []interface{}
	if err = client.RpcClient.Call("supervisor.stopAllProcesses", wait, &results); err == nil {
		info = make([]ProcessStatus, len(results))
		for i, result := range results {
			info[i] = newProcessStatus(result.(map[string]interface{}))
		}
	}
	return
}

// StartProcessGroup tells Supervisor to start all stopped processes in the named group.
func (client Client) StartProcessGroup(name string, wait bool) (result bool, err error) {
	params := makeParams(name, wait)
	err = client.RpcClient.Call("supervisor.startProcessGroup", params, &result)
	return
}

// StopProcessGroup tells Supervisor to start all stopped processes in the named group.
func (client Client) StopProcessGroup(name string, wait bool) (result bool, err error) {
	params := makeParams(name, wait)
	err = client.RpcClient.Call("supervisor.stopProcessGroup", params, &result)
	return
}

// SendProcessStdin send data to the stdin of a running process.
func (client Client) SendProcessStdin(name string, chars string) (result bool, err error) {
	params := makeParams(name, chars)
	err = client.RpcClient.Call("supervisor.sendProcessStdin", params, &result)
	return
}

// SendRemoteCommEvent sends an event to Supervisor processes listening to RemoveCommunicationEvents..
func (client Client) SendRemoteCommEvent(typeKey string, data string) (result bool, err error) {
	params := makeParams(typeKey, data)
	err = client.RpcClient.Call("supervisor.sendRemoteCommEvent", params, &result)
	return
}

// AddProcessGroup adds a configured process group to Supervisor.
func (client Client) AddProcessGroup(name string) (result bool, err error) {
	err = client.RpcClient.Call("supervisor.addProcessGroup", name, &result)
	return
}

// RemoveProcessGroup removes a configured process group from Supervisor.
func (client Client) RemoveProcessGroup(name string) (result bool, err error) {
	err = client.RpcClient.Call("supervisor.removeProcessGroup", name, &result)
	return
}

// ReadLog reads the Supervisor process log.
func (client Client) ReadLog(offset int64, length int64) (log string, err error) {
	params := makeParams(offset, length)
	err = client.RpcClient.Call("supervisor.readLog", params, &log)
	return
}

// ReadProcessStdoutLog reads the stdout log for the named process.
func (client Client) ReadProcessStdoutLog(name string, offset int64, length int64) (log string, err error) {
	params := makeParams(name, offset, length)
	err = client.RpcClient.Call("supervisor.readProcessStdoutLog", params, &log)
	return
}

// ReadProcessStderrLog reads the stderr log for the named process.
func (client Client) ReadProcessStderrLog(name string, offset int64, length int64) (log string, err error) {
	params := makeParams(name, offset, length)
	err = client.RpcClient.Call("supervisor.readProcessStderrLog", params, &log)
	return
}

// TailProcessStdoutLog reads the stdout log for the named process.
func (client Client) TailProcessStdoutLog(name string, offset int64, length int64) (tail *ProcessTail, err error) {
	params := makeParams(name, offset, length)
	result := make([]interface{}, 0, 3)
	if err = client.RpcClient.Call("supervisor.tailProcessStdoutLog", params, &result); err == nil {
		tail = newProcessTail(result)
	}
	return
}

// TailProcessStderrLog reads the stderr log for the named process.
func (client Client) TailProcessStderrLog(name string, offset int64, length int64) (tail *ProcessTail, err error) {
	params := makeParams(name, offset, length)
	result := make([]interface{}, 0, 3)
	if err = client.RpcClient.Call("supervisor.tailProcessStderrLog", params, &result); err == nil {
		tail = newProcessTail(result)
	}
	return
}

// ClearProcessLogs clears all logs for the named process.
func (client Client) ClearProcessLogs(name string) (result bool, err error) {
	err = client.RpcClient.Call("supervisor.clearProcessLogs", name, &result)
	return
}

// ClearAllProcessLogs clears all logs all processes.
func (client Client) ClearAllProcessLogs(name string) (result bool, err error) {
	err = client.RpcClient.Call("supervisor.clearAllProcessLogs", name, &result)
	return
}
