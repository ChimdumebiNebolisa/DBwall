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
- Average runtime per case: `4.481 ms`

## Assumptions and Definitions

- Coverage mode is taken from the built `dbguard` JSON `coverage_mode` field (`full`).
- Cases marked `requires_full` are skipped when coverage mode is not `full`.
- Positive class for precision/recall: `block`
- accuracy is exact decision match rate across allow, warn, and block
- average runtime per case is the arithmetic mean wall-clock runtime of one sequential CLI execution per case after one uncaptured warmup command

## Case Results

| ID | Category | Expected | Actual | Exact Match | Runtime (ms) |
| --- | --- | --- | --- | --- | ---: |
| allow_dollar_quoted | benign | allow | allow | true | 4.603 |
| allow_exists_predicate | benign | allow | allow | true | 4.832 |
| allow_keyword_ident | benign | allow | allow | true | 4.092 |
| allow_safe_insert | benign | allow | allow | true | 4.436 |
| allow_scoped_update | benign | allow | allow | true | 4.540 |
| allow_select_constant | benign | allow | allow | true | 4.510 |
| block_copy_select_stdout | dangerous | block | block | true | 4.373 |
| block_copy_table_stdout | dangerous | block | block | true | 4.511 |
| block_cte_delete | dangerous | block | block | true | 4.179 |
| block_default_privileges | dangerous | block | block | true | 4.202 |
| block_delete_true_or | dangerous | block | block | true | 4.273 |
| block_delete_using | dangerous | block | block | true | 4.226 |
| block_delete_without_where | dangerous | block | block | true | 4.792 |
| block_drop_table_multi | dangerous | block | block | true | 4.401 |
| block_grant_multi | dangerous | block | block | true | 4.836 |
| block_grant_public_protected | dangerous | block | block | true | 4.276 |
| block_role_grant | dangerous | block | block | true | 4.332 |
| block_truncate_multi | dangerous | block | block | true | 4.308 |
| block_truncate_table | dangerous | block | block | true | 3.935 |
| borderline_cte_protected | borderline | warn | warn | true | 4.965 |
| borderline_join_protected | borderline | warn | warn | true | 4.473 |
| borderline_nested_subquery | borderline | warn | warn | true | 4.617 |
| borderline_protected_select_star_limit | borderline | warn | warn | true | 4.404 |
| borderline_protected_select_without_limit | borderline | warn | warn | true | 4.225 |
| borderline_protected_update | borderline | warn | warn | true | 4.945 |
| borderline_schema_quoted | borderline | warn | warn | true | 5.090 |
| warn_insert_select_protected | dangerous | warn | warn | true | 4.771 |
| warn_unsupported_create_index | dangerous | warn | warn | true | 4.321 |
| warn_update_from_protected | dangerous | warn | warn | true | 4.490 |
