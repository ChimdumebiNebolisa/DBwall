# dbguard – Milestones

## M1. Repository and scaffold

- [x] **M1.1** Initialize Go module and CLI scaffold
- [x] **M1.2** Create repo structure and placeholder packages
- [x] **M1.3** Create README and docs skeleton
- [x] **M1.4** Add Makefile and basic developer commands

## M2. Policy system

- [x] **M2.1** Define policy structs and defaults
- [x] **M2.2** Implement YAML loading
- [x] **M2.3** Implement validation
- [x] **M2.4** Test policy behavior

## M3. Parser integration

- [x] **M3.1** Choose PostgreSQL parser and integrate wrapper
- [x] **M3.2** Parse single statement SQL
- [x] **M3.3** Parse multi-statement SQL
- [x] **M3.4** Add parser tests and document parser limitations

## M4. Rule engine

- [x] **M4.1** Implement finding and decision model
- [x] **M4.2** Implement delete_without_where
- [x] **M4.3** Implement update_without_where
- [x] **M4.4** Implement drop_table
- [x] **M4.5** Implement drop_column
- [x] **M4.6** Implement writes_to_protected_tables
- [x] **M4.7** Implement result aggregation
- [ ] **M4.8** Add rule tests

## M5. Reporting and CLI behavior

- [ ] **M5.1** Implement human output
- [ ] **M5.2** Implement JSON output
- [ ] **M5.3** Implement review-sql command
- [ ] **M5.4** Implement review-file command
- [ ] **M5.5** Implement exit code behavior
- [ ] **M5.6** Add CLI tests / smoke checks

## M6. Examples, CI, and polish

- [ ] **M6.1** Add example SQL files and policy file
- [ ] **M6.2** Add GitHub Actions workflow
- [ ] **M6.3** Tighten README and docs
- [ ] **M6.4** Final test pass and cleanup
