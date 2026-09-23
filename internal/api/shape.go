package api

import "encoding/json"

// UnmarshalObject decodes a resource that the API sends under key or as the bare object.
func UnmarshalObject(data []byte, key string, v interface{}) error {
	fields, err := topLevelFields(data)
	if err != nil {
		return err
	}
	if wrapped, ok := fields[key]; ok {
		return json.Unmarshal(wrapped, v)
	}
	return json.Unmarshal(data, v)
}

// UnmarshalList decodes a list that the API sends under key or as {"object": "list", "data": [...]}.
func UnmarshalList(data []byte, key string, v interface{}) error {
	fields, err := topLevelFields(data)
	if err != nil {
		return err
	}
	if items, ok := fields["data"]; ok {
		return json.Unmarshal(items, v)
	}
	if items, ok := fields[key]; ok {
		return json.Unmarshal(items, v)
	}
	return nil
}

func topLevelFields(data []byte) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}
