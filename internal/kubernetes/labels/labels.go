package labels

import (
	"encoding/base32"
	"fmt"
	"strings"
)

const (
	// LabelKeySource is the label key that will be used to mark the generated
	// resources based on the original resource (ingress), this can be used to
	// listen on changes on these generated resources and reconcile.
	LabelKeySource = "bilrost.slok.dev/src"
)

// EncodeSourceLabelValue will generate the value of the source label
// it will encode in base32 the name and namespace of the original source.
func EncodeSourceLabelValue(ns, name string) string {
	if ns == "" {
		ns = "default"
	}

	id := fmt.Sprintf("%s/%s", ns, name)
	b32ID := base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(id))
	return strings.ToLower(b32ID)
}

// DecodeSourceLabelValue returns the original value that once was enconded
// as the value of the source label.
func DecodeSourceLabelValue(value string) (ns, name string, err error) {
	value = strings.ToUpper(value)

	id, err := base32.HexEncoding.WithPadding(base32.NoPadding).DecodeString(value)
	if err != nil {
		return "", "", fmt.Errorf("could not decode form base32: %w", err)
	}

	parts := strings.Split(string(id), "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("the lable values isn't composed of a ns and a name, got: %s", id)
	}

	ns = parts[0]
	name = parts[1]

	if ns == "" {
		ns = "default"
	}

	return ns, name, nil
}
