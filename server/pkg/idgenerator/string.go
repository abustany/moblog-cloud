package idgenerator

import (
	"strconv"
	"sync/atomic"
)

type StringIdGenerator struct {
	lastID uint64
}

func (g *StringIdGenerator) Next() string {
	return strconv.FormatUint(atomic.AddUint64(&g.lastID, 1), 10)
}
