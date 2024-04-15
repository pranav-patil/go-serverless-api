package util

import (
	"fmt"
	"reflect"
	"strings"
)

// Converts the Struct fields into a map with key and value pair.
// Use the value in specified tagname as key value, and field value as value.
// The fields which has list of tags at the end will be ignored
func StructToMap(in interface{}, tagName string, ignoreDefaultValue bool,
	tags ...string) (map[string]interface{}, error) {
	out := make(map[string]interface{})

	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct { // Non-structural return error
		return nil, fmt.Errorf("ToMap only accepts struct or struct pointer; got %T", v)
	}

	t := v.Type()

	// the tagName value is specified as the key in the map; the field value as the value in the map
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)

		// cannot access the value of unexported fields
		if field.PkgPath != "" {
			continue
		}

		// boolean value with false are ignored as they can be defaults
		if ignoreDefaultValue && reflect.Zero(field.Type).Interface() == v.Field(i).Interface() {
			continue
		}

		// ignore fields which has corresponding tags
		if len(tags) > 0 && doesFieldHaveAnyTag(&field, tags...) {
			continue
		}

		if tagValue := field.Tag.Get(tagName); tagValue != "" {
			res := strings.Split(tagValue, ",")
			tagValue = res[0]
			tagOpts := res[1:]

			if Contains(tagOpts, "omitempty") {
				name := field.Name
				val := v.FieldByName(name)
				zero := reflect.Zero(val.Type()).Interface()
				current := val.Interface()

				if reflect.DeepEqual(current, zero) {
					continue
				}
			}
			out[tagValue] = v.Field(i).Interface()
		}
	}
	return out, nil
}

func GetStructField(structInstance interface{}, fieldName string) interface{} {
	v := reflect.ValueOf(structInstance)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		field := v.FieldByName(fieldName)
		if field.IsValid() {
			return field.Interface()
		}
	}
	return nil
}

func SetStructField(structInstance interface{}, fieldName string, value interface{}) {
	v := reflect.ValueOf(structInstance)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		field := v.FieldByName(fieldName)
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf(value))
		}
	}
}

func doesFieldHaveAnyTag(field *reflect.StructField, tags ...string) bool {
	for _, tagName := range tags {
		if tagValue := field.Tag.Get(tagName); tagValue != "" {
			return true
		}
	}
	return false
}

func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	var chunks [][]T
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

func FindElement[T comparable](slice []T, value T) int {
	for p, v := range slice {
		if v == value {
			return p
		}
	}
	return -1
}
