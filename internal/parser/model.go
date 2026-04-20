package parser

// StmtType is the kind of SQL statement.
type StmtType string

const (
	StmtTypeDelete                 StmtType = "DELETE"
	StmtTypeUpdate                 StmtType = "UPDATE"
	StmtTypeDropTable              StmtType = "DROP_TABLE"
	StmtTypeDropSchema             StmtType = "DROP_SCHEMA"
	StmtTypeDropDatabase           StmtType = "DROP_DATABASE"
	StmtTypeAlterTable             StmtType = "ALTER_TABLE"
	StmtTypeAlterTableDropCol      StmtType = "ALTER_TABLE_DROP_COLUMN"
	StmtTypeAlterDefaultPrivileges StmtType = "ALTER_DEFAULT_PRIVILEGES"
	StmtTypeSelect                 StmtType = "SELECT"
	StmtTypeInsert                 StmtType = "INSERT"
	StmtTypeGrant                  StmtType = "GRANT"
	StmtTypeTruncate               StmtType = "TRUNCATE"
	StmtTypeCopy                   StmtType = "COPY"
	StmtTypeOther                  StmtType = "OTHER"
)

// Statement is an analyzer-friendly representation of one SQL statement.
type Statement struct {
	Type                  StmtType
	RawSQL                string
	StartLine             int
	Table                 string // target table name when applicable
	Schema                string
	Object                string
	HasWhere              bool
	WhereTrivial          bool
	HasLimit              bool
	SelectAll             bool
	DropColumn            bool
	DropConstraint        bool
	DropNotNull           bool
	CopyToStdout          bool
	CopyToProgram         bool
	IsGrantToPublic       bool
	IsRoleMembershipGrant bool
	GrantedRoles          []string
	Grantees              []string
}
