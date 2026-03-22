package parser

// StmtType is the kind of SQL statement.
type StmtType string

const (
	StmtTypeDelete            StmtType = "DELETE"
	StmtTypeUpdate            StmtType = "UPDATE"
	StmtTypeDropTable         StmtType = "DROP_TABLE"
	StmtTypeAlterTableDropCol StmtType = "ALTER_TABLE_DROP_COLUMN"
	StmtTypeSelect            StmtType = "SELECT"
	StmtTypeInsert            StmtType = "INSERT"
	StmtTypeOther             StmtType = "OTHER"
)

// Statement is an analyzer-friendly representation of one SQL statement.
type Statement struct {
	Type     StmtType
	Table    string // target table name when applicable
	HasWhere bool   // for DELETE/UPDATE
}
