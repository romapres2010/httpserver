package bytespool

import (
	"sync"
	"sync/atomic"

	mylog "github.com/romapres2010/httpserver/log"
)

// Pool represent pooling of []byte
type Pool struct {
	cfg  *Config // конфигурационные параметры
	pool sync.Pool
}

// Config repsent BytesPool Service configurations
type Config struct {
	PooledSize int
}

// Counter represent a pool statistics for benchmarking
var (
	countGetBytes uint64 // количество запросов кэша
	countPutBytes uint64 // количество возвратов в кэша
	countNewBytes uint64 // количество создания нового объекта
)

// New create new BytesPool
func New(cfg *Config) *Pool {
	p := &Pool{
		cfg: cfg,
		pool: sync.Pool{
			New: func() interface{} {
				atomic.AddUint64(&countNewBytes, 1)
				return make([]byte, cfg.PooledSize)
			},
		},
	}
	return p
}

// GetBuf allocates a new []byte
func (p *Pool) GetBuf() []byte {
	atomic.AddUint64(&countGetBytes, 1)
	return p.pool.Get().([]byte)
}

// PutBuf return byte buf to cache
func (p *Pool) PutBuf(buf []byte) {
	size := cap(buf)
	if size < p.cfg.PooledSize { // не выгодно хранить маленькие буферы
		return
	}
	atomic.AddUint64(&countPutBytes, 1)
	p.pool.Put(buf[:0])
}

// PrintBytesPoolStats print statistics about bytes pool
func (p *Pool) PrintBytesPoolStats() {
	mylog.PrintfInfoMsg("Usage butes pool: countGet, countPut, countNew", countGetBytes, countPutBytes, countNewBytes)
}
