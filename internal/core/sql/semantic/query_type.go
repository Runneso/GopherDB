package semantic

type QueryType int

const (
	QueryTypeCreateTable QueryType = iota
	QueryTypeInsert
	QueryTypeSelect
	QueryTypeCreateIndex
	QueryTypeExplain
)

var queryTypeNames = map[QueryType]string{
	QueryTypeCreateTable: "CREATE_TABLE",
	QueryTypeInsert:      "INSERT",
	QueryTypeSelect:      "SELECT",
	QueryTypeCreateIndex: "CREATE_INDEX",
	QueryTypeExplain:     "EXPLAIN",
}

func (t QueryType) String() string {
	if name, ok := queryTypeNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}
