package verifyu

import "fmt"

func VerifyOrder[T any](s []T, ok func(a, b T) bool) {
	for i := 1; i < len(s); i++ {
		a, b := s[i-1], s[i]
		if !ok(a, b) {
			panic(fmt.Sprintf("bad ordering: %v before %v", a, b))
		}
	}
}
