package pixie

import (
	"github.com/stretchr/testify/assert"
	"px.dev/pxapi/proto/uuidpb"
	"px.dev/pxapi/utils"
	"testing"
)

func TestGetScriptClusterIdsAsString(t *testing.T) {
	assert.Equal(t, "", getClusterIdsAsString([]*uuidpb.UUID{}))

	assert.Equal(t, "b8749d5b-3352-4a0c-92ef-4a1479464b74", getClusterIdsAsString([]*uuidpb.UUID{
		utils.ProtoFromUUIDStrOrNil("b8749d5b-3352-4a0c-92ef-4a1479464b74"),
	}))

	assert.Equal(t, "b8749d5b-3352-4a0c-92ef-4a1479464b74,94fb8941-d353-43e0-b3e1-248f941c3af6", getClusterIdsAsString([]*uuidpb.UUID{
		utils.ProtoFromUUIDStrOrNil("b8749d5b-3352-4a0c-92ef-4a1479464b74"),
		utils.ProtoFromUUIDStrOrNil("94fb8941-d353-43e0-b3e1-248f941c3af6"),
	}))
}
