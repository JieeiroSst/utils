package hash

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/redis/go-redis/v9"
)

type cacheHelper struct {
	redis *redis.Client
}

type CacheHelper interface {
	Get(ctx context.Context, key string, dest interface{}) error
	GetInterface(ctx context.Context, key string, dataType interface{}) (map[string]interface{}, error)
	Set(ctx context.Context, key string, value interface{}) error
	SetInterface(ctx context.Context, key string, value map[string]interface{}) error
	RemoveKey(ctx context.Context, key string) error
}

func NewCacheHelper(dns string) CacheHelper {
	rdb := redis.NewClient(&redis.Options{
		Addr:     dns,
		Password: "",
		DB:       0,
	})
	return &cacheHelper{
		redis: rdb,
	}
}

func (h *cacheHelper) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := h.redis.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

func (h *cacheHelper) GetInterface(ctx context.Context, key string, dataType interface{}) (map[string]interface{}, error) {
	fields := compileStructSpec(dataType)
	if len(fields) == 0 {
		return nil, fmt.Errorf("dataType must be a struct")
	}

	values, err := h.redis.HMGet(ctx, key, fields...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{}, len(fields))

	for i, field := range fields {
		if i < len(values) && values[i] != nil {
			var v interface{}
			strValue, isString := values[i].(string)
			if isString {
				if err := json.Unmarshal([]byte(strValue), &v); err != nil {
					v = strValue
				}
			} else {
				v = values[i]
			}
			result[field] = v
		}
	}

	return result, nil
}

func (h *cacheHelper) Set(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return h.redis.Set(ctx, key, data, 0).Err()
}

func (h *cacheHelper) SetInterface(ctx context.Context, key string, value map[string]interface{}) error {
	marshaledValues := make(map[string]interface{}, len(value))
	for k, v := range value {
		var err error
		marshaledValues[k], err = h.marshalValue(v)
		if err != nil {
			return err
		}
	}

	return h.redis.HSet(ctx, key, marshaledValues).Err()
}

func (h *cacheHelper) marshalValue(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil
		}
		return h.marshalValue(rv.Elem().Interface())
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return v, nil
	case reflect.String:
		return v, nil
	case reflect.Struct:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return string(data), nil
	case reflect.Slice, reflect.Array:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return string(data), nil
	case reflect.Map:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return string(data), nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("unsupported type %s: %v", rv.Kind(), err)
		}
		return string(data), nil
	}
}

func (h *cacheHelper) RemoveKey(ctx context.Context, key string) error {
	return h.redis.Del(ctx, key).Err()
}

func compileStructSpec(dataStr interface{}) []string {
	rv := reflect.ValueOf(dataStr)
	
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	
	if rv.Kind() != reflect.Struct {
		return nil
	}

	v := rv.Type()
	fields := make([]string, 0, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			tag = field.Name
		} else {
			if comma := len(tag); comma >= 0 {
				tag = tag[:comma]
			}
		}
		
		fields = append(fields, tag)
	}
	
	return fields
}