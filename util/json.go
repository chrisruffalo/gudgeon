package util

import jsoniter "github.com/json-iterator/go"

// write all metrics out to encoder
var Json = jsoniter.Config{
	EscapeHTML:                    false,
	MarshalFloatWith6Digits:       true,
	ObjectFieldMustBeSimpleString: true,
	SortMapKeys:                   false,
	ValidateJsonRawMessage:        true,
	DisallowUnknownFields:         false,
}.Froze()
