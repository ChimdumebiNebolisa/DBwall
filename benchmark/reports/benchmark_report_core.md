# DBwall Benchmark Report

## Measured Results

- Coverage mode: `core`
- Total cases: `27`
- Correct blocks: `10`
- Correct allows: `9`
- Correct warns: `8`
- False positives: `0`
- False negatives: `0`
- Precision (`block` as positive class): `1.0000`
- Recall (`block` as positive class): `1.0000`
- Accuracy (exact decision match): `1.0000`
- Average runtime per case: `313.269 ms`

## Assumptions and Definitions

- Coverage mode is taken from the built `dbguard` JSON `coverage_mode` field (`core`).
- Cases marked `requires_full` are skipped when coverage mode is not `full`.
- Positive class for precision/recall: `block`
- accuracy is exact decision match rate across allow, warn, and block
- average runtime per case is the arithmetic mean wall-clock runtime of one sequential CLI execution per case after one uncaptured warmup command

## Case Results

| ID | Category | Expected | Actual | Exact Match | Runtime (ms) |
| --- | --- | --- | --- | --- | ---: |
| allow_comment_only | benign | allow | allow | true | 431.483 |
| allow_dollar_quoted | benign | allow | allow | true | 420.662 |
| allow_exists_predicate | benign | allow | allow | true | 335.580 |
| allow_keyword_ident | benign | allow | allow | true | 269.845 |
| allow_revoke_all_protected | benign | allow | allow | true | 276.807 |
| allow_revoke_role_highrisk | benign | allow | allow | true | 313.687 |
| allow_safe_insert | benign | allow | allow | true | 288.990 |
| allow_scoped_update | benign | allow | allow | true | 245.778 |
| allow_select_constant | benign | allow | allow | true | 285.382 |
| block_bom_delete | dangerous | block | block | true | 248.539 |
| block_copy_table_stdout | dangerous | block | block | true | 274.094 |
| block_default_privileges | dangerous | block | block | true | 258.276 |
| block_delete_without_where | dangerous | block | block | true | 258.041 |
| block_grant_multi | dangerous | block | block | true | 275.174 |
| block_grant_public_protected | dangerous | block | block | true | 268.673 |
| block_role_grant | dangerous | block | block | true | 403.811 |
| block_truncate_table | dangerous | block | block | true | 519.941 |
| block_unicode_ident_delete | dangerous | block | block | true | 347.813 |
| block_update_set_subquery_trivial_where | dangerous | block | block | true | 466.483 |
| borderline_nested_subquery | borderline | warn | warn | true | 274.181 |
| borderline_protected_select_star_limit | borderline | warn | warn | true | 274.168 |
| borderline_protected_select_without_limit | borderline | warn | warn | true | 284.252 |
| borderline_protected_update | borderline | warn | warn | true | 281.616 |
| warn_alter_role_superuser | dangerous | warn | warn | true | 281.274 |
| warn_drop_type_unsupported | dangerous | warn | warn | true | 288.589 |
| warn_fetch_first_bounded | borderline | warn | warn | true | 305.730 |
| warn_insert_select_read_source | dangerous | warn | warn | true | 279.402 |
