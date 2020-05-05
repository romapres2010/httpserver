package model

import (
	"sync"
	"sync/atomic"

	mylog "github.com/romapres2010/httpserver/log"
)

// represent a pool statistics for benchmarking
var (
	getDepts    uint64 // количество запросов кэша
	getEmps     uint64 // количество запросов кэша
	getEmpSlice uint64 // количество запросов кэша
	putDepts    uint64 // количество возвратов в кэша
	putEmps     uint64 // количество возвратов в кэша
	putEmpSlice uint64 // количество возвратов в кэша
	newDepts    uint64 // количество создания нового объекта
	newEmps     uint64 // количество создания нового объекта
	newEmpSlice uint64 // количество создания нового объекта
)

// PrintModelPoolStats print pool statistics
func PrintModelPoolStats() {
	mylog.PrintfInfoMsg("Usage model Depts pool: Get, Put, New", getDepts, putDepts, newDepts)
	mylog.PrintfInfoMsg("Usage model Emps  pool: Get, Put, New", getEmps, putEmps, newEmps)
	mylog.PrintfInfoMsg("Usage model EmpSlice pool: Get, Put, New", getEmpSlice, putEmpSlice, newEmpSlice)
}

// deptsPool represent depts pooling
var deptsPool = sync.Pool{
	New: func() interface{} {
		atomic.AddUint64(&newDepts, 1)
		return new(Dept)
	},
}

// empsPool represent emps pooling
var empsPool = sync.Pool{
	New: func() interface{} {
		atomic.AddUint64(&newEmps, 1)
		return new(Emp)
	},
}

// empSlicePool represent emps Slice pooling
var empSlicePool = sync.Pool{
	New: func() interface{} {
		v := make([]*Emp, 0)
		atomic.AddUint64(&newEmpSlice, 1)
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
	atomic.AddUint64(&getDepts, 1)
	return p
}

// PutDept return struct to cache
func PutDept(p *Dept, isCascad bool) {
	if p != nil {
		PutEmpSlice(p.Emps, isCascad)
		p.Emps = nil
		deptsPool.Put(p)
		atomic.AddUint64(&putDepts, 1)
	}
}

// GetEmp allocates a new struct or grabs a cached one
func GetEmp() *Emp {
	p := empsPool.Get().(*Emp)
	p.Reset()
	atomic.AddUint64(&getEmps, 1)
	return p
}

// PutEmp return struct to cache
func PutEmp(p *Emp) {
	if p != nil {
		empsPool.Put(p)
		atomic.AddUint64(&putEmps, 1)
	}
}

// GetEmpSlice allocates a new struct or grabs a cached one
func GetEmpSlice() EmpSlice {
	p := empSlicePool.Get().(EmpSlice)
	p.Reset()
	atomic.AddUint64(&getEmpSlice, 1)
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
				PutEmp(p[i])
			}
			p[i] = nil // что бы не осталось подвисших ссылок
		}
		p = p[:0] // сброс указателя среза
		empSlicePool.Put(p)
		atomic.AddUint64(&putEmpSlice, 1)
	}
}
