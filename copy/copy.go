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

func CopyObject(src, dest interface{}) error {
	if src == nil || dest == nil {
		return fmt.Errorf("source or destination cannot be nil")
	}

	srcVal := reflect.ValueOf(src)
	destVal := reflect.ValueOf(dest)

	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}
	if destVal.Kind() == reflect.Ptr {
		destVal = destVal.Elem()
	}

	if srcVal.Kind() != reflect.Struct || destVal.Kind() != reflect.Struct {
		return fmt.Errorf("both source and destination must be structs")
	}

	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		srcType := srcVal.Type().Field(i)

		if destField := destVal.FieldByName(srcType.Name); destField.IsValid() && destField.CanSet() {
			if srcField.Type() == destField.Type() {
				destField.Set(srcField)
			}
		}
	}

	return nil
}
