package model

import (
	"sync"
)

// deptsPool represent depts pooling
var deptsPool = sync.Pool{
	New: func() interface{} { return new(Dept) },
}

// empsPool represent emps pooling
var empsPool = sync.Pool{
	New: func() interface{} { return new(Emp) },
}

// deptsPKPool represent empsPK pooling
var deptsPKPool = sync.Pool{
	New: func() interface{} {
		v := make([]*DeptPK, 0)
		return DeptPKs(v)
	},
}

/*
// EmpsPKPool represent empsPK pooling
var empsPKPool = sync.Pool{
	New: func() interface{} {
		v := make([]*EmpPK, 0)
		return EmpPKs(v)
	},
}
*/

// empSlicePool represent emps Slice pooling
var empSlicePool = sync.Pool{
	New: func() interface{} {
		v := make([]*Emp, 0)
		return EmpSlice(v)
	},
}

// Reset reset all fields in structure - use for sync.Pool
func (p *Dept) Reset() {
	p.Deptno = 0
	p.Dname = ""
	p.Loc.String = ""
	p.Loc.Valid = false
	EmpSlice(p.Emps).Reset()
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

/*
// Reset reset all fields in structure - use for sync.Pool
func (p EmpPKs) Reset() {
	for i := range p {
		p[i].Empno = 0
	}
}
*/

// Reset reset all fields in structure - use for sync.Pool
func (p DeptPKs) Reset() {
	for i := range p {
		p[i].Deptno = 0
	}
}

// Reset reset all fields in structure - use for sync.Pool
func (p EmpSlice) Reset() {
	for i := range p {
		p[i].Reset()
		PutEmp(p[i])
	}
}

// GetDept allocates a new struct or grabs a cached one
func GetDept() *Dept {
	p := deptsPool.Get().(*Dept)
	p.Reset()
	return p
}

// PutDept return struct to cache
func PutDept(p *Dept, isCascad bool) {
	if p != nil {
		PutEmpSlice(p.Emps, isCascad)
		p.Emps = nil
		deptsPool.Put(p)
	}
}

// GetEmp allocates a new struct or grabs a cached one
func GetEmp() *Emp {
	p := empsPool.Get().(*Emp)
	p.Reset()
	return p
}

// PutEmp return struct to cache
func PutEmp(p *Emp) {
	if p != nil {
		empsPool.Put(p)
	}
}

// GetDeptsPK allocates a new struct or grabs a cached one
func GetDeptsPK() DeptPKs {
	p := deptsPKPool.Get().(DeptPKs)
	p.Reset()
	return p
}

// PutDeptsPK return struct to cache
func PutDeptsPK(p DeptPKs) {
	p = p[:0] // сброс
	deptsPKPool.Put(p)
}

/*
// GetEmpsPK allocates a new struct or grabs a cached one
func GetEmpsPK() EmpPKs {
	p := empsPKPool.Get().(EmpPKs)
	p.Reset()
	return p
}

// PutEmpsPK return struct to cache
func PutEmpsPK(p EmpPKs) {
	p = p[:0] // сброс
	empsPKPool.Put(p)
}
*/

// GetEmpSlice allocates a new struct or grabs a cached one
func GetEmpSlice() EmpSlice {
	p := empSlicePool.Get().(EmpSlice)
	p.Reset()
	return p
}

// PutEmpSlice return struct to cache
func PutEmpSlice(p EmpSlice, isCascad bool) {
	if p != nil {
		if isCascad {
			for i := range p {
				PutEmp(p[i])
			}
		}
		p = p[:0] // сброс указателя слайса
		empSlicePool.Put(p)
	}
}
