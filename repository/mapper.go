package repository

import (
	"database/sql"
	"reflect"
	"strings"
)

// Mapper handles entity-database mapping
type Mapper[T Entity] interface {
	ToRow(entity *T) (map[string]interface{}, error)
	FromRow(scanner Scanner) (*T, error)
	Columns() []string
	TableName() string
}

// Scanner interface for row scanning
type Scanner interface {
	Scan(dest ...interface{}) error
}

// BaseMapper provides a generic mapper implementation using reflection
type BaseMapper[T Entity] struct {
	tableName string
	columns   []string
}

// NewBaseMapper creates a new base mapper
func NewBaseMapper[T Entity]() *BaseMapper[T] {
	// For types with pointer receivers, we need to get the table name differently
	var entity T
	var tableName string
	
	// Try to get table name from the zero value
	if tn, ok := any(entity).(interface{ TableName() string }); ok {
		tableName = tn.TableName()
	} else if tn, ok := any(&entity).(interface{ TableName() string }); ok {
		// If the zero value doesn't work, try with a pointer
		tableName = tn.TableName()
	}
	
	// Use reflection to get columns from struct tags
	columns := getColumnsFromStruct[T]()
	
	return &BaseMapper[T]{
		tableName: tableName,
		columns:   columns,
	}
}

// TableName returns the table name
func (m *BaseMapper[T]) TableName() string {
	return m.tableName
}

// Columns returns the column names
func (m *BaseMapper[T]) Columns() []string {
	return m.columns
}

// ToRow converts an entity to a map of column values
func (m *BaseMapper[T]) ToRow(entity *T) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		
		if tag == "" || tag == "-" {
			continue
		}
		
		// Handle nested structs (like BaseEntity)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			nestedV := v.Field(i)
			nestedT := nestedV.Type()
			
			for j := 0; j < nestedV.NumField(); j++ {
				nestedField := nestedT.Field(j)
				nestedTag := nestedField.Tag.Get("db")
				
				if nestedTag == "" || nestedTag == "-" {
					continue
				}
				
				value := nestedV.Field(j).Interface()
				result[nestedTag] = value
			}
		} else {
			value := v.Field(i).Interface()
			result[tag] = value
		}
	}
	
	return result, nil
}

// FromRow creates an entity from a database row
func (m *BaseMapper[T]) FromRow(scanner Scanner) (*T, error) {
	var entity T
	
	// Create scan destinations based on columns
	dest := m.createScanDest(&entity)
	
	if err := scanner.Scan(dest...); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	
	return &entity, nil
}

// createScanDest creates scan destinations for the entity
func (m *BaseMapper[T]) createScanDest(entity *T) []interface{} {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()
	
	dest := make([]interface{}, 0, len(m.columns))
	fieldMap := make(map[string]reflect.Value)
	
	// Build field map
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		
		if tag == "" || tag == "-" {
			continue
		}
		
		// Handle nested structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			nestedV := v.Field(i)
			nestedT := nestedV.Type()
			
			for j := 0; j < nestedV.NumField(); j++ {
				nestedField := nestedT.Field(j)
				nestedTag := nestedField.Tag.Get("db")
				
				if nestedTag == "" || nestedTag == "-" {
					continue
				}
				
				fieldMap[nestedTag] = nestedV.Field(j)
			}
		} else {
			fieldMap[tag] = v.Field(i)
		}
	}
	
	// Create destinations in column order
	for _, col := range m.columns {
		if fieldValue, ok := fieldMap[col]; ok {
			dest = append(dest, fieldValue.Addr().Interface())
		} else {
			// If column not found in struct, scan into a dummy variable
			var dummy interface{}
			dest = append(dest, &dummy)
		}
	}
	
	return dest
}

// getColumnsFromStruct extracts column names from struct tags
func getColumnsFromStruct[T Entity]() []string {
	var entity T
	v := reflect.ValueOf(&entity).Elem()
	t := v.Type()
	
	var columns []string
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		
		if tag == "" || tag == "-" {
			continue
		}
		
		// Handle nested structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			nestedT := field.Type
			
			for j := 0; j < nestedT.NumField(); j++ {
				nestedField := nestedT.Field(j)
				nestedTag := nestedField.Tag.Get("db")
				
				if nestedTag == "" || nestedTag == "-" {
					continue
				}
				
				columns = append(columns, nestedTag)
			}
		} else {
			columns = append(columns, tag)
		}
	}
	
	return columns
}

// buildColumnList builds a comma-separated list of quoted columns
func buildColumnList(columns []string, dialect interface{ Quote(string) string }) string {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = dialect.Quote(col)
	}
	return strings.Join(quoted, ", ")
}