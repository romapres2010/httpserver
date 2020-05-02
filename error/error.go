package errors

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	pkgerr "github.com/pkg/errors"
	mylog "github.com/romapres2010/httpserver/log"
)

var errorID uint64 // уникальный номер ошибки

// Error represent custom error
type Error struct {
	ID       uint64 // уникальный номер ошибки
	Code     string // код ошибки
	Msg      string // текст ошибки
	Caller   string // файл, строка и наименование метода в котором произошла ошибка
	Args     string // строка аргументов
	CauseErr error  // ошибка - причина
	CauseMsg string // текст ошибки - причины
	Trace    string // стек вызова
}

// getNextErrorID - запросить номер следующей ошибки
func getNextErrorID() uint64 {
	return atomic.AddUint64(&errorID, 1)
}

// Format output
//     %s    print the error code, message, arguments, and cause message.
//     %v    in addition to %s, print caller
//     %+v   extended format. Each Frame of the error's StackTrace will be printed in detail.
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprint(s, e.Error())
		fmt.Fprintf(s, ", caller=[%s]", e.Caller)
		if s.Flag('+') {
			fmt.Fprintf(s, ", trace=%s", e.Trace)
			return
		}
	case 's':
		fmt.Fprint(s, e.Error())
	case 'q':
		fmt.Fprint(s, e.Error())
	}
}

//Error print custom error
func (e *Error) Error() string {
	mes := fmt.Sprintf("ErrID=[%v], code=[%s], mes=[%s]", e.ID, e.Code, e.Msg)
	if e.Args != "" {
		mes = fmt.Sprintf("%s, args=[%s]", mes, e.Args)
	}
	if e.CauseMsg != "" {
		mes = fmt.Sprintf("%s, causemes=[%s]", mes, e.CauseMsg)
	}
	return mes
}

//PrintfInfo print custom error
func (e *Error) PrintfInfo(depths ...int) *Error {
	depth := 1
	if len(depths) == 1 {
		depth = depth + depths[0]
	}
	mylog.PrintfMsg("[INFO]", depth, e.Error())
	return e
}

// New - create new custom error
func New(code string, msg string, args ...interface{}) *Error {
	err := Error{
		ID:     getNextErrorID(),
		Code:   code,
		Msg:    msg,
		Caller: caller(2),
		Args:   getArgsString(args...),               // get formated string with arguments
		Trace:  fmt.Sprintf("'%+v'", pkgerr.New("")), // create err and print it trace
	}

	return &err
}

// WithCause - create new custom error with cause
func WithCause(code string, msg string, causeErr error, args ...interface{}) *Error {
	err := Error{
		ID:       getNextErrorID(),
		Code:     code,
		Msg:      msg,
		Caller:   caller(2),
		Args:     getArgsString(args...),               // get formated string with arguments
		Trace:    fmt.Sprintf("'%+v'", pkgerr.New("")), // create err and print it trace
		CauseMsg: fmt.Sprintf("'%+v'", causeErr),       // get formated string from cause error
		CauseErr: causeErr,
	}

	return &err
}

// getArgsString return formated string with arguments
func getArgsString(args ...interface{}) (argsStr string) {
	for _, arg := range args {
		if arg != nil {
			argsStr = argsStr + fmt.Sprintf("'%v', ", arg)
		}
	}
	argsStr = strings.TrimRight(argsStr, ", ")
	return
}

// caller returns a Valuer that returns a file and line from a specified depth in the callstack.
func caller(depth int) string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(depth+1, pc)
	frame, _ := runtime.CallersFrames(pc[:n]).Next()
	idxFile := strings.LastIndexByte(frame.File, '/')
	idx := strings.LastIndexByte(frame.Function, '/')
	idxName := strings.IndexByte(frame.Function[idx+1:], '.') + idx + 1

	return frame.File[idxFile+1:] + ":[" + strconv.Itoa(frame.Line) + "] - " + frame.Function[idxName+1:] + "()"
}
