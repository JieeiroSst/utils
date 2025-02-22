package copy

import (
	"fmt"
	"reflect"

	"github.com/jinzhu/copier"
)

func CopyStructArrays(src interface{}, dest interface{}) error {
	srcValue := reflect.ValueOf(src)
	destValue := reflect.ValueOf(dest)

	if srcValue.Kind() != reflect.Ptr || destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("both source and destination must be pointers to slices")
	}

	srcSlice := srcValue.Elem()
	destSlice := destValue.Elem()

	if srcSlice.Kind() != reflect.Slice || destSlice.Kind() != reflect.Slice {
		return fmt.Errorf("both source and destination must be slices")
	}

	newSlice := reflect.MakeSlice(destSlice.Type(), srcSlice.Len(), srcSlice.Cap())

	for i := 0; i < srcSlice.Len(); i++ {
		srcElem := srcSlice.Index(i).Interface()
		destElem := reflect.New(newSlice.Index(i).Type()).Interface()

		if err := copier.Copy(destElem, srcElem); err != nil {
			return fmt.Errorf("error copying element %d: %v", i, err)
		}

		newSlice.Index(i).Set(reflect.ValueOf(destElem).Elem())
	}

	destSlice.Set(newSlice)
	return nil
}
