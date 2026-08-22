# DBwall Benchmark Report

## Measured Results

- Coverage mode: `core`
- Total cases: `26`
- Correct blocks: `10`
- Correct allows: `9`
- Correct warns: `7`
- False positives: `0`
- False negatives: `0`
- Precision (`block` as positive class): `1.0000`
- Recall (`block` as positive class): `1.0000`
- Accuracy (exact decision match): `1.0000`
- Average runtime per case: `198.108 ms`

## Assumptions and Definitions

- Coverage mode is taken from the built `dbguard` JSON `coverage_mode` field (`core`).
- Cases marked `requires_full` are skipped when coverage mode is not `full`.
- Positive class for precision/recall: `block`
- accuracy is exact decision match rate across allow, warn, and block
- average runtime per case is the arithmetic mean wall-clock runtime of one sequential CLI execution per case after one uncaptured warmup command

## Case Results

| ID | Category | Expected | Actual | Exact Match | Runtime (ms) |
| --- | --- | --- | --- | --- | ---: |
| allow_comment_only | benign | allow | allow | true | 177.298 |
| allow_dollar_quoted | benign | allow | allow | true | 163.177 |
| allow_exists_predicate | benign | allow | allow | true | 179.675 |
| allow_keyword_ident | benign | allow | allow | true | 174.093 |
| allow_revoke_all_protected | benign | allow | allow | true | 157.539 |
| allow_revoke_role_highrisk | benign | allow | allow | true | 339.230 |
| allow_safe_insert | benign | allow | allow | true | 190.591 |
| allow_scoped_update | benign | allow | allow | true | 420.500 |
| allow_select_constant | benign | allow | allow | true | 197.659 |
| block_bom_delete | dangerous | block | block | true | 149.998 |
| block_copy_table_stdout | dangerous | block | block | true | 171.763 |
| block_default_privileges | dangerous | block | block | true | 233.224 |
| block_delete_without_where | dangerous | block | block | true | 302.575 |
| block_grant_multi | dangerous | block | block | true | 229.763 |
| block_grant_public_protected | dangerous | block | block | true | 182.362 |
| block_role_grant | dangerous | block | block | true | 216.558 |
| block_truncate_table | dangerous | block | block | true | 200.175 |
| block_unicode_ident_delete | dangerous | block | block | true | 229.097 |
| block_update_set_subquery_trivial_where | dangerous | block | block | true | 179.953 |
| borderline_nested_subquery | borderline | warn | warn | true | 181.118 |
| borderline_protected_select_star_limit | borderline | warn | warn | true | 151.773 |
| borderline_protected_select_without_limit | borderline | warn | warn | true | 161.841 |
| borderline_protected_update | borderline | warn | warn | true | 147.817 |
| warn_alter_role_superuser | dangerous | warn | warn | true | 133.793 |
| warn_drop_type_unsupported | dangerous | warn | warn | true | 138.694 |
| warn_fetch_first_bounded | borderline | warn | warn | true | 140.537 |
