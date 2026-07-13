# DBwall Benchmark Report

## Measured Results

- Coverage mode: `full`
- Total cases: `29`
- Correct blocks: `13`
- Correct allows: `6`
- Correct warns: `10`
- False positives: `0`
- False negatives: `0`
- Precision (`block` as positive class): `1.0000`
- Recall (`block` as positive class): `1.0000`
- Accuracy (exact decision match): `1.0000`
- Average runtime per case: `4.157 ms`

## Assumptions and Definitions

- Coverage mode is taken from the built `dbguard` JSON `coverage_mode` field (`full`).
- Cases marked `requires_full` are skipped when coverage mode is not `full`.
- Positive class for precision/recall: `block`
- accuracy is exact decision match rate across allow, warn, and block
- average runtime per case is the arithmetic mean wall-clock runtime of one sequential CLI execution per case after one uncaptured warmup command

## Case Results

| ID | Category | Expected | Actual | Exact Match | Runtime (ms) |
| --- | --- | --- | --- | --- | ---: |
| allow_dollar_quoted | benign | allow | allow | true | 4.046 |
| allow_exists_predicate | benign | allow | allow | true | 3.984 |
| allow_keyword_ident | benign | allow | allow | true | 4.239 |
| allow_safe_insert | benign | allow | allow | true | 3.993 |
| allow_scoped_update | benign | allow | allow | true | 4.174 |
| allow_select_constant | benign | allow | allow | true | 4.275 |
| block_copy_select_stdout | dangerous | block | block | true | 4.028 |
| block_copy_table_stdout | dangerous | block | block | true | 4.742 |
| block_cte_delete | dangerous | block | block | true | 3.773 |
| block_default_privileges | dangerous | block | block | true | 3.918 |
| block_delete_true_or | dangerous | block | block | true | 4.053 |
| block_delete_using | dangerous | block | block | true | 4.118 |
| block_delete_without_where | dangerous | block | block | true | 3.852 |
| block_drop_table_multi | dangerous | block | block | true | 4.137 |
| block_grant_multi | dangerous | block | block | true | 4.085 |
| block_grant_public_protected | dangerous | block | block | true | 4.287 |
| block_role_grant | dangerous | block | block | true | 4.067 |
| block_truncate_multi | dangerous | block | block | true | 4.837 |
| block_truncate_table | dangerous | block | block | true | 4.022 |
| borderline_cte_protected | borderline | warn | warn | true | 5.000 |
| borderline_join_protected | borderline | warn | warn | true | 4.012 |
| borderline_nested_subquery | borderline | warn | warn | true | 5.004 |
| borderline_protected_select_star_limit | borderline | warn | warn | true | 3.954 |
| borderline_protected_select_without_limit | borderline | warn | warn | true | 4.087 |
| borderline_protected_update | borderline | warn | warn | true | 3.886 |
| borderline_schema_quoted | borderline | warn | warn | true | 3.999 |
| warn_insert_select_protected | dangerous | warn | warn | true | 4.073 |
| warn_unsupported_create_index | dangerous | warn | warn | true | 3.934 |
| warn_update_from_protected | dangerous | warn | warn | true | 3.988 |
