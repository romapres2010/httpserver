package errors

import (
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
)

// Error represent custom error
type Error struct {
	//err       errors.Error // ошибка
	Code      string // код ошибки
	Msg       string // сформатированное сообщение об ошибке
	File      string // файл где произошла ошибка
	Line      int    // строка кода
	Method    string // дополнительная информация о методе, который привел к ошибке
	Args      string // аргументы метода
	CauseCode string // код ошибки - причины
	CauseMes  string // текст ошибки - причины
	Stack     *stack // стек вызова
}

// Format output
//     %s    print the error. If the error has a Cause it will be
//           printed recursively.
//     %v    see %s
//     %+v   extended format. Each Frame of the error's StackTrace will
//           be printed in detail.
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, e.Error())
			e.Stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.Msg)
	case 'q':
		fmt.Fprintf(s, "%q", e.Msg)
	}
}

//Error print custom error
func (e *Error) Error() string {
	return fmt.Sprintf("[ERROR] code=[%s], caller='%s:%s', method='%s', args='%s', mes='%s', causecode='%s', causemes='%s'", e.Code, e.File, strconv.Itoa(e.Line), e.Method, e.Args, e.Msg, e.CauseCode, e.CauseMes)
}

// New - create new custom error
func New(code string, msg string, method string, args string) error {
	err := Error{
		Code:   code,
		Msg:    msg,
		Method: method,
		Args:   args,
		Stack:  callers(),
	}

	err.File, err.Line = CallerFileLine(2)

	return &err
}

// WithCause - create new custom error with cause
func WithCause(code string, msg string, method string, args string, causeCode string, causeMes string) error {
	err := Error{
		Code:      code,
		Msg:       msg,
		Method:    method,
		Args:      args,
		CauseCode: causeCode,
		CauseMes:  causeMes,
		Stack:     callers(),
	}

	err.File, err.Line = CallerFileLine(2)

	return &err
}

// CallerFileLine returns a Valuer that returns a file and line
func CallerFileLine(depth int) (string, int) {
	_, file, line, _ := runtime.Caller(depth)
	_ = line
	idx := strings.LastIndexByte(file, '/')
	// using idx+1 below handles both of following cases:
	// idx == -1 because no "/" was found, or
	// idx >= 0 and we want to start at the character after the found "/".
	return file[idx+1:], line
}
