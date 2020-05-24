package v2

import (
	"reflect"
	"runtime"
)

type agent interface {
	hooks() []func()
}

func functionName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func runHooks(a agent) {
	for _, hook := range a.hooks() {
		logger.Debugf("Prepare hook %s:", functionName(hook))
		go hook()
	}
}
