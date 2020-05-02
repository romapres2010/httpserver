package model

import (
	"gopkg.in/guregu/null.v4"
)

// Dept represent object "Department"
//easyjson:json
type Dept struct {
	Deptno int         `db:"deptno" json:"deptNumber" validate:"required"`
	Dname  string      `db:"dname" json:"deptName" validate:"required"`
	Loc    null.String `db:"loc" json:"deptLocation,nullempty"`
	Emps   []*Emp      `json:"emps,omitempty"` // срез указателей на дочерние emp
}

// DeptPK represent Primary Key of the object "Department"
type DeptPK struct {
	Deptno int `db:"deptno" json:"deptNumber" validate:"required"`
}

//DeptPKs represent slice of Primary Keys of the objects "Department"
type DeptPKs []*DeptPK

// Emp represent object "Employee"
//easyjson:json
type Emp struct {
	Empno    int         `db:"empno" json:"empNo" validate:"required"`
	Ename    null.String `db:"ename" json:"empName,nullempty"`
	Job      null.String `db:"job" json:"job,nullempty"`
	Mgr      null.Int    `db:"mgr" json:"mgr,omitempty"`
	Hiredate null.String `db:"hiredate" json:"hiredate,nullempty"`
	Sal      null.Int    `db:"sal" json:"sal,nullempty" validate:"gte=0"`
	Comm     null.Int    `db:"comm" json:"comm,nullempty" validate:"gte=0"`
	Deptno   null.Int    `db:"deptno" json:"deptNumber,nullempty"`
}

// EmpPK represent Primary Key of the object "Employee"
type EmpPK struct {
	Empno int `db:"empno" json:"empNo"`
}

//EmpPKs represent slice of Primary Keys of the objects "Employee"
type EmpPKs []*EmpPK

//EmpSlice represent slice of Emps
type EmpSlice []*Emp
