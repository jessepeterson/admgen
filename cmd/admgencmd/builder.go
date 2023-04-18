package main

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
)

type jenBuilder struct {
	file *File
}

func newJenBuilder(pkgName string) *jenBuilder {
	j := &jenBuilder{file: NewFile(pkgName)}
	j.file.PackageComment("Code generated by \"admgencmd\"; DO NOT EDIT.")
	return j
}

var commandUUIDKey = Key{
	Key:      "CommandUUID",
	Type:     "<string>",
	Presence: "required",
}

func (j *jenBuilder) walkCommand(keys []Key, name string) {
	// create a "const" string of the RequestType for the command
	j.file.Const().Id(name + "RequestType").Op("=").Lit(name)

	requestTypeKey := Key{
		Key:            "RequestType",
		Type:           "<string>",
		Presence:       "required",
		Content:        "must be set to \"" + name + "\"",
		includeContent: true,
	}

	networkTetherKey := Key{
		Key:      "RequestRequiresNetworkTether",
		Type:     "<boolean>",
		Presence: "optional",
	}

	// insert the RequestType and 'NetworkTether fields into the keys.
	// these aren't specified in the schema.
	keys = append(keys, requestTypeKey, networkTetherKey)

	// the command "payload" is the actual command data, one level up
	// from the command.
	payload := Key{
		Key:         name + "Payload",
		Type:        "<dictionary>",
		SubKeys:     keys,
		keyOverride: "Command",
		Presence:    "required",
	}

	// finally put together the actual base-level MDM command struct
	cmd := Key{
		Key:     name + "Command",
		Type:    "<dictionary>",
		SubKeys: []Key{payload, commandUUIDKey},
	}

	// Go (hah) convert it to code now
	j.handleKey(cmd, "")

	// create a helper function to instantiate our command with the correct RequestType
	j.file.Comment("New" + cmd.Key + " creates a new \"" + name + "\" Apple MDM command.")
	j.file.Func().Id("New" + cmd.Key).Params().Op("*").Id(cmd.Key).Block(
		Return(Op("&").Id(cmd.Key).Values(Dict{
			Id("Command"): Id(payload.Key).Values(Dict{
				Id("RequestType"): Id(name + "RequestType"),
			}),
		})),
	)
}

func (j *jenBuilder) walkResponse(keys []Key, name string) {
	statusKey := Key{
		Key:      "Status",
		Type:     "<string>",
		Presence: "required",
	}
	keys = append(keys,
		commandUUIDKey,
		statusKey,
		// TODO:
		//
		// EnrollmentID
		// EnrollmentUserID
		// ErrorChain
		// NotOnConsole
		// UDID
		// UserID
		// UserLongName
		// UserShortName
	)
	response := Key{
		Key:     name + "Response",
		SubKeys: keys,
		Type:    "<dictionary>",
	}
	j.handleKey(response, "")
}

func (j *jenBuilder) handleKey(key Key, parentType string) (s *Statement, comment string) {
	switch key.Type {
	case "<string>":
		s = String()
	case "<integer>":
		s = Int()
	case "<boolean>":
		s = Bool()
	case "<real>":
		s = Float64()
	case "<data>":
		s = Index().Byte()
	case "<date>":
		s = Qual("time", "Time")
	case "<dictionary>":
		s, comment = j.handleDict(key)
	case "<array>":
		s, comment = j.handleArray(key)
		s = Index().Add(s)
	default:
		s = Interface()
		comment = "unknown type: " + key.Type
	}
	if parentType != "<array>" && s != nil && key.Presence != "required" {
		s = Op("*").Add(s)
	}
	return
}

func (j *jenBuilder) handleArray(key Key) (s *Statement, comment string) {
	keys := key.SubKeys
	if len(keys) < 1 {
		return Interface(), "missing array keys in schema"
	}
	keyType := keys[0].Type
	for _, key := range keys[1:] {
		if key.Type != keyType {
			// return an interface{} as we seem to have mismatched types within our array
			return Interface(), "mismatched array types in schema"
		}
	}

	s, comment = j.handleKey(keys[0], key.Type)
	if len(keys) == 1 && keys[0].Type != "<dictionary>" && len(keys[0].SubKeys) > 0 {
		// if our single key is a scalar type and we have subkeys
		// then the the subkeys describe actual array values
		if comment != "" {
			comment += ", "
		}
		comment += fmt.Sprintf("%d array values defined in schema", len(keys[0].SubKeys))
	}
	return
}

func (j *jenBuilder) handleDict(key Key) (s *Statement, comment string) {
	var fields []Code
	for _, k := range key.SubKeys {
		s, comment := j.handleKey(k, key.Type)
		if s == nil {
			panic("handleKey should not have returned nil")
		}
		fieldName := normalizeFieldName(k.Key)
		if k.keyOverride != "" {
			fieldName = k.keyOverride
		}
		jenField := Id(fieldName).Add(s)
		var tag string
		if k.keyOverride == "" && (k.Key != fieldName) {
			tag = k.Key
		}
		if k.Presence == "optional" {
			tag += ",omitempty"
		}
		if tag != "" {
			jenField.Tag(map[string]string{"plist": tag})
		}
		if k.includeContent {
			if comment != "" {
				comment += ", "
			}
			comment += k.Content
		}
		if comment != "" {
			jenField.Comment(comment)
		}
		fields = append(fields, jenField)
	}
	// create a new struct in the file with fields
	j.file.Type().Id(key.Key).Struct(fields...)
	return Id(key.Key), ""
}

func strip(s string) string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') ||
			('0' <= b && b <= '9') {
			result.WriteByte(b)
		}
	}
	return result.String()
}

func normalizeFieldName(s string) string {
	s = strings.ToUpper(s[0:1]) + s[1:]
	return strip(s)
}
