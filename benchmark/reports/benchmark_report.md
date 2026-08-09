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
- Average runtime per case: `4.272 ms`

## Assumptions and Definitions

- Coverage mode is taken from the built `dbguard` JSON `coverage_mode` field (`full`).
- Cases marked `requires_full` are skipped when coverage mode is not `full`.
- Positive class for precision/recall: `block`
- accuracy is exact decision match rate across allow, warn, and block
- average runtime per case is the arithmetic mean wall-clock runtime of one sequential CLI execution per case after one uncaptured warmup command

## Case Results

| ID | Category | Expected | Actual | Exact Match | Runtime (ms) |
| --- | --- | --- | --- | --- | ---: |
| allow_dollar_quoted | benign | allow | allow | true | 4.532 |
| allow_exists_predicate | benign | allow | allow | true | 4.385 |
| allow_keyword_ident | benign | allow | allow | true | 5.121 |
| allow_safe_insert | benign | allow | allow | true | 4.691 |
| allow_scoped_update | benign | allow | allow | true | 4.073 |
| allow_select_constant | benign | allow | allow | true | 4.539 |
| block_copy_select_stdout | dangerous | block | block | true | 4.553 |
| block_copy_table_stdout | dangerous | block | block | true | 4.122 |
| block_cte_delete | dangerous | block | block | true | 4.696 |
| block_default_privileges | dangerous | block | block | true | 3.941 |
| block_delete_true_or | dangerous | block | block | true | 3.928 |
| block_delete_using | dangerous | block | block | true | 3.802 |
| block_delete_without_where | dangerous | block | block | true | 3.965 |
| block_drop_table_multi | dangerous | block | block | true | 3.977 |
| block_grant_multi | dangerous | block | block | true | 4.016 |
| block_grant_public_protected | dangerous | block | block | true | 4.292 |
| block_role_grant | dangerous | block | block | true | 3.924 |
| block_truncate_multi | dangerous | block | block | true | 3.952 |
| block_truncate_table | dangerous | block | block | true | 4.704 |
| borderline_cte_protected | borderline | warn | warn | true | 4.908 |
| borderline_join_protected | borderline | warn | warn | true | 4.287 |
| borderline_nested_subquery | borderline | warn | warn | true | 4.680 |
| borderline_protected_select_star_limit | borderline | warn | warn | true | 3.914 |
| borderline_protected_select_without_limit | borderline | warn | warn | true | 4.083 |
| borderline_protected_update | borderline | warn | warn | true | 4.117 |
| borderline_schema_quoted | borderline | warn | warn | true | 4.046 |
| warn_insert_select_protected | dangerous | warn | warn | true | 4.065 |
| warn_unsupported_create_index | dangerous | warn | warn | true | 4.206 |
| warn_update_from_protected | dangerous | warn | warn | true | 4.356 |
