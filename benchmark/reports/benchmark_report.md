# DBwall Benchmark Report

## Measured Results

- Total cases: `9`
- Correct blocks: `3`
- Correct allows: `3`
- Correct warns: `3`
- False positives: `0`
- False negatives: `0`
- Precision (`block` as positive class): `1.0000`
- Recall (`block` as positive class): `1.0000`
- Accuracy (exact decision match): `1.0000`
- Average runtime per case: `91.973 ms`

## Assumptions and Definitions

- Positive class for precision/recall: `block`
- accuracy is exact decision match rate across allow, warn, and block
- average runtime per case is the arithmetic mean wall-clock runtime of one sequential CLI execution per case after one uncaptured warmup command

## Case Results

| ID | Category | Expected | Actual | Exact Match | Runtime (ms) |
| --- | --- | --- | --- | --- | ---: |
| allow_safe_insert | benign | allow | allow | true | 82.823 |
| allow_scoped_update | benign | allow | allow | true | 83.757 |
| allow_select_constant | benign | allow | allow | true | 79.899 |
| block_delete_without_where | dangerous | block | block | true | 81.760 |
| block_grant_public_protected | dangerous | block | block | true | 89.855 |
| block_truncate_table | dangerous | block | block | true | 104.232 |
| borderline_protected_select_star_limit | borderline | warn | warn | true | 93.384 |
| borderline_protected_select_without_limit | borderline | warn | warn | true | 90.216 |
| borderline_protected_update | borderline | warn | warn | true | 121.832 |
