//package csv_model
package main

import (
  "fmt"
  "reflect"
  "os"
  "encoding/csv"
)

type Row struct {
  Header  []string
  Body    []string
}

func (row *Row) Get(key string) string {
  for index, column := range row.Header {
    if column == key {
      return row.Body[index]
    }
  }

  return row.Body[0]
}

type QueryCondition func(row *Row) bool

type QueryBuilder struct {
  RModel  *any
  Conds   []QueryCondition
  File    *os.File
}

func (builder *QueryBuilder) Where(cond QueryCondition) *QueryBuilder {
  builder.Conds = append(builder.Conds, cond)
  return builder
}

func MapToStruct(data map[string]interface{}, model interface{}) interface{} {
	modelType := reflect.TypeOf(model)
	modelValue := reflect.New(modelType).Elem()

	for key, value := range data {
		field := modelValue.FieldByName(key)
		if field.IsValid() && field.CanSet() {
			fieldValue := reflect.ValueOf(value)
			if fieldValue.Type().AssignableTo(field.Type()) {
				field.Set(fieldValue)
			}
		}
	}

	return modelValue.Interface()
}

func Hydrate[T any](model T, row *Row) interface{} {
  modelReflect := reflect.TypeOf(model)
  structMap := make(map[string]interface{})

  for index := 0; index < modelReflect.NumField(); index++ {
    field := modelReflect.Field(index)

    if column, ok := field.Tag.Lookup("column"); ok {
      structMap[field.Name] = row.Get(column)
    }
  }

  hydratedModel := MapToStruct(structMap, model)

  return hydratedModel
}

func CastToStructArray(data []interface{}, structType interface{}) interface{} {
	sliceType := reflect.SliceOf(reflect.TypeOf(structType))
	sliceValue := reflect.MakeSlice(sliceType, len(data), len(data))

	for i, item := range data {
		structValue := reflect.New(reflect.TypeOf(structType)).Elem()
		itemValue := reflect.ValueOf(item)
		if itemValue.Type().AssignableTo(structValue.Type()) {
			structValue.Set(itemValue)
			sliceValue.Index(i).Set(structValue)
		}
	}

	return sliceValue.Interface()
}

func (builder *QueryBuilder) Find(tstruct any) interface{} {
  var models []any
  var header []string

  records, err := csv.NewReader(builder.File).ReadAll()
  if err != nil {
    panic(err)
  }

  for index, record := range records {
    if index == 0 {
      header = record
      continue
    }

    row := Row{header, record}
    allSatisfied := true
    for _, cond := range builder.Conds {
      allSatisfied = cond(&row)
    }

    if !allSatisfied {
      continue
    }

    models = append(models, Hydrate(tstruct, &row))
  }

  builder.File.Close()
  return CastToStructArray(models, tstruct)
}

func Using(path string) *QueryBuilder {
  file, err := os.Open(path)
  if err != nil {
    panic(err)
  }

  var builder QueryBuilder
  builder.File = file
  return &builder
}

// using csv as models

type Test struct {
  Id      string `column:"id"`
  Name    string `column:"name"`
  Active  string `column:"active"`
}


func main() {
  tests := Using("./tests.csv").
    Where(func(row *Row)bool {return row.Get("active") == "true"}).
    Find(Test{}).([]Test)

  fmt.Println(tests[0].Id)
}


