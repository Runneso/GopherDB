package optimizer

import "GopherDB/internal/core/catalog/model"

type rangeInfo struct {
	column        *model.ColumnDefinition
	from          any
	fromInclusive bool
	to            any
	toInclusive   bool
}

func buildEqRange(r *rangeInfo, v any) {
	r.from, r.to = v, v
	r.fromInclusive, r.toInclusive = true, true
}

func buildGtRange(r *rangeInfo, v any) {
	r.from, r.fromInclusive = v, false
}

func buildGteRange(r *rangeInfo, v any) {
	r.from, r.fromInclusive = v, true
}

func buildLtRange(r *rangeInfo, v any) {
	r.to, r.toInclusive = v, false
}

func buildLteRange(r *rangeInfo, v any) {
	r.to, r.toInclusive = v, true
}

func mergeRanges(left, right *rangeInfo) *rangeInfo {
	merged := &rangeInfo{column: left.column}
	merged.from, merged.fromInclusive = mergeFrom(left, right)
	merged.to, merged.toInclusive = mergeTo(left, right)
	if merged.from != nil && merged.to != nil && compareAny(merged.from, merged.to) > 0 {
		return nil
	}
	return merged
}

func mergeFrom(left, right *rangeInfo) (any, bool) {
	if left.from == nil {
		return right.from, right.fromInclusive
	}
	if right.from == nil {
		return left.from, left.fromInclusive
	}
	cmp := compareAny(left.from, right.from)
	if cmp > 0 {
		return left.from, left.fromInclusive
	}
	if cmp < 0 {
		return right.from, right.fromInclusive
	}
	return left.from, left.fromInclusive && right.fromInclusive
}

func mergeTo(left, right *rangeInfo) (any, bool) {
	if left.to == nil {
		return right.to, right.toInclusive
	}
	if right.to == nil {
		return left.to, left.toInclusive
	}
	cmp := compareAny(left.to, right.to)
	if cmp < 0 {
		return left.to, left.toInclusive
	}
	if cmp > 0 {
		return right.to, right.toInclusive
	}
	return left.to, left.toInclusive && right.toInclusive
}
