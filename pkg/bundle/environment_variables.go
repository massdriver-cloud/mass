package bundle

import (
	"fmt"
	"reflect"

	"github.com/itchyny/gojq"
)

type ParsedEnvironmentVariable struct {
	Error string `json:"error"`
	Value string `json:"value"`
}

const nonStringReturnErrorMessage = "failed to return value of type string"

func ParseEnvironmentVariables(params map[string]interface{}, query map[string]string) map[string]ParsedEnvironmentVariable {
	results := make(map[string]ParsedEnvironmentVariable)

	for k, v := range query {
		result := ParsedEnvironmentVariable{}
		query, err := gojq.Parse(v)

		if err != nil {
			result.Error = fmt.Sprint(err)
			results[k] = result
			continue
		}

		iter := query.Run(params)

		for {
			v, ok := iter.Next()

			if !ok {
				break
			}

			if valueErr, valOk := v.(error); valOk {
				result.Error = fmt.Sprint(valueErr)
				results[k] = result
				continue
			}
			ofType := reflect.TypeOf(v)

			var castValue string

			if ofType == nil {
				result.Error = "failed to produce a result"
				results[k] = result
				continue
			}

			switch ofType.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				castValue = fmt.Sprintf("%d", v)
			case reflect.Float32, reflect.Float64:
				castValue = fmt.Sprintf("%f", v)
			case reflect.String:
				castValue = fmt.Sprintf("%s", v)
			case reflect.Bool:
				castValue = fmt.Sprintf("%t", v)
			// Lint wants an exhaustive list. Making it to appease the linter
			case reflect.Invalid, reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct, reflect.UnsafePointer:
				result.Error = nonStringReturnErrorMessage
				results[k] = result
			default:
				result.Error = nonStringReturnErrorMessage
				results[k] = result
			}

			result.Value = castValue
			results[k] = result
		}
	}

	return results
}
