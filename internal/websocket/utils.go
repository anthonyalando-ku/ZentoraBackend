// internal/websocket/utils.go
package websocket

import "encoding/json"

// mapToStruct converts interface{} to a specific struct using JSON marshaling
func mapToStruct(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}