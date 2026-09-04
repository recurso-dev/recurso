package porttest

import (
	"reflect"
	"strings"
	"testing"
)

// Every generated method must panic with the interface and method name, so a
// test that forgot to override a method it needs fails with a message that
// says exactly which one. Driving each method through reflection also keeps
// the generated file fully covered.
func TestEveryUnimplementedMethodPanicsWithItsName(t *testing.T) {
	checked := 0
	for _, v := range allUnimplemented() {
		rv := reflect.ValueOf(v)
		typeName := strings.TrimPrefix(rv.Type().Name(), "Unimplemented")
		for i := 0; i < rv.NumMethod(); i++ {
			m := rv.Type().Method(i)
			args := make([]reflect.Value, 0, m.Type.NumIn()-1)
			for j := 1; j < m.Type.NumIn(); j++ { // 0 is the receiver
				args = append(args, reflect.Zero(m.Type.In(j)))
			}
			want := "porttest: " + typeName + "." + m.Name + " is not implemented"
			got := recoverPanic(func() {
				if m.Type.IsVariadic() {
					rv.Method(i).CallSlice(args)
				} else {
					rv.Method(i).Call(args)
				}
			})
			if got != want {
				t.Errorf("%s.%s: panic = %q, want %q", typeName, m.Name, got, want)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no methods checked; is unimplemented.go generated?")
	}
}

func recoverPanic(fn func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg, _ = r.(string)
		}
	}()
	fn()
	return ""
}
