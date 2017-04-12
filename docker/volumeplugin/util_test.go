package volumeplugin

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

const jsonStr = `[{"Name":"qq","Mountpoint":"/var/lib/rancher/volumes/rancher-longhorn/qq","Status":{"name":"qq"}},{"Name":"zz","Mountpoint":"/var/lib/rancher/volumes/rancher-longhorn/zz","Status":{"name":"zz"}}]`

func TestToVols(t *testing.T) {
	assert := require.New(t)

	var data interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	assert.Nil(err)

	vols, err := toVols(data)
	assert.Equal("qq", vols[0].Name)
	assert.Equal("zz", vols[1].Name)
}
