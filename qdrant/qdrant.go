package main

import (
	"fmt"
	"reflect"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

type ScoredPointMapper struct{}

func NewScoredPointMapper() *ScoredPointMapper {
	return &ScoredPointMapper{}
}

func (m *ScoredPointMapper) MapToStruct(point *qdrant.ScoredPoint, target interface{}) error {
	if point == nil {
		return fmt.Errorf("scored point is nil")
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	targetStruct := targetValue.Elem()
	targetType := targetStruct.Type()

	for i := 0; i < targetStruct.NumField(); i++ {
		field := targetStruct.Field(i)
		fieldType := targetType.Field(i)

		if !field.CanSet() {
			continue
		}

		tag := fieldType.Tag.Get("qdrant")
		if tag == "" {
			tag = fieldType.Name
		}

		switch tag {
		case "Id", "id":
			if point.GetId() != nil {
				m.setFieldValue(field, point.GetId().GetNum())
			}
		case "Score", "score":
			m.setFieldValue(field, point.GetScore())
		case "Version", "version":
			m.setFieldValue(field, point.GetVersion())
		default:
			if payload := point.GetPayload(); payload != nil {
				if value, exists := payload[tag]; exists {
					if err := m.setFieldFromValue(field, value); err != nil {
						return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
					}
				}
			}
		}
	}

	return nil
}

func (m *ScoredPointMapper) MapToSlice(points []*qdrant.ScoredPoint, target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("target must be a pointer to slice")
	}

	targetSlice := targetValue.Elem()
	elementType := targetSlice.Type().Elem()

	newSlice := reflect.MakeSlice(targetSlice.Type(), len(points), len(points))

	for i, point := range points {
		var element reflect.Value
		if elementType.Kind() == reflect.Ptr {
			element = reflect.New(elementType.Elem())
		} else {
			element = reflect.New(elementType)
		}

		if err := m.MapToStruct(point, element.Interface()); err != nil {
			return fmt.Errorf("failed to map point %d: %w", i, err)
		}

		if elementType.Kind() == reflect.Ptr {
			newSlice.Index(i).Set(element)
		} else {
			newSlice.Index(i).Set(element.Elem())
		}
	}

	targetSlice.Set(newSlice)
	return nil
}

func (m *ScoredPointMapper) ToMap(point *qdrant.ScoredPoint) map[string]interface{} {
	if point == nil {
		return nil
	}

	result := make(map[string]interface{})

	if point.GetId() != nil {
		result["id"] = point.GetId().GetNum()
	}
	result["score"] = point.GetScore()
	result["version"] = point.GetVersion()

	if payload := point.GetPayload(); payload != nil {
		for key, value := range payload {
			result[key] = m.valueToInterface(value)
		}
	}

	return result
}

func (m *ScoredPointMapper) ToMaps(points []*qdrant.ScoredPoint) []map[string]interface{} {
	if points == nil {
		return nil
	}

	result := make([]map[string]interface{}, len(points))
	for i, point := range points {
		result[i] = m.ToMap(point)
	}

	return result
}

func (m *ScoredPointMapper) setFieldValue(field reflect.Value, value interface{}) {
	if !field.CanSet() {
		return
	}

	valueReflect := reflect.ValueOf(value)
	if valueReflect.Type().ConvertibleTo(field.Type()) {
		field.Set(valueReflect.Convert(field.Type()))
	}
}

func (m *ScoredPointMapper) setFieldFromValue(field reflect.Value, value *qdrant.Value) error {
	if !field.CanSet() || value == nil {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		if str := value.GetStringValue(); str != "" {
			field.SetString(str)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal := value.GetIntegerValue(); intVal != 0 {
			field.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if intVal := value.GetIntegerValue(); intVal != 0 {
			field.SetUint(uint64(intVal))
		}
	case reflect.Float32, reflect.Float64:
		if floatVal := value.GetDoubleValue(); floatVal != 0 {
			field.SetFloat(floatVal)
		}
	case reflect.Bool:
		field.SetBool(value.GetBoolValue())
	case reflect.Slice:
		if listVal := value.GetListValue(); listVal != nil {
			m.setSliceFromList(field, listVal.GetValues())
		}
	case reflect.Map:
		if structVal := value.GetStructValue(); structVal != nil {
			m.setMapFromStruct(field, structVal.GetFields())
		}
	case reflect.Struct:
		if field.Type() == reflect.TypeOf(time.Time{}) {
			if str := value.GetStringValue(); str != "" {
				if t, err := time.Parse(time.RFC3339, str); err == nil {
					field.Set(reflect.ValueOf(t))
				}
			}
		}
	case reflect.Interface:
		field.Set(reflect.ValueOf(m.valueToInterface(value)))
	}

	return nil
}

func (m *ScoredPointMapper) setSliceFromList(field reflect.Value, values []*qdrant.Value) {
	if len(values) == 0 {
		return
	}

	elementType := field.Type().Elem()
	slice := reflect.MakeSlice(field.Type(), len(values), len(values))

	for i, value := range values {
		element := reflect.New(elementType).Elem()
		m.setFieldFromValue(element, value)
		slice.Index(i).Set(element)
	}

	field.Set(slice)
}

func (m *ScoredPointMapper) setMapFromStruct(field reflect.Value, fields map[string]*qdrant.Value) {
	if len(fields) == 0 {
		return
	}

	mapType := field.Type()
	mapValue := reflect.MakeMap(mapType)

	for key, value := range fields {
		keyReflect := reflect.ValueOf(key)
		valueReflect := reflect.ValueOf(m.valueToInterface(value))

		if keyReflect.Type().ConvertibleTo(mapType.Key()) &&
			valueReflect.Type().ConvertibleTo(mapType.Elem()) {
			mapValue.SetMapIndex(
				keyReflect.Convert(mapType.Key()),
				valueReflect.Convert(mapType.Elem()),
			)
		}
	}

	field.Set(mapValue)
}

func (m *ScoredPointMapper) valueToInterface(value *qdrant.Value) interface{} {
	if value == nil {
		return nil
	}

	switch {
	case value.GetStringValue() != "":
		return value.GetStringValue()
	case value.GetIntegerValue() != 0:
		return value.GetIntegerValue()
	case value.GetDoubleValue() != 0:
		return value.GetDoubleValue()
	case value.GetBoolValue():
		return value.GetBoolValue()
	case value.GetListValue() != nil:
		list := value.GetListValue().GetValues()
		result := make([]interface{}, len(list))
		for i, item := range list {
			result[i] = m.valueToInterface(item)
		}
		return result
	case value.GetStructValue() != nil:
		fields := value.GetStructValue().GetFields()
		result := make(map[string]interface{})
		for key, val := range fields {
			result[key] = m.valueToInterface(val)
		}
		return result
	default:
		return nil
	}
}
