package soy

import (
	"github.com/zoobz-io/astql"
)

// whereBuilder provides shared WHERE clause building logic for query builders.
// This helper eliminates code duplication across Select, Update, Delete, and aggregate builders.
type whereBuilder struct {
	instance *astql.ASTQL
	builder  *astql.Builder
}

// newWhereBuilder creates a new WHERE clause builder helper.
func newWhereBuilder(instance *astql.ASTQL, builder *astql.Builder) *whereBuilder {
	return &whereBuilder{
		instance: instance,
		builder:  builder,
	}
}

// addWhere adds a simple WHERE condition with field operator param pattern.
// Returns the updated builder and any error encountered.
func (w *whereBuilder) addWhere(field, operator, param string) (*astql.Builder, error) {
	builder, _, err := w.addWhereWithCondition(field, operator, param)
	return builder, err
}

// addWhereWithCondition adds a simple WHERE condition and also returns the condition item.
// This is used when the caller needs to track conditions for fallback queries.
func (w *whereBuilder) addWhereWithCondition(field, operator, param string) (*astql.Builder, astql.ConditionItem, error) {
	astqlOp, err := validateOperator(operator)
	if err != nil {
		return w.builder, nil, err
	}

	f, err := w.instance.TryF(field)
	if err != nil {
		return w.builder, nil, newFieldError(field, err)
	}

	p, err := w.instance.TryP(param)
	if err != nil {
		return w.builder, nil, newParamError(param, err)
	}

	condition, err := w.instance.TryC(f, astqlOp, p)
	if err != nil {
		return w.builder, nil, newConditionError(err)
	}

	return w.builder.Where(condition), condition, nil
}

// addWhereAnd adds multiple conditions combined with AND.
func (w *whereBuilder) addWhereAnd(conditions ...Condition) (*astql.Builder, error) {
	builder, _, err := w.addWhereAndWithCondition(conditions...)
	return builder, err
}

// addWhereAndWithCondition adds multiple conditions combined with AND and returns the group.
func (w *whereBuilder) addWhereAndWithCondition(conditions ...Condition) (*astql.Builder, astql.ConditionItem, error) {
	if len(conditions) == 0 {
		return w.builder, nil, nil
	}

	conditionItems := w.instance.ConditionItems()
	for _, cond := range conditions {
		condItem, err := w.buildCondition(cond)
		if err != nil {
			return w.builder, nil, err
		}
		conditionItems = append(conditionItems, condItem)
	}

	andGroup, err := w.instance.TryAnd(conditionItems...)
	if err != nil {
		return w.builder, nil, newConditionError(err)
	}

	return w.builder.Where(andGroup), andGroup, nil
}

// addWhereOr adds multiple conditions combined with OR.
func (w *whereBuilder) addWhereOr(conditions ...Condition) (*astql.Builder, error) {
	builder, _, err := w.addWhereOrWithCondition(conditions...)
	return builder, err
}

// addWhereOrWithCondition adds multiple conditions combined with OR and returns the group.
func (w *whereBuilder) addWhereOrWithCondition(conditions ...Condition) (*astql.Builder, astql.ConditionItem, error) {
	if len(conditions) == 0 {
		return w.builder, nil, nil
	}

	conditionItems := w.instance.ConditionItems()
	for _, cond := range conditions {
		condItem, err := w.buildCondition(cond)
		if err != nil {
			return w.builder, nil, err
		}
		conditionItems = append(conditionItems, condItem)
	}

	orGroup, err := w.instance.TryOr(conditionItems...)
	if err != nil {
		return w.builder, nil, newConditionError(err)
	}

	return w.builder.Where(orGroup), orGroup, nil
}

// addWhereNull adds a WHERE field IS NULL condition.
func (w *whereBuilder) addWhereNull(field string) (*astql.Builder, error) {
	builder, _, err := w.addWhereNullWithCondition(field)
	return builder, err
}

// addWhereNullWithCondition adds a WHERE field IS NULL condition and returns the condition.
func (w *whereBuilder) addWhereNullWithCondition(field string) (*astql.Builder, astql.ConditionItem, error) {
	f, err := w.instance.TryF(field)
	if err != nil {
		return w.builder, nil, newFieldError(field, err)
	}

	condition, err := w.instance.TryNull(f)
	if err != nil {
		return w.builder, nil, newConditionError(err)
	}

	return w.builder.Where(condition), condition, nil
}

// addWhereNotNull adds a WHERE field IS NOT NULL condition.
func (w *whereBuilder) addWhereNotNull(field string) (*astql.Builder, error) {
	builder, _, err := w.addWhereNotNullWithCondition(field)
	return builder, err
}

// addWhereNotNullWithCondition adds a WHERE field IS NOT NULL condition and returns the condition.
func (w *whereBuilder) addWhereNotNullWithCondition(field string) (*astql.Builder, astql.ConditionItem, error) {
	f, err := w.instance.TryF(field)
	if err != nil {
		return w.builder, nil, newFieldError(field, err)
	}

	condition, err := w.instance.TryNotNull(f)
	if err != nil {
		return w.builder, nil, newConditionError(err)
	}

	return w.builder.Where(condition), condition, nil
}

// addWhereBetween adds a WHERE field BETWEEN low AND high condition.
func (w *whereBuilder) addWhereBetween(field, lowParam, highParam string) (*astql.Builder, error) {
	builder, _, err := w.addWhereBetweenWithCondition(field, lowParam, highParam)
	return builder, err
}

// addWhereBetweenWithCondition adds a WHERE field BETWEEN low AND high condition and returns the condition.
func (w *whereBuilder) addWhereBetweenWithCondition(field, lowParam, highParam string) (*astql.Builder, astql.ConditionItem, error) {
	f, err := w.instance.TryF(field)
	if err != nil {
		return w.builder, nil, newFieldError(field, err)
	}

	lowP, err := w.instance.TryP(lowParam)
	if err != nil {
		return w.builder, nil, newParamError(lowParam, err)
	}

	highP, err := w.instance.TryP(highParam)
	if err != nil {
		return w.builder, nil, newParamError(highParam, err)
	}

	condition := astql.Between(f, lowP, highP)
	return w.builder.Where(condition), condition, nil
}

// addWhereNotBetween adds a WHERE field NOT BETWEEN low AND high condition.
func (w *whereBuilder) addWhereNotBetween(field, lowParam, highParam string) (*astql.Builder, error) {
	builder, _, err := w.addWhereNotBetweenWithCondition(field, lowParam, highParam)
	return builder, err
}

// addWhereNotBetweenWithCondition adds a WHERE field NOT BETWEEN low AND high condition and returns the condition.
func (w *whereBuilder) addWhereNotBetweenWithCondition(field, lowParam, highParam string) (*astql.Builder, astql.ConditionItem, error) {
	f, err := w.instance.TryF(field)
	if err != nil {
		return w.builder, nil, newFieldError(field, err)
	}

	lowP, err := w.instance.TryP(lowParam)
	if err != nil {
		return w.builder, nil, newParamError(lowParam, err)
	}

	highP, err := w.instance.TryP(highParam)
	if err != nil {
		return w.builder, nil, newParamError(highParam, err)
	}

	condition := astql.NotBetween(f, lowP, highP)
	return w.builder.Where(condition), condition, nil
}

// Subquery is implemented by soy query builders that can serve as the right-hand
// side of a subquery WHERE condition (WhereInSubquery, WhereExists, and friends).
// Both *Query[T] and *Select[T] satisfy it, for any row type T — the subquery's row
// type is independent of the outer query, so cross-table subqueries work
// (e.g. WHERE user_id IN (SELECT id FROM users ...)).
//
// The interface method is unexported, so only soy's own builders can act as
// subqueries; callers construct one with soy.Query()/soy.Select().
//
// Parameter naming: the renderer prefixes a subquery's parameters with sq<depth>_
// to keep them distinct from the outer query's parameters. A subquery param named
// "batch" is rendered as :sq1_batch, so the Exec param map must use the prefixed
// key. The authoritative list of expected keys is QueryResult.RequiredParams from
// Render() — read it if you are unsure what a nested query expects.
type Subquery interface {
	// subqueryBuilder returns the underlying astql builder to embed, or the
	// subquery's accumulated build error.
	subqueryBuilder() (*astql.Builder, error)
}

// resolveSubquery extracts and validates the underlying astql builder from a
// Subquery. It returns an error rather than letting astql.Sub panic when the
// subquery fails to build: Build() is idempotent and side-effect-free, so
// pre-validating here guarantees the later astql.Sub call cannot panic.
func resolveSubquery(sub Subquery) (*astql.Builder, error) {
	if sub == nil {
		return nil, ErrNilSubquery
	}
	subBuilder, err := sub.subqueryBuilder()
	if err != nil {
		return nil, err
	}
	if subBuilder == nil {
		return nil, ErrNilSubquery
	}
	if _, err := subBuilder.Build(); err != nil {
		return nil, newSubqueryError(err)
	}
	return subBuilder, nil
}

// addWhereInSubquery adds a WHERE field IN/NOT IN (subquery) condition.
// op must be astql.IN or astql.NotIn.
func (w *whereBuilder) addWhereInSubquery(field string, op astql.Operator, sub Subquery) (*astql.Builder, astql.ConditionItem, error) {
	subBuilder, err := resolveSubquery(sub)
	if err != nil {
		return w.builder, nil, err
	}

	f, err := w.instance.TryF(field)
	if err != nil {
		return w.builder, nil, newFieldError(field, err)
	}

	condition := astql.CSub(f, op, astql.Sub(subBuilder))
	return w.builder.Where(condition), condition, nil
}

// addWhereExists adds a WHERE EXISTS/NOT EXISTS (subquery) condition.
// op must be astql.EXISTS or astql.NotExists.
func (w *whereBuilder) addWhereExists(op astql.Operator, sub Subquery) (*astql.Builder, astql.ConditionItem, error) {
	subBuilder, err := resolveSubquery(sub)
	if err != nil {
		return w.builder, nil, err
	}

	condition := astql.CSubExists(op, astql.Sub(subBuilder))
	return w.builder.Where(condition), condition, nil
}

// buildCondition converts a Condition to an ASTQL condition.
func (w *whereBuilder) buildCondition(cond Condition) (astql.ConditionItem, error) {
	return buildConditionWithInstance(w.instance, cond)
}

// buildConditionWithInstance is a shared helper that converts a Condition to an ASTQL condition.
// This is extracted to avoid code duplication across Select, Query, Update, Delete builders.
func buildConditionWithInstance(instance *astql.ASTQL, cond Condition) (astql.ConditionItem, error) {
	f, err := instance.TryF(cond.field)
	if err != nil {
		return nil, newFieldError(cond.field, err)
	}

	if cond.isNull {
		if cond.operator == opIsNull {
			return instance.TryNull(f)
		}
		return instance.TryNotNull(f)
	}

	if cond.isBetween {
		lowP, lowErr := instance.TryP(cond.lowParam)
		if lowErr != nil {
			return nil, newParamError(cond.lowParam, lowErr)
		}
		highP, highErr := instance.TryP(cond.highParam)
		if highErr != nil {
			return nil, newParamError(cond.highParam, highErr)
		}
		if cond.operator == opBetween {
			return astql.Between(f, lowP, highP), nil
		}
		return astql.NotBetween(f, lowP, highP), nil
	}

	astqlOp, err := validateOperator(cond.operator)
	if err != nil {
		return nil, err
	}

	p, err := instance.TryP(cond.param)
	if err != nil {
		return nil, newParamError(cond.param, err)
	}

	return instance.TryC(f, astqlOp, p)
}

// buildCaseWhenCondition builds the condition for a CASE WHEN clause.
// This is extracted to avoid code duplication across SelectCaseBuilder and QueryCaseBuilder.
// The caller is responsible for resolving the result param.
func buildCaseWhenCondition(instance *astql.ASTQL, field, operator, param string) (astql.ConditionItem, error) {
	astqlOp, err := validateOperator(operator)
	if err != nil {
		return nil, err
	}

	f, err := instance.TryF(field)
	if err != nil {
		return nil, newFieldError(field, err)
	}

	p, err := instance.TryP(param)
	if err != nil {
		return nil, newParamError(param, err)
	}

	condition, err := instance.TryC(f, astqlOp, p)
	if err != nil {
		return nil, newConditionError(err)
	}

	return condition, nil
}

// Operator constants to avoid duplication.
const (
	opIsNull     = "IS NULL"
	opIsNotNull  = "IS NOT NULL"
	opBetween    = "BETWEEN"
	opNotBetween = "NOT BETWEEN"
)
