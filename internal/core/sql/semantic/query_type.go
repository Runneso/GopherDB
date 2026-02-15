package semantic

type QueryType int

const (
	QueryTypeCreateTable QueryType = iota
	QueryTypeInsert
	QueryTypeSelect
	QueryTypeCreateIndex
	QueryTypeExplain
)
