@AGENTS:MBVM:3.2.0%lang=go%mode=solo%engine=mbvm-db

^mbvm_code {
  agents_only_mbvm!
  skills_only_mbvm!
  mcps_only_mbvm_1line!
  primary_lang_go!
  mbvm_db_slicing_first[mbvm-db slice . <symbol> <depth>]!
  codebase_mbvm_root!
  solo_agent_only!
  compiler_as_truth!
  root_cause_first!
  targeted_minimal_diffs!
  no_import_cycles!
}

^mbvm_workflow {
  phases=[plan,reproduce_test,implement,go_vet,go_test,compile_mbvm,inspect_mbvm_db,commit]
  tools=[go build,go vet,go test,mbvm-db slice,go-tools mbvm .,mbvm-db inspect,git commit]
  enforcement=[mbvm_db_first,codebase_mbvm_required,go_idiomatic,compiler_driven_feedback]
}

^R21:mode=strict_solo!direct_agent_execution_only
