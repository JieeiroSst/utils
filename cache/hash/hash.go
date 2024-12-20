package hash

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/redis/go-redis/v9"
)

type cacheHelper struct {
	redis *redis.Client
}

type CacheHelper interface {
	GetInterface(ctx context.Context, key string, dataType interface{}) (interface{}, error)
	SetInterface(ctx context.Context, key string, value map[string]interface{}) error
	Removekey(ctx context.Context, key string) error
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

func (h *cacheHelper) GetInterface(ctx context.Context, key string, dataType interface{}) (interface{}, error) {
	pipe := h.redis.Pipeline()
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	fields := compileStructSpec(dataType)
	cmd := pipe.HMGet(ctx, key, fields...)

	var data interface{}
	if err := cmd.Scan(&data); err != nil {
		return nil, err
	}

	return data, nil
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

	err := h.redis.HMSet(ctx, key, marshaledValues).Err()
	if err != nil {
		return err
	}
	return nil
}

func (h *cacheHelper) marshalValue(v interface{}) (interface{}, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Struct:
		dataJson, err := json.Marshal(&v)
		if err != nil {
			return nil, err
		}
		return dataJson, nil
	case reflect.Slice:
		array := make([]interface{}, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			elem, err := h.marshalValue(rv.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			array = append(array, elem)
		}
		return array, nil
	case reflect.Map:
		mappedValues := make(map[string]interface{}, rv.Len())
		for _, kk := range rv.MapKeys() {
			elem, err := h.marshalValue(rv.MapIndex(kk).Interface())
			if err != nil {
				return nil, err
			}
			mappedValues[kk.Interface().(string)] = elem
		}
		return mappedValues, nil
	default:
		return v, nil
	}
}

func (h *cacheHelper) Removekey(ctx context.Context, key string) error {
	if err := h.redis.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}

func compileStructSpec(dataStr interface{}) []string {
	array := make([]string, 0)
	rv := reflect.ValueOf(dataStr)
	if rv.Kind() != reflect.Struct {
		return nil
	}

	v := reflect.TypeOf(dataStr)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := field.Tag.Get("json")

		array = append(array, tag)
	}
	return array
}
