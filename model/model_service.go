package model

import (
	"context"
)

// DeptService represent basic interface for Dept
type DeptService interface {
	GetDept(ctx context.Context, out *Dept) (bool, error)
	GetDeptsPK(ctx context.Context, out *DeptPKs) error
	CreateDept(ctx context.Context, in *Dept, out *Dept) error
	UpdateDept(ctx context.Context, in *Dept, out *Dept) (bool, error)

	//RandomGetDept(ctx context.Context, v *Dept) error    // Для целей нагрузочного тестирования
	//RandomUpdateDept(ctx context.Context, v *Dept) error // Для целей нагрузочного тестирования
}

// EmpService represent basic interface for Emp
type EmpService interface {
	GetEmp(ctx context.Context, out *Emp) (bool, error)
	GetEmpsByDept(ctx context.Context, in *Dept, out *EmpSlice) error
	CreateEmp(ctx context.Context, in *Emp, out *Emp) error
	UpdateEmp(ctx context.Context, in *Emp, out *Emp) (bool, error)
}
