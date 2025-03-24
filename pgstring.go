package pgstring

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

const (
	TableOptionIfNotExists = "IF_NOT_EXISTS"
	TableOptionDrop        = "DROP"
	TableOptionDropCascade = "DROP_CASCADE"
)

type PgString struct {
	str       string
	fields    []string
	namedArgs map[string]any
}

// GenerateFieldPointers creates a slice of pointers to struct fields based on db or json tags
func GenerateFieldPointers(obj any) []any {
	// Ensure we have a pointer to a struct
	v := reflect.ValueOf(obj)

	// If not a pointer, return nil
	if v.Kind() != reflect.Ptr {
		return nil
	}

	// Dereference the pointer to get the struct value
	val := v.Elem()

	// Ensure it's a struct
	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	var pointers []any

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Check for db tag, otherwise use JSON tag
		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		if dbTag == "" {
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] == "-" {
					continue
				}
			}
		}

		// Get pointer to the field
		fieldPtr := val.Field(i).Addr().Interface()
		pointers = append(pointers, fieldPtr)
	}

	return pointers
}

func (pg PgString) String() string {
	return pg.str
}

func (pg PgString) NamedArgs() map[string]any {
	return pg.namedArgs
}

func (pg PgString) Result() (string, map[string]any) {
	return pg.str, pg.namedArgs
}

// extractFields extracts field names from a struct and returns them as a slice
func extractFields(obj any) []string {
	val := reflect.ValueOf(obj)

	// If pointer, get the underlying value
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Only struct types are supported
	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	var fields []string

	// Extract field names
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Check for db tag, otherwise use field name
		dbTag := field.Tag.Get("db")
		fieldName := field.Name

		if dbTag != "" && dbTag != "-" {
			// Split on comma in case there are options like omitempty
			parts := strings.Split(dbTag, ",")
			fieldName = parts[0]
		} else if dbTag == "-" {
			continue
		} else {
			// Check for JSON tag if no db tag
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					fieldName = parts[0]
				}
			}
		}

		fields = append(fields, fieldName)
	}

	return fields
}

func extractNamedArgs(obj any) map[string]any {
	result := map[string]any{}

	v := reflect.ValueOf(obj)

	// If pointer, get the underlying element
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Only process if it's a struct
	if v.Kind() != reflect.Struct {
		return result
	}

	t := v.Type()

	// Iterate through each field in the struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get the db tag, if no tag, use field name
		dbTag := field.Tag.Get("db")
		if dbTag == "" {
			// Skip if tag is explicitly set to "-"
			if field.Tag.Get("json") == "-" {
				continue
			}

			// Use JSON tag if available
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" {
				// Handle cases where json tag has options like "name,omitempty"
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					dbTag = parts[0]
				} else {
					dbTag = field.Name
				}
			} else {
				// Fallback to field name
				dbTag = field.Name
			}
		} else if dbTag == "-" {
			// Skip if db tag is explicitly set to "-"
			continue
		} else {
			// Handle db tag with options like "name,omitempty"
			parts := strings.Split(dbTag, ",")
			dbTag = parts[0]
		}

		// Get field value
		fieldValue := v.Field(i)

		// Add to named args
		result[dbTag] = fieldValue.Interface()
	}

	return result
}

// InsertInto creates a new PgString for an INSERT query
func InsertInto(table string) PgString {
	return PgString{
		str:       fmt.Sprintf("INSERT INTO %s", table),
		namedArgs: map[string]any{},
	}
}

// Obj extracts field names from the provided object and adds them to the query
func (pg PgString) Obj(obj any) PgString {
	fields := extractFields(obj)

	if fields == nil {
		return PgString{str: "Error: only struct types are supported", namedArgs: pg.namedArgs}
	}

	pg.fields = fields
	pg.str = fmt.Sprintf("%s (%s)", pg.str, strings.Join(fields, ", "))
	return pg
}

// Values extracts values from the provided object and adds placeholders to the query
func (pg PgString) Values(obj any) PgString {
	val := reflect.ValueOf(obj)

	// If pointer, get the underlying value
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Only struct types are supported
	if val.Kind() != reflect.Struct {
		return PgString{str: "Error: only struct types are supported", namedArgs: pg.namedArgs}
	}

	// Collect named arguments
	namedArgs := extractNamedArgs(obj)
	pg.namedArgs = namedArgs

	// Generate placeholders for the values
	placeholders := make([]string, len(pg.fields))
	for i, field := range pg.fields {
		placeholders[i] = fmt.Sprintf("@%s", field)
	}

	pg.str = fmt.Sprintf("%s VALUES (%s)", pg.str, strings.Join(placeholders, ", "))
	return pg
}

// Where adds a WHERE clause to the query
func (pg PgString) Where(condition string, args ...any) PgString {
	pg.str = fmt.Sprintf("%s WHERE %s", pg.str, condition)

	// If additional args are provided, add them to namedArgs
	if len(args) == 1 {
		if obj, ok := args[0].(map[string]any); ok {
			for k, v := range obj {
				pg.namedArgs[k] = v
			}
		} else {
			// Extract named args from struct
			namedArgs := extractNamedArgs(args[0])
			for k, v := range namedArgs {
				pg.namedArgs[k] = v
			}
		}
	}

	return pg
}

// Select creates a new PgString for a SELECT query with explicit fields from an object
func Select(obj any) PgString {
	fields := extractFields(obj)

	if fields == nil {
		// If not an object, treat it as a list of field names
		if strArgs, ok := obj.([]string); ok {
			return PgString{
				str:       fmt.Sprintf("SELECT %s", strings.Join(strArgs, ", ")),
				namedArgs: map[string]any{},
			}
		}

		// If it's a string, just use that directly
		if strArg, ok := obj.(string); ok {
			return PgString{
				str:       fmt.Sprintf("SELECT %s", strArg),
				namedArgs: map[string]any{},
			}
		}

		// Default to SELECT *
		return PgString{
			str:       "SELECT *",
			namedArgs: map[string]any{},
		}
	}

	// Use extracted fields from object
	return PgString{
		str:       fmt.Sprintf("SELECT %s", strings.Join(fields, ", ")),
		fields:    fields,
		namedArgs: extractNamedArgs(obj),
	}
}

// SelectStr creates a SELECT query with manually specified fields
func SelectStr(fields ...string) PgString {
	return PgString{
		str:       fmt.Sprintf("SELECT %s", strings.Join(fields, ", ")),
		fields:    fields,
		namedArgs: map[string]any{},
	}
}

// From adds a FROM clause to the query
func (pg PgString) From(table string) PgString {
	pg.str = fmt.Sprintf("%s FROM %s", pg.str, table)
	return pg
}

// Update creates a new PgString for an UPDATE query
func Update(table string) PgString {
	return PgString{
		str:       fmt.Sprintf("UPDATE %s", table),
		namedArgs: map[string]any{},
	}
}

// Set adds a SET clause for an UPDATE query
func (pg PgString) Set(obj any) PgString {
	val := reflect.ValueOf(obj)

	// If pointer, get the underlying value
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Only struct types are supported
	if val.Kind() != reflect.Struct {
		return PgString{str: "Error: only struct types are supported", namedArgs: pg.namedArgs}
	}

	// Extract named args and fields
	namedArgs := extractNamedArgs(obj)
	pg.namedArgs = namedArgs

	var setters []string

	// Generate field=@field pairs using the actual field names from the object
	for key := range namedArgs {
		setters = append(setters, fmt.Sprintf("%s = @%s", key, key))
	}

	// Sort for consistent output
	sort.Strings(setters)

	pg.str = fmt.Sprintf("%s SET %s", pg.str, strings.Join(setters, ", "))
	return pg
}

// Delete creates a new PgString for a DELETE query
func Delete() PgString {
	return PgString{
		str:       "DELETE",
		namedArgs: map[string]any{},
	}
}

// OrderBy adds an ORDER BY clause to the query
func (pg PgString) OrderBy(clause string) PgString {
	pg.str = fmt.Sprintf("%s ORDER BY %s", pg.str, clause)
	return pg
}

// Limit adds a LIMIT clause to the query
func (pg PgString) Limit(limit int) PgString {
	pg.str = fmt.Sprintf("%s LIMIT %d", pg.str, limit)
	return pg
}

// Offset adds an OFFSET clause to the query
func (pg PgString) Offset(offset int) PgString {
	pg.str = fmt.Sprintf("%s OFFSET %d", pg.str, offset)
	return pg
}

// Join adds a JOIN clause to the query
func (pg PgString) Join(joinType, table, condition string) PgString {
	pg.str = fmt.Sprintf("%s %s JOIN %s ON %s", pg.str, joinType, table, condition)
	return pg
}

// AndWhere adds an AND condition to an existing WHERE clause
func (pg PgString) AndWhere(condition string, args ...any) PgString {
	// Check if WHERE clause already exists
	if !strings.Contains(pg.str, " WHERE ") {
		return pg.Where(condition, args...)
	}

	pg.str = fmt.Sprintf("%s AND %s", pg.str, condition)

	// If additional args are provided, add them to namedArgs
	if len(args) == 1 {
		if obj, ok := args[0].(map[string]any); ok {
			for k, v := range obj {
				pg.namedArgs[k] = v
			}
		} else {
			// Extract named args from struct
			namedArgs := extractNamedArgs(args[0])
			for k, v := range namedArgs {
				pg.namedArgs[k] = v
			}
		}
	}

	return pg
}

// Returning adds a RETURNING clause to the query
func (pg PgString) Returning(obj any) PgString {
	fields := extractFields(obj)

	if fields == nil {
		// If it's a string, use it directly
		if strArg, ok := obj.(string); ok {
			pg.str = fmt.Sprintf("%s RETURNING %s", pg.str, strArg)
			return pg
		}

		// If it's a string slice, join them
		if strArgs, ok := obj.([]string); ok {
			pg.str = fmt.Sprintf("%s RETURNING %s", pg.str, strings.Join(strArgs, ", "))
			return pg
		}

		// Default to RETURNING *
		pg.str = fmt.Sprintf("%s RETURNING *", pg.str)
		return pg
	}

	// Use extracted fields
	pg.str = fmt.Sprintf("%s RETURNING %s", pg.str, strings.Join(fields, ", "))
	return pg
}

// GroupBy adds a GROUP BY clause to the query
func (pg PgString) GroupBy(clause string) PgString {
	pg.str = fmt.Sprintf("%s GROUP BY %s", pg.str, clause)
	return pg
}

// Having adds a HAVING clause to the query
func (pg PgString) Having(condition string, args ...any) PgString {
	pg.str = fmt.Sprintf("%s HAVING %s", pg.str, condition)

	// If additional args are provided, add them to namedArgs
	if len(args) == 1 {
		if obj, ok := args[0].(map[string]any); ok {
			for k, v := range obj {
				pg.namedArgs[k] = v
			}
		} else {
			// Extract named args from struct
			namedArgs := extractNamedArgs(args[0])
			for k, v := range namedArgs {
				pg.namedArgs[k] = v
			}
		}
	}

	return pg
}

func (pg PgString) OnConflict(clause string) PgString {
	if clause == "" {
		pg.str = fmt.Sprintf("%s ON CONFLICT", pg.str)
	} else {
		pg.str = fmt.Sprintf("%s ON CONFLICT %s", pg.str, clause)
	}
	return pg
}

// DoNothing adds DO NOTHING to an ON CONFLICT clause
func (pg PgString) DoNothing() PgString {
	pg.str = fmt.Sprintf("%s DO NOTHING", pg.str)
	return pg
}

// DoUpdate adds DO UPDATE SET to an ON CONFLICT clause
func (pg PgString) DoUpdate() PgString {
	pg.str = fmt.Sprintf("%s DO UPDATE", pg.str)
	return pg
}

func CreateTable(table string, obj any, options ...string) PgString {
	val := reflect.ValueOf(obj)

	// If pointer, get the underlying value
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Only struct types are supported
	if val.Kind() != reflect.Struct {
		return PgString{
			str: fmt.Sprintf("-- Error: %v is not a struct", obj),
		}
	}

	typ := val.Type()
	var columns []string
	var primaryKeys []string
	var uniqueColumns []string

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Skip fields with db:"-" tag
		if tag := field.Tag.Get("db"); tag == "-" {
			continue
		}

		// Determine column name (use db tag or field name)
		columnName := field.Name
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			// Split tag to handle potential options
			parts := strings.Split(dbTag, ",")
			columnName = parts[0]
		}

		// Determine SQL type based on Go type
		var sqlType string
		fieldType := field.Type
		isArray := false

		// Check if it's a slice/array
		if fieldType.Kind() == reflect.Slice {
			isArray = true
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.String:
			sqlType = "TEXT"
			if isArray {
				sqlType = "TEXT[]"
			}
		case reflect.Bool:
			sqlType = "BOOLEAN"
			if isArray {
				sqlType = "BOOLEAN[]"
			}
		case reflect.Int, reflect.Int32:
			sqlType = "INTEGER"
			if isArray {
				sqlType = "INTEGER[]"
			}
		case reflect.Int64:
			sqlType = "BIGINT"
			if isArray {
				sqlType = "BIGINT[]"
			}
		case reflect.Float32:
			sqlType = "REAL"
			if isArray {
				sqlType = "REAL[]"
			}
		case reflect.Float64:
			sqlType = "DOUBLE PRECISION"
			if isArray {
				sqlType = "DOUBLE PRECISION[]"
			}
		default:
			// Handle special types
			switch fieldType.String() {
			case "time.Time":
				sqlType = "TIMESTAMP"
				if isArray {
					sqlType = "TIMESTAMP[]"
				}
			case "*time.Time":
				sqlType = "TIMESTAMP"
				if isArray {
					sqlType = "TIMESTAMP[]"
				}
			default:
				sqlType = "TEXT" // fallback
				if isArray {
					sqlType = "TEXT[]"
				}
			}
		}

		// Check for constraints
		columnDef := fmt.Sprintf("%s %s", columnName, sqlType)

		// Check for primary key
		if strings.Contains(dbTag, "primarykey") {
			primaryKeys = append(primaryKeys, columnName)
		}

		// Check for NOT NULL
		if strings.Contains(dbTag, "notnull") {
			columnDef += " NOT NULL"
		}

		// Check for UNIQUE
		if strings.Contains(dbTag, "unique") {
			uniqueColumns = append(uniqueColumns, columnName)
			columnDef += " UNIQUE"
		}

		columns = append(columns, columnDef)
	}

	// Construct CREATE TABLE statement with options
	var createTableSQL strings.Builder

	// Handle table existence options
	if len(options) > 0 {
		switch options[0] {
		case TableOptionIfNotExists:
			createTableSQL.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table))
		case TableOptionDropCascade:
			createTableSQL.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;\n", table))
			createTableSQL.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table))
		case TableOptionDrop:
			createTableSQL.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", table))
			createTableSQL.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table))
		default:
			createTableSQL.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table))
		}
	} else {
		createTableSQL.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table))
	}

	createTableSQL.WriteString("    " + strings.Join(columns, ",\n    "))

	// Add primary key constraint
	if len(primaryKeys) > 0 {
		createTableSQL.WriteString(",\n    PRIMARY KEY (" + strings.Join(primaryKeys, ", ") + ")")
	}

	createTableSQL.WriteString("\n)")

	return PgString{
		str: createTableSQL.String(),
	}
}

// Left joins (add this to the existing methods)
func (pg PgString) LeftJoin(table, condition string) PgString {
	pg.str = fmt.Sprintf("%s LEFT JOIN %s ON %s", pg.str, table, condition)
	return pg
}

// Right joins
func (pg PgString) RightJoin(table, condition string) PgString {
	pg.str = fmt.Sprintf("%s RIGHT JOIN %s ON %s", pg.str, table, condition)
	return pg
}

// Full outer joins
func (pg PgString) FullOuterJoin(table, condition string) PgString {
	pg.str = fmt.Sprintf("%s FULL OUTER JOIN %s ON %s", pg.str, table, condition)
	return pg
}

// Distinct modifier for SELECT
func (pg PgString) Distinct() PgString {
	if strings.HasPrefix(pg.str, "SELECT") {
		pg.str = strings.Replace(pg.str, "SELECT", "SELECT DISTINCT", 1)
	}
	return pg
}

// Like condition (for WHERE clauses)
func (pg PgString) Like(column, pattern string) PgString {
	condition := fmt.Sprintf("%s LIKE @%s_pattern", column, column)
	pg.str = fmt.Sprintf("%s WHERE %s", pg.str, condition)
	pg.namedArgs[column+"_pattern"] = pattern
	return pg
}

// In condition
func (pg PgString) In(column string, values []any) PgString {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholderKey := fmt.Sprintf("%s_in_%d", column, i)
		placeholders[i] = fmt.Sprintf("@%s", placeholderKey)
		pg.namedArgs[placeholderKey] = values[i]
	}

	condition := fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", "))

	if strings.Contains(pg.str, " WHERE ") {
		pg.str = fmt.Sprintf("%s AND %s", pg.str, condition)
	} else {
		pg.str = fmt.Sprintf("%s WHERE %s", pg.str, condition)
	}

	return pg
}

// Between condition
func (pg PgString) Between(column string, start, end any) PgString {
	condition := fmt.Sprintf("%s BETWEEN @%s_start AND @%s_end", column, column, column)
	pg.str = fmt.Sprintf("%s WHERE %s", pg.str, condition)
	pg.namedArgs[column+"_start"] = start
	pg.namedArgs[column+"_end"] = end
	return pg
}

// Raw SQL method for complex queries
func RawSQL(query string) PgString {
	return PgString{
		str:       query,
		namedArgs: map[string]any{},
	}
}
