@CLAUDE:3.3.0%cli=claude%lang=go%runtime=mbvm%mode=solo%engine=mbvm-db

^claude {
  follow_AGENTS_mbvm!
  global="~/.claude/AGENTS.mbvm"!
  project="./AGENTS.mbvm"!
}

^ops {
  solo_agent_execution!
  lang_primary_go!
  single_ingestion_law!
  mbvm_db_first[mbvm-db slice <db> <symbol/term> <depth>]!
  never_read_raw_catalogs_or_protobufs!
  codebase_mbvm_root!
  context_before_dump!
  compiler_as_truth[go_vet]!
  test_flags[go_test_-race_-count=1_-v]!
  root_cause_first!
  targeted_minimal_diffs!
  no_import_cycles!
  modern_stdlib[slog,slices]!
  err_wrap_w_only!no_v_in_errors!
  consumer_interfaces!
  goroutine_leak_prevention[ctx_done_select]!
  no_secrets!maskSecret()!
  max_loc_400!
  skills_mandatory!
  hook[git_global_pre_commit]!
}

^R0:mbvm_canonical!
^R1:solo_agent_only!
^R8:go_primary!
^R10:compiler_as_truth!
^R12:root_cause_first!
^R14:mbvm_db_slicing!
^R15:mbvm_first!
^R21:mode=strict_solo!
^R22:context_rich=per_project!engine=mbvm-db!subgraph_slice!codebase_mbvm!full_project_ast_map
