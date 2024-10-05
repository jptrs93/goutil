package reflectu

import (
	"reflect"
	"strings"
)

func CallNoArgSetters(obj interface{}) {
	v := reflect.ValueOf(obj)
	t := v.Type()
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		methodName := method.Name

		if strings.HasPrefix(strings.ToLower(methodName), "set") {
			continue
		}
		if method.Type.NumIn() == 1 && method.Type.NumOut() == 0 { // NumIn() includes the receiver
			v.Method(i).Call(nil)
		}
	}
}
