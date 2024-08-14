package helpers

import (
	"github.com/threatwinds/logger"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func ToString(object protoreflect.ProtoMessage) (*string, *logger.Error) {
	objectBytes, err := protojson.Marshal(object)
	if err != nil {
		return nil, Logger().ErrorF(err.Error())
	}

	objectString := string(objectBytes)

	return &objectString, nil
}

func ToObject(str *string, object protoreflect.ProtoMessage) *logger.Error {
	err := protojson.Unmarshal([]byte(*str), object)
	if err != nil {
		return Logger().ErrorF(err.Error())
	}

	return nil
}
