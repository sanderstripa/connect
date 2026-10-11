package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/rpc"
	"reflect"
)

type NetworkRequest struct {
	Token  string
	Method string
	Args   []json.RawMessage
}
type NetworkReply struct {
	Result json.RawMessage
	Error  string
}
type networkClient struct {
	client *rpc.Client
	token  string
}

func (c *networkClient) call(method string, args []any, result any) error {
	req := NetworkRequest{Token: c.token, Method: method}
	for _, arg := range args {
		data, err := json.Marshal(arg)
		if err != nil {
			return err
		}
		req.Args = append(req.Args, data)
	}
	var reply NetworkReply
	if err := c.client.Call("Network.Invoke", req, &reply); err != nil {
		return fmt.Errorf("network helper: %w", err)
	}
	if reply.Error != "" {
		return errors.New(reply.Error)
	}
	if result != nil && len(reply.Result) != 0 {
		return json.Unmarshal(reply.Result, result)
	}
	return nil
}

// Explicitly restrict the privileged endpoint to the application API.
var networkMethods = map[string]bool{
	"GetConfig": true, "SaveConfig": true, "SetRoutingMode": true, "GetState": true,
	"StartProxy": true, "StopProxy": true, "RestartProxy": true, "UpsertNode": true,
	"DeleteNode": true, "SetActiveNode": true, "SwitchNode": true, "ImportShortLink": true,
	"ExportShortLink": true, "ProbeNode": true, "ProbeAllNodes": true, "AutoSelectLowestLatency": true,
	"SortNodesByName": true, "SortNodesByLatency": true, "DetectIPDirect": true, "DetectIPProxy": true,
	"GetLANProxyInfo": true, "StartReverseForwarder": true, "StopReverseForwarder": true,
	"GetLogs": true, "ExportLogs": true, "GetConnections": true, "CloseConnection": true,
	"CloseAllConnections": true, "GetUsageHistory": true, "EnsureCoreBinaries": true,
	"BuildInfo": true, "ValidateYAML": true, "OpenRuntimeDir": true, "OpenConfigPath": true,
	"TunHasPrivileges": true, "TunAcquirePrivileges": true, "TunDropPrivileges": true,
}

type NetworkRPC struct {
	app      *App
	token    string
	shutdown func()
}

func (n *NetworkRPC) Invoke(req NetworkRequest, reply *NetworkReply) error {
	if req.Token == "" || req.Token != n.token {
		return errors.New("unauthorized")
	}
	if req.Method == "Shutdown" {
		n.shutdown()
		return nil
	}
	if !networkMethods[req.Method] {
		return errors.New("unsupported network operation")
	}
	method := reflect.ValueOf(n.app).MethodByName(req.Method)
	if !method.IsValid() || method.Type().NumIn() != len(req.Args) {
		return errors.New("invalid network arguments")
	}
	args := make([]reflect.Value, len(req.Args))
	for i, raw := range req.Args {
		value := reflect.New(method.Type().In(i))
		if err := json.Unmarshal(raw, value.Interface()); err != nil {
			return err
		}
		args[i] = value.Elem()
	}
	values := method.Call(args)
	for _, value := range values {
		if value.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !value.IsNil() {
				reply.Error = value.Interface().(error).Error()
			}
		} else {
			data, err := json.Marshal(value.Interface())
			if err != nil {
				return err
			}
			reply.Result = data
		}
	}
	return nil
}
