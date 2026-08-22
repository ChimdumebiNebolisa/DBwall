# DBwall Benchmark Report

## Measured Results

- Coverage mode: `full`
- Total cases: `41`
- Correct blocks: `17`
- Correct allows: `9`
- Correct warns: `15`
- False positives: `0`
- False negatives: `0`
- Precision (`block` as positive class): `1.0000`
- Recall (`block` as positive class): `1.0000`
- Accuracy (exact decision match): `1.0000`
- Average runtime per case: `227.797 ms`

## Assumptions and Definitions

- Coverage mode is taken from the built `dbguard` JSON `coverage_mode` field (`full`).
- Cases marked `requires_full` are skipped when coverage mode is not `full`.
- Positive class for precision/recall: `block`
- accuracy is exact decision match rate across allow, warn, and block
- average runtime per case is the arithmetic mean wall-clock runtime of one sequential CLI execution per case after one uncaptured warmup command

## Case Results

| ID | Category | Expected | Actual | Exact Match | Runtime (ms) |
| --- | --- | --- | --- | --- | ---: |
| allow_comment_only | benign | allow | allow | true | 164.943 |
| allow_dollar_quoted | benign | allow | allow | true | 378.923 |
| allow_exists_predicate | benign | allow | allow | true | 244.834 |
| allow_keyword_ident | benign | allow | allow | true | 326.064 |
| allow_revoke_all_protected | benign | allow | allow | true | 202.392 |
| allow_revoke_role_highrisk | benign | allow | allow | true | 150.218 |
| allow_safe_insert | benign | allow | allow | true | 202.176 |
| allow_scoped_update | benign | allow | allow | true | 175.807 |
| allow_select_constant | benign | allow | allow | true | 158.433 |
| block_begin_commit_wrapper | dangerous | block | block | true | 216.572 |
| block_bom_delete | dangerous | block | block | true | 216.866 |
| block_copy_select_stdout | dangerous | block | block | true | 269.123 |
| block_copy_table_stdout | dangerous | block | block | true | 253.534 |
| block_cte_delete | dangerous | block | block | true | 615.315 |
| block_default_privileges | dangerous | block | block | true | 184.113 |
| block_delete_true_or | dangerous | block | block | true | 249.342 |
| block_delete_using | dangerous | block | block | true | 192.206 |
| block_delete_without_where | dangerous | block | block | true | 170.654 |
| block_drop_table_multi | dangerous | block | block | true | 213.609 |
| block_grant_multi | dangerous | block | block | true | 240.755 |
| block_grant_public_protected | dangerous | block | block | true | 169.956 |
| block_role_grant | dangerous | block | block | true | 184.167 |
| block_truncate_multi | dangerous | block | block | true | 267.330 |
| block_truncate_table | dangerous | block | block | true | 203.579 |
| block_unicode_ident_delete | dangerous | block | block | true | 170.151 |
| block_update_set_subquery_trivial_where | dangerous | block | block | true | 224.114 |
| borderline_cte_protected | borderline | warn | warn | true | 192.395 |
| borderline_join_protected | borderline | warn | warn | true | 213.477 |
| borderline_nested_subquery | borderline | warn | warn | true | 201.322 |
| borderline_protected_select_star_limit | borderline | warn | warn | true | 218.188 |
| borderline_protected_select_without_limit | borderline | warn | warn | true | 215.946 |
| borderline_protected_update | borderline | warn | warn | true | 181.843 |
| borderline_schema_quoted | borderline | warn | warn | true | 204.843 |
| warn_alter_role_superuser | dangerous | warn | warn | true | 179.119 |
| warn_do_block_unsupported | dangerous | warn | warn | true | 239.966 |
| warn_drop_type_unsupported | dangerous | warn | warn | true | 358.133 |
| warn_fetch_first_bounded | borderline | warn | warn | true | 202.531 |
| warn_insert_select_protected | dangerous | warn | warn | true | 263.123 |
| warn_union_arm_limit | dangerous | warn | warn | true | 208.119 |
| warn_unsupported_create_index | dangerous | warn | warn | true | 223.764 |
| warn_update_from_protected | dangerous | warn | warn | true | 191.717 |
