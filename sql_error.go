package tgsql

import (
	"fmt"
	"runtime"
	"strings"
)

func recoverLog(err error) *DBFuncPanicError {
	if err != nil {
		var pc []uintptr = make([]uintptr, MaxStackLen)
		n := runtime.Callers(3, pc[:])
		frames := runtime.CallersFrames(pc[:n])
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf("%v \n", err))
		for frame, more := frames.Next(); more; frame, more = frames.Next() {
			sb.WriteString(fmt.Sprintf("%s:%d \n", frame.File, frame.Line))
		}
		msg := sb.String()
		return &DBFuncPanicError{msg: msg, err: err}
	}
	return nil
}
func funcErr(funcName string, err error) *DBFuncError {
	if err != nil {
		var pc []uintptr = make([]uintptr, 2)
		n := runtime.Callers(3, pc)
		frames := runtime.CallersFrames(pc[:n])
		var msg string
		for frame, more := frames.Next(); more; frame, more = frames.Next() {
			msg = fmt.Sprintf("%s:%d", frame.File, frame.Line)
		}
		return &DBFuncError{funcName: funcName, funcFileLine: msg, err: err}
	}
	return nil
}

type DBFuncPanicError struct {
	msg string
	err error
}

func (e *DBFuncPanicError) Error() string {
	return e.msg
}

func (e *DBFuncPanicError) Unwrap() error {
	return e.err
}

type DBFuncError struct {
	funcName     string
	funcFileLine string
	err          error
}

func (e *DBFuncError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s FuncName:%s An error has occurred [%s]", e.funcFileLine, e.funcName, e.err.Error())
	}
	return e.funcName
}

func (e *DBFuncError) Unwrap() error {
	return e.err
}
