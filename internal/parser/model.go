package parser

import (
	"fmt"
	"sort"
	"strings"
)

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

// AnalysisCompleteness reports how thoroughly statement semantics were extracted.
type AnalysisCompleteness string

const (
	AnalysisComplete    AnalysisCompleteness = "complete"
	AnalysisPartial     AnalysisCompleteness = "partial"
	AnalysisUnsupported AnalysisCompleteness = "unsupported"
)

// RelationRole describes how a statement uses a relation.
type RelationRole string

const (
	RelationRead   RelationRole = "read"
	RelationWrite  RelationRole = "write"
	RelationTarget RelationRole = "target"
)

// PredicateAssessment classifies a WHERE/filter predicate.
type PredicateAssessment string

const (
	PredicateAbsent     PredicateAssessment = "absent"
	PredicateNonTrivial PredicateAssessment = "non_trivial"
	PredicateAlwaysTrue PredicateAssessment = "always_true"
	PredicateUnknown    PredicateAssessment = "unknown"
)

// RelationRef is one schema-qualified relation reference.
type RelationRef struct {
	Schema string
	Name   string
	Role   RelationRole
}

// QualifiedName returns schema.name or name.
func (r RelationRef) QualifiedName() string {
	if r.Schema == "" {
		return r.Name
	}
	return r.Schema + "." + r.Name
}

// Statement is an analyzer-friendly semantic representation of one SQL statement.
// Relations is the source of truth for relation references. Table/Object/Schema
// remain as unambiguous convenience accessors derived from that set.
type Statement struct {
	Type                  StmtType
	RawSQL                string
	StartLine             int
	Relations             []RelationRef
	Completeness          AnalysisCompleteness
	IncompleteReasons     []string
	Predicate             PredicateAssessment
	Table                 string // convenience: primary display relation; not the authority for protected-object checks
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
	Nested                []Statement
}

// ReadRelations returns relation refs with read role.
func (s Statement) ReadRelations() []RelationRef {
	return filterRelations(s.Relations, RelationRead)
}

// WriteRelations returns relation refs with write role.
func (s Statement) WriteRelations() []RelationRef {
	return filterRelations(s.Relations, RelationWrite)
}

// TargetRelations returns relation refs with target role.
func (s Statement) TargetRelations() []RelationRef {
	return filterRelations(s.Relations, RelationTarget)
}

// MutationRelations returns write and target relations (deterministic order).
func (s Statement) MutationRelations() []RelationRef {
	var out []RelationRef
	for _, rel := range s.Relations {
		if rel.Role == RelationWrite || rel.Role == RelationTarget {
			out = append(out, rel)
		}
	}
	return out
}

// AllRelationNames returns sorted unique qualified names across roles.
func (s Statement) AllRelationNames() []string {
	seen := map[string]struct{}{}
	var out []string
	for _, rel := range s.Relations {
		name := rel.QualifiedName()
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// AnalysisIsComplete reports whether semantics were extracted completely.
func (s Statement) AnalysisIsComplete() bool {
	return s.Completeness == AnalysisComplete || s.Completeness == ""
}

func filterRelations(rels []RelationRef, role RelationRole) []RelationRef {
	var out []RelationRef
	for _, rel := range rels {
		if rel.Role == role {
			out = append(out, rel)
		}
	}
	return out
}

func addRelation(stmt *Statement, schema, name string, role RelationRole) {
	schema = strings.TrimSpace(schema)
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	ref := RelationRef{Schema: schema, Name: name, Role: role}
	qual := ref.QualifiedName()
	for _, existing := range stmt.Relations {
		if existing.Role == role && existing.QualifiedName() == qual {
			return
		}
	}
	stmt.Relations = append(stmt.Relations, ref)
}

func addRelationQualified(stmt *Statement, qualified string, role RelationRole) {
	schema, name := splitQualified(qualified)
	addRelation(stmt, schema, name, role)
}

func splitQualified(name string) (schema, rel string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return "", parts[0]
	}
	return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
}

func markIncomplete(stmt *Statement, completeness AnalysisCompleteness, reason string) {
	if completeness == AnalysisComplete {
		return
	}
	switch stmt.Completeness {
	case AnalysisUnsupported:
		// keep unsupported
	case AnalysisPartial:
		if completeness == AnalysisUnsupported {
			stmt.Completeness = AnalysisUnsupported
		}
	default:
		stmt.Completeness = completeness
	}
	if reason == "" {
		return
	}
	for _, existing := range stmt.IncompleteReasons {
		if existing == reason {
			return
		}
	}
	stmt.IncompleteReasons = append(stmt.IncompleteReasons, reason)
}

func syncConvenienceFields(stmt *Statement) {
	if stmt.Predicate == "" {
		if stmt.HasWhere {
			if stmt.WhereTrivial {
				stmt.Predicate = PredicateAlwaysTrue
			} else {
				stmt.Predicate = PredicateNonTrivial
			}
		} else {
			stmt.Predicate = PredicateAbsent
		}
	} else {
		stmt.HasWhere = stmt.Predicate != PredicateAbsent
		stmt.WhereTrivial = stmt.Predicate == PredicateAlwaysTrue
	}

	if stmt.Completeness == "" {
		stmt.Completeness = AnalysisComplete
	}

	primary := primaryRelation(*stmt)
	if primary.Name != "" {
		qual := primary.QualifiedName()
		if stmt.Table == "" {
			stmt.Table = qual
		}
		if stmt.Object == "" {
			stmt.Object = qual
		}
		if stmt.Schema == "" {
			stmt.Schema = primary.Schema
			if stmt.Schema == "" {
				stmt.Schema = relationSchema(qual)
			}
		}
	}
}

func primaryRelation(stmt Statement) RelationRef {
	prefer := []RelationRole{RelationWrite, RelationTarget, RelationRead}
	for _, role := range prefer {
		for _, rel := range stmt.Relations {
			if rel.Role == role {
				return rel
			}
		}
	}
	if len(stmt.Relations) > 0 {
		return stmt.Relations[0]
	}
	return RelationRef{}
}

func formatIncompleteReason(reasons []string) string {
	if len(reasons) == 0 {
		return "semantic coverage is incomplete"
	}
	return strings.Join(reasons, "; ")
}

// IncompleteSummary returns a bounded, non-sensitive incompleteness explanation.
func (s Statement) IncompleteSummary() string {
	return formatIncompleteReason(s.IncompleteReasons)
}

// EnsureReasonsBounded trims reasons to a small deterministic set.
func ensureReasonsBounded(reasons []string, limit int) []string {
	if limit <= 0 {
		limit = 5
	}
	if len(reasons) <= limit {
		return reasons
	}
	out := append([]string{}, reasons[:limit]...)
	out = append(out, fmt.Sprintf("and_%d_more", len(reasons)-limit))
	return out
}
