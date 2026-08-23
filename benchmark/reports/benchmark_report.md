# DBwall Benchmark Report

## Measured Results

- Coverage mode: `full`
- Total cases: `42`
- Correct blocks: `17`
- Correct allows: `9`
- Correct warns: `16`
- False positives: `0`
- False negatives: `0`
- Precision (`block` as positive class): `1.0000`
- Recall (`block` as positive class): `1.0000`
- Accuracy (exact decision match): `1.0000`
- Average runtime per case: `456.861 ms`

## Assumptions and Definitions

- Coverage mode is taken from the built `dbguard` JSON `coverage_mode` field (`full`).
- Cases marked `requires_full` are skipped when coverage mode is not `full`.
- Positive class for precision/recall: `block`
- accuracy is exact decision match rate across allow, warn, and block
- average runtime per case is the arithmetic mean wall-clock runtime of one sequential CLI execution per case after one uncaptured warmup command

## Case Results

| ID | Category | Expected | Actual | Exact Match | Runtime (ms) |
| --- | --- | --- | --- | --- | ---: |
| allow_comment_only | benign | allow | allow | true | 253.944 |
| allow_dollar_quoted | benign | allow | allow | true | 283.400 |
| allow_exists_predicate | benign | allow | allow | true | 271.415 |
| allow_keyword_ident | benign | allow | allow | true | 264.703 |
| allow_revoke_all_protected | benign | allow | allow | true | 257.138 |
| allow_revoke_role_highrisk | benign | allow | allow | true | 253.238 |
| allow_safe_insert | benign | allow | allow | true | 267.971 |
| allow_scoped_update | benign | allow | allow | true | 276.581 |
| allow_select_constant | benign | allow | allow | true | 300.481 |
| block_begin_commit_wrapper | dangerous | block | block | true | 373.797 |
| block_bom_delete | dangerous | block | block | true | 343.564 |
| block_copy_select_stdout | dangerous | block | block | true | 315.700 |
| block_copy_table_stdout | dangerous | block | block | true | 318.858 |
| block_cte_delete | dangerous | block | block | true | 355.139 |
| block_default_privileges | dangerous | block | block | true | 296.163 |
| block_delete_true_or | dangerous | block | block | true | 282.777 |
| block_delete_using | dangerous | block | block | true | 270.994 |
| block_delete_without_where | dangerous | block | block | true | 306.747 |
| block_drop_table_multi | dangerous | block | block | true | 313.928 |
| block_grant_multi | dangerous | block | block | true | 283.402 |
| block_grant_public_protected | dangerous | block | block | true | 434.108 |
| block_role_grant | dangerous | block | block | true | 308.570 |
| block_truncate_multi | dangerous | block | block | true | 396.636 |
| block_truncate_table | dangerous | block | block | true | 326.626 |
| block_unicode_ident_delete | dangerous | block | block | true | 269.786 |
| block_update_set_subquery_trivial_where | dangerous | block | block | true | 378.423 |
| borderline_cte_protected | borderline | warn | warn | true | 667.843 |
| borderline_join_protected | borderline | warn | warn | true | 887.916 |
| borderline_nested_subquery | borderline | warn | warn | true | 513.435 |
| borderline_protected_select_star_limit | borderline | warn | warn | true | 829.483 |
| borderline_protected_select_without_limit | borderline | warn | warn | true | 822.285 |
| borderline_protected_update | borderline | warn | warn | true | 2199.545 |
| borderline_schema_quoted | borderline | warn | warn | true | 661.748 |
| warn_alter_role_superuser | dangerous | warn | warn | true | 428.073 |
| warn_do_block_unsupported | dangerous | warn | warn | true | 501.271 |
| warn_drop_type_unsupported | dangerous | warn | warn | true | 562.841 |
| warn_fetch_first_bounded | borderline | warn | warn | true | 409.358 |
| warn_insert_select_protected | dangerous | warn | warn | true | 468.557 |
| warn_insert_select_read_source | dangerous | warn | warn | true | 576.139 |
| warn_union_arm_limit | dangerous | warn | warn | true | 303.725 |
| warn_unsupported_create_index | dangerous | warn | warn | true | 333.263 |
| warn_update_from_protected | dangerous | warn | warn | true | 1018.592 |
