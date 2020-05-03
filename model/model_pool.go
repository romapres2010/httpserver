package model

import (
	"sync"
	"sync/atomic"

	mylog "github.com/romapres2010/httpserver/log"
)

// represent a pool statistics for benchmarking
var (
	countGet uint64 // количество запросов кэша
	countPut uint64 // количество возвратов в кэша
	countNew uint64 // количество создания нового объекта
)

// PrintModelPoolStats print pool statistics
func PrintModelPoolStats() {
	mylog.PrintfInfoMsg("Usage model pool: countGet, countPut, countNew", countGet, countPut, countNew)
}

// deptsPool represent depts pooling
var deptsPool = sync.Pool{
	New: func() interface{} {
		atomic.AddUint64(&countNew, 1)
		return new(Dept)
	},
}

// empsPool represent emps pooling
var empsPool = sync.Pool{
	New: func() interface{} {
		atomic.AddUint64(&countNew, 1)
		return new(Emp)
	},
}

// empSlicePool represent emps Slice pooling
var empSlicePool = sync.Pool{
	New: func() interface{} {
		v := make([]*Emp, 0)
		atomic.AddUint64(&countNew, 1)
		return EmpSlice(v)
	},
}

// Reset reset all fields in structure - use for sync.Pool
func (p *Dept) Reset() {
	p.Deptno = 0
	p.Dname = ""
	p.Loc.String = ""
	p.Loc.Valid = false
	if p.Emps != nil {
		EmpSlice(p.Emps).Reset()
	}
	p.Emps = nil
}

// Reset reset all fields in structure - use for sync.Pool
func (p *Emp) Reset() {
	p.Empno = 0
	p.Ename.String = ""
	p.Ename.Valid = false
	p.Job.String = ""
	p.Job.Valid = false
	p.Mgr.Int64 = 0
	p.Mgr.Valid = false
	p.Hiredate.String = ""
	p.Hiredate.Valid = false
	p.Sal.Int64 = 0
	p.Sal.Valid = false
	p.Comm.Int64 = 0
	p.Comm.Valid = false
	p.Deptno.Int64 = 0
	p.Deptno.Valid = false
}

// GetDept allocates a new struct or grabs a cached one
func GetDept() *Dept {
	p := deptsPool.Get().(*Dept)
	p.Reset()
	atomic.AddUint64(&countGet, 1)
	return p
}

// PutDept return struct to cache
func PutDept(p *Dept, isCascad bool) {
	if p != nil {
		PutEmpSlice(p.Emps, isCascad)
		p.Emps = nil
		deptsPool.Put(p)
		atomic.AddUint64(&countPut, 1)
	}
}

// GetEmp allocates a new struct or grabs a cached one
func GetEmp() *Emp {
	p := empsPool.Get().(*Emp)
	p.Reset()
	atomic.AddUint64(&countGet, 1)
	return p
}

// PutEmp return struct to cache
func PutEmp(p *Emp) {
	if p != nil {
		empsPool.Put(p)
		atomic.AddUint64(&countPut, 1)
	}
}

// GetEmpSlice allocates a new struct or grabs a cached one
func GetEmpSlice() EmpSlice {
	p := empSlicePool.Get().(EmpSlice)
	p.Reset()
	atomic.AddUint64(&countGet, 1)
	return p
}

// Reset reset all fields in structure - use for sync.Pool
func (p EmpSlice) Reset() {
	for i := range p {
		p[i].Reset()
		PutEmp(p[i])
		p[i] = nil // что бы не осталось подвисших ссылок
	}
}

// PutEmpSlice return struct to cache
func PutEmpSlice(p EmpSlice, isCascad bool) {
	if p != nil {
		for i := range p {
			if isCascad {
				// PutEmp(p[i])
			}
			p[i] = nil // что бы не осталось подвисших ссылок
		}
		p = p[:0] // сброс указателя среза
		empSlicePool.Put(p)
		atomic.AddUint64(&countPut, 1)
	}
}
