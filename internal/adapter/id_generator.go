package adapter

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
)

// IDGenerator allows custom generators for TraceID and SpanID.
type IDGenerator interface {
	NewTraceID() TraceID
	NewSpanID() SpanID
}

type randomIDGenerator struct {
	sync.Mutex
	randSource *rand.Rand
}

var _ IDGenerator = &randomIDGenerator{}

// NewSpanID returns a non-zero adapter ID from a randomly-chosen sequence.
func (gen *randomIDGenerator) NewSpanID() SpanID {
	gen.Lock()
	defer gen.Unlock()
	sid := SpanID{}
	gen.randSource.Read(sid[:])
	return sid
}

// NewIDs returns a non-zero trace ID and a non-zero adapter ID from a
// randomly-chosen sequence.
func (gen *randomIDGenerator) NewTraceID() TraceID {
	gen.Lock()
	defer gen.Unlock()
	tid := TraceID{}
	gen.randSource.Read(tid[:])
	return tid
}

func defaultIDGenerator() IDGenerator {
	gen := &randomIDGenerator{}
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	gen.randSource = rand.New(rand.NewSource(rngSeed))
	return gen
}
