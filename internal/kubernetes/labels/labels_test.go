package labels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slok/bilrost/internal/kubernetes/labels"
)

func TestEncodeSourceLabelValue(t *testing.T) {
	tests := map[string]struct {
		ns   string
		name string
		exp  string
	}{
		"A regular value composed of a namespace and a name should have a valid label value.": {
			ns:   "test-ns",
			name: "test-name",
			exp:  "ehin6t1ddppiut35edq2qrj1dlig",
		},

		"A regular value composed of a default ns and a name should have a valid label value.": {
			ns:   "default",
			name: "test-name",
			exp:  "chimcobldhq2ut35edq2qrj1dlig",
		},

		"A regular value composed of a default ns as empty and a name should have a valid label value.": {
			ns:   "",
			name: "test-name",
			exp:  "chimcobldhq2ut35edq2qrj1dlig",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := labels.EncodeSourceLabelValue(test.ns, test.name)
			assert.Equal(t, test.exp, got)
		})
	}
}

func TestDecodeSourceLabelValue(t *testing.T) {
	tests := map[string]struct {
		value   string
		expNs   string
		expName string
		expErr  bool
	}{
		"A correct value should be decoded correctly": {
			value:   "ehin6t1ddppiut35edq2qrj1dlig",
			expNs:   "test-ns",
			expName: "test-name",
		},

		"An incorrect encoded value should error.": {
			value:  "this should fail",
			expErr: true,
		},

		"A value with a wrong ns/name schema should fail.": {
			value:  "ehin6t1ddpgmqp8", // 'test-name' in b32
			expErr: true,
		},

		"A value with empty namespace should be treated as default.": {
			value:   "5tq6asrk5ln62rb5", // '/test-name' in b32
			expNs:   "default",
			expName: "test-name",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotNs, gotName, err := labels.DecodeSourceLabelValue(test.value)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expNs, gotNs)
				assert.Equal(test.expName, gotName)
			}
		})
	}
}
