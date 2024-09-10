package persist

import (
	"context"

	"github.com/fxamacker/cbor/v2"

	"git.defalsify.org/vise.git/db"
	"git.defalsify.org/vise.git/state"
	"git.defalsify.org/vise.git/cache"
)

// Persister abstracts storage and retrieval of state and cache.
type Persister struct {
	State *state.State
	Memory *cache.Cache
	ctx context.Context
	db db.Db
	flush bool
}

// NewPersister creates a new Persister instance.
func NewPersister(db db.Db) *Persister {
	return &Persister{
		db: db,
		ctx: context.Background(),
	}
}

// WithSession is a chainable function that sets the current golang context of the persister.
func(p *Persister) WithContext(ctx context.Context) *Persister {
	p.ctx = ctx
	return p
}

// WithSession is a chainable function that sets the current session context of the persister.
func(p *Persister) WithSession(sessionId string) *Persister {
	p.db.SetSession(sessionId)
	return p
}


// WithContent is a chainable function that sets a current State and Cache object.
//
// This method is normally called before Serialize / Save.
func(p *Persister) WithContent(st *state.State, ca *cache.Cache) *Persister {
	p.State = st
	p.Memory = ca
	return p
}

// WithFlush is a chainable function that instructs the persister to flush its memory and state
// after successful Save.
func(p *Persister) WithFlush() *Persister {
	p.flush = true
	return p
}

// Invalid checks if the underlying state has been invalidated.
//
// An invalid state will cause Save to panic.
func(p *Persister) Invalid() bool {
	return p.GetState().Invalid() || p.GetMemory().Invalid()
}

// GetState returns the state enclosed by the Persister.
func(p *Persister) GetState() *state.State {
	return p.State
}

// GetMemory returns the cache (memory) enclosed by the Persister.
func(p *Persister) GetMemory() cache.Memory {
	return p.Memory
}

// Serialize encodes the state and cache into byte form for storage.
func(p *Persister) Serialize() ([]byte, error) {
	return cbor.Marshal(p)
}

// Deserialize decodes the state and cache from storage, and applies them to the persister.
func(p *Persister) Deserialize(b []byte) error {
	err := cbor.Unmarshal(b, p)
	return err
}

// Save perists the state and cache to the db.Db backend.
//
// If save is successful and WithFlush() has been called, the state and memory
// will be empty when the method returns.
func(p *Persister) Save(key string) error {
	if p.Invalid() {
		panic("persister has been invalidated")
	}
	b, err := p.Serialize()
	if err != nil {
		return err
	}
	p.db.SetPrefix(db.DATATYPE_STATE)
	logg.Debugf("saving state and cache", "key", key, "state", p.State)
	err = p.db.Put(p.ctx, []byte(key), b)
	if err != nil {
		return err
	}
	if p.flush {
		logg.Tracef("state and cache flushed from persister")
		p.Memory.Reset()
		p.State = p.State.CloneEmpty()
	}
	return nil
}

// Load retrieves state and cache from the db.Db backend.
func(p *Persister) Load(key string) error {
	p.db.SetPrefix(db.DATATYPE_STATE)
	b, err := p.db.Get(p.ctx, []byte(key))
	if err != nil {
		return err
	}
	err = p.Deserialize(b)
	if err != nil {
		return err
	}
	logg.Debugf("loaded state and cache", "key", key, "state", p.State)
	return nil
}
