package semantic

type ExplainQueryTree struct {
	inner QueryTree
}

func NewExplainQueryTree(inner QueryTree) *ExplainQueryTree {
	return &ExplainQueryTree{
		inner: inner,
	}
}

func (query *ExplainQueryTree) Type() QueryType {
	return QueryTypeExplain
}

func (query *ExplainQueryTree) Inner() QueryTree {
	return query.inner
}
