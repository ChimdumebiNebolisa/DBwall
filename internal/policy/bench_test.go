package policy

import (
	"testing"
)

func BenchmarkValidate(b *testing.B) {
	p := &Policy{
		Dialect: DialectPostgres,
		Rules: map[string]string{
			RuleDeleteWithoutWhere:     "block",
			RuleUpdateWithoutWhere:     "block",
			RuleDropTable:              "block",
			RuleDropColumn:             "block",
			RuleWritesToProtectedTable: "warn",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Validate(p)
		if err != nil {
			b.Fatal(err)
		}
	}
}
