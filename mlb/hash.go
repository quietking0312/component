package mlb

import (
	"hash/fnv"
	"strconv"
)

func Hash(key string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return h.Sum64()
}

func vnodeKey(nodeID string, idx int) string {
	return nodeID + "#" + strconv.Itoa(idx)
}
