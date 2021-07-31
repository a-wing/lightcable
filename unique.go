package lightcable

import (
	"strconv"
	"sync/atomic"
)

var uniqueID uint64

func getUniqueID() string {
	return strconv.FormatUint(atomic.AddUint64(&uniqueID, 1), 10)
}
