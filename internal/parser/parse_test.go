package parser

import (
	"testing"
)

func TestParse_SingleStatement(t *testing.T) {
	// M3.2: single statement
	stmts, err := Parse("DELETE FROM users;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if stmts[0].Type != StmtTypeDelete {
		t.Errorf("want type DELETE, got %s", stmts[0].Type)
	}
	if stmts[0].Table != "users" {
		t.Errorf("want table users, got %q", stmts[0].Table)
	}
	if stmts[0].HasWhere {
		t.Error("want HasWhere false for DELETE WITHOUT where")
	}
}

func TestParse_MultiStatement(t *testing.T) {
	// M3.3: multi-statement
	sql := "DELETE FROM a; DELETE FROM b;"
	stmts, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 2 {
		t.Fatalf("want 2 statements, got %d", len(stmts))
	}
	if stmts[0].Table != "a" || stmts[1].Table != "b" {
		t.Errorf("want tables a and b, got %q and %q", stmts[0].Table, stmts[1].Table)
	}
}

func TestParse_InvalidSQL(t *testing.T) {
	_, err := Parse("DELETE FROMM users;")
	if err == nil {
		t.Fatal("expected error for invalid SQL")
	}
}

func TestParse_UpdateWithWhere(t *testing.T) {
	stmts, err := Parse("UPDATE users SET x = 1 WHERE id = 1;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if stmts[0].Type != StmtTypeUpdate {
		t.Errorf("want type UPDATE, got %s", stmts[0].Type)
	}
	if !stmts[0].HasWhere {
		t.Error("want HasWhere true")
	}
	if stmts[0].Table != "users" {
		t.Errorf("want table users, got %q", stmts[0].Table)
	}
}

func TestParse_UpdateWithoutWhere(t *testing.T) {
	stmts, err := Parse("UPDATE users SET x = 1;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if stmts[0].HasWhere {
		t.Error("want HasWhere false")
	}
}

func TestParse_DropTable(t *testing.T) {
	stmts, err := Parse("DROP TABLE users;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if stmts[0].Type != StmtTypeDropTable {
		t.Errorf("want type DROP_TABLE, got %s", stmts[0].Type)
	}
	if stmts[0].Table != "users" {
		t.Errorf("want table users, got %q", stmts[0].Table)
	}
}

func TestParse_AlterTableDropColumn(t *testing.T) {
	stmts, err := Parse("ALTER TABLE t1 DROP COLUMN c1;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if stmts[0].Type != StmtTypeAlterTableDropCol {
		t.Errorf("want type ALTER_TABLE_DROP_COLUMN, got %s", stmts[0].Type)
	}
	if stmts[0].Table != "t1" {
		t.Errorf("want table t1, got %q", stmts[0].Table)
	}
}

func TestParse_SelectBasic(t *testing.T) {
	stmts, err := Parse("SELECT * FROM users;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if stmts[0].Type != StmtTypeSelect {
		t.Errorf("want type SELECT, got %s", stmts[0].Type)
	}
	if stmts[0].Table != "users" {
		t.Errorf("want table users, got %q", stmts[0].Table)
	}
}

func TestParse_SelectWithWhere(t *testing.T) {
	stmts, err := Parse("SELECT * FROM users WHERE id = 1;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if !stmts[0].HasWhere {
		t.Error("want HasWhere true")
	}
}

func TestParse_SelectNoFrom(t *testing.T) {
	stmts, err := Parse("SELECT 1;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if stmts[0].Table != "" {
		t.Errorf("want empty table, got %q", stmts[0].Table)
	}
}

func TestParse_InsertBasic(t *testing.T) {
	stmts, err := Parse("INSERT INTO users (id) VALUES (1);")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if stmts[0].Type != StmtTypeInsert {
		t.Errorf("want type INSERT, got %s", stmts[0].Type)
	}
	if stmts[0].Table != "users" {
		t.Errorf("want table users, got %q", stmts[0].Table)
	}
}

func TestParse_TruncateTable(t *testing.T) {
	stmts, err := Parse("TRUNCATE TABLE users;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 || stmts[0].Type != StmtTypeTruncate {
		t.Fatalf("want TRUNCATE statement, got %#v", stmts)
	}
	if stmts[0].Table != "users" {
		t.Errorf("want table users, got %q", stmts[0].Table)
	}
}

func TestParse_DropSchema(t *testing.T) {
	stmts, err := Parse("DROP SCHEMA reporting;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 || stmts[0].Type != StmtTypeDropSchema {
		t.Fatalf("want DROP_SCHEMA statement, got %#v", stmts)
	}
	if stmts[0].Object != "reporting" {
		t.Errorf("want object reporting, got %q", stmts[0].Object)
	}
}

func TestParse_GrantRoleMembership(t *testing.T) {
	stmts, err := Parse("GRANT pg_read_all_data TO analyst;")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 || stmts[0].Type != StmtTypeGrant {
		t.Fatalf("want GRANT statement, got %#v", stmts)
	}
	if !stmts[0].IsRoleMembershipGrant {
		t.Error("expected role membership grant")
	}
	if len(stmts[0].GrantedRoles) != 1 || stmts[0].GrantedRoles[0] != "pg_read_all_data" {
		t.Errorf("unexpected granted roles: %#v", stmts[0].GrantedRoles)
	}
}

func TestParse_CopyToProgram(t *testing.T) {
	stmts, err := Parse("COPY users TO PROGRAM 'cat';")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 || stmts[0].Type != StmtTypeCopy {
		t.Fatalf("want COPY statement, got %#v", stmts)
	}
	if !stmts[0].CopyToProgram {
		t.Error("expected COPY TO PROGRAM")
	}
	if stmts[0].Table != "users" {
		t.Errorf("want table users, got %q", stmts[0].Table)
	}
}
