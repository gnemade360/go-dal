package repository

import (
	"fmt"
	"strings"

	"github.com/gnemade360/go-dal/dal/interfaces"
)

// Specification represents a query specification
type Specification interface {
	ToSQL(dialect interfaces.Dialect) (string, []interface{}, error)
	And(spec Specification) Specification
	Or(spec Specification) Specification
	Not() Specification
}

// SpecificationOperator represents logical operators
type SpecificationOperator string

const (
	OperatorAnd SpecificationOperator = "AND"
	OperatorOr  SpecificationOperator = "OR"
	OperatorNot SpecificationOperator = "NOT"
)

// FieldOperator represents comparison operators
type FieldOperator string

const (
	OpEqual          FieldOperator = "="
	OpNotEqual       FieldOperator = "!="
	OpGreater        FieldOperator = ">"
	OpGreaterOrEqual FieldOperator = ">="
	OpLess           FieldOperator = "<"
	OpLessOrEqual    FieldOperator = "<="
	OpLike           FieldOperator = "LIKE"
	OpNotLike        FieldOperator = "NOT LIKE"
	OpIn             FieldOperator = "IN"
	OpNotIn          FieldOperator = "NOT IN"
	OpIsNull         FieldOperator = "IS NULL"
	OpIsNotNull      FieldOperator = "IS NOT NULL"
	OpBetween        FieldOperator = "BETWEEN"
)

// BaseSpecification provides common specification functionality
type BaseSpecification struct{}

// ToSQL must be implemented by concrete specifications
func (s BaseSpecification) ToSQL(dialect interfaces.Dialect) (string, []interface{}, error) {
	return "", nil, fmt.Errorf("ToSQL must be implemented by concrete specification")
}

// And creates an AND composite specification
func (s BaseSpecification) And(spec Specification) Specification {
	return &CompositeSpecification{
		Left:     s,
		Right:    spec,
		Operator: OperatorAnd,
	}
}

// Or creates an OR composite specification
func (s BaseSpecification) Or(spec Specification) Specification {
	return &CompositeSpecification{
		Left:     s,
		Right:    spec,
		Operator: OperatorOr,
	}
}

// Not creates a NOT specification
func (s BaseSpecification) Not() Specification {
	return &NotSpecification{
		Spec: s,
	}
}

// CompositeSpecification combines two specifications
type CompositeSpecification struct {
	BaseSpecification
	Left     Specification
	Right    Specification
	Operator SpecificationOperator
}

// ToSQL converts the composite specification to SQL
func (s *CompositeSpecification) ToSQL(dialect interfaces.Dialect) (string, []interface{}, error) {
	leftSQL, leftArgs, err := s.Left.ToSQL(dialect)
	if err != nil {
		return "", nil, err
	}

	rightSQL, rightArgs, err := s.Right.ToSQL(dialect)
	if err != nil {
		return "", nil, err
	}

	sql := fmt.Sprintf("(%s %s %s)", leftSQL, s.Operator, rightSQL)
	args := append(leftArgs, rightArgs...)

	return sql, args, nil
}

// NotSpecification negates a specification
type NotSpecification struct {
	BaseSpecification
	Spec Specification
}

// ToSQL converts the NOT specification to SQL
func (s *NotSpecification) ToSQL(dialect interfaces.Dialect) (string, []interface{}, error) {
	sql, args, err := s.Spec.ToSQL(dialect)
	if err != nil {
		return "", nil, err
	}

	return fmt.Sprintf("NOT (%s)", sql), args, nil
}

// FieldSpecification represents a field comparison
type FieldSpecification struct {
	BaseSpecification
	Field    string
	Operator FieldOperator
	Value    interface{}
}

// NewFieldSpec creates a new field specification
func NewFieldSpec(field string, operator FieldOperator, value interface{}) *FieldSpecification {
	return &FieldSpecification{
		Field:    field,
		Operator: operator,
		Value:    value,
	}
}

// ToSQL converts the field specification to SQL
func (s *FieldSpecification) ToSQL(dialect interfaces.Dialect) (string, []interface{}, error) {
	quotedField := dialect.Quote(s.Field)

	switch s.Operator {
	case OpIsNull, OpIsNotNull:
		return fmt.Sprintf("%s %s", quotedField, s.Operator), nil, nil

	case OpIn, OpNotIn:
		values, ok := s.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("IN/NOT IN operator requires slice value")
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = dialect.Placeholder(i + 1)
		}
		sql := fmt.Sprintf("%s %s (%s)", quotedField, s.Operator, strings.Join(placeholders, ", "))
		return sql, values, nil

	case OpBetween:
		values, ok := s.Value.([]interface{})
		if !ok || len(values) != 2 {
			return "", nil, fmt.Errorf("BETWEEN operator requires exactly 2 values")
		}
		sql := fmt.Sprintf("%s BETWEEN %s AND %s", quotedField, dialect.Placeholder(1), dialect.Placeholder(2))
		return sql, values, nil

	default:
		sql := fmt.Sprintf("%s %s %s", quotedField, s.Operator, dialect.Placeholder(1))
		return sql, []interface{}{s.Value}, nil
	}
}

// IDSpecification is a convenience specification for ID lookup
type IDSpecification struct {
	BaseSpecification
	ID string
}

// NewIDSpec creates a new ID specification
func NewIDSpec(id string) *IDSpecification {
	return &IDSpecification{ID: id}
}

// ToSQL converts the ID specification to SQL
func (s *IDSpecification) ToSQL(dialect interfaces.Dialect) (string, []interface{}, error) {
	return fmt.Sprintf("%s = %s", dialect.Quote("id"), dialect.Placeholder(1)), []interface{}{s.ID}, nil
}

// TrueSpecification always returns true
type TrueSpecification struct {
	BaseSpecification
}

// ToSQL returns a SQL condition that is always true
func (s *TrueSpecification) ToSQL(dialect interfaces.Dialect) (string, []interface{}, error) {
	return "1 = 1", nil, nil
}

// FalseSpecification always returns false
type FalseSpecification struct {
	BaseSpecification
}

// ToSQL returns a SQL condition that is always false
func (s *FalseSpecification) ToSQL(dialect interfaces.Dialect) (string, []interface{}, error) {
	return "1 = 0", nil, nil
}