# Audit Log Table Configuration
#
# Use case: Immutable audit trail for compliance and security monitoring
# - Compound primary key (entity_id + timestamp) for chronological audit records
# - Streams enabled for real-time audit log processing and alerting
# - PITR enabled for compliance and data recovery requirements
# - Provisioned capacity for consistent audit logging workload

capacity = {
  billing_mode   = "PROVISIONED"
  read_capacity  = 20   # Moderate read capacity for audit log queries
  write_capacity = 50   # High write capacity for continuous audit logging
}

global_secondary_indexes = [
  {
    attributes = {
      partition_key      = "user_id"
      partition_key_type = "S"
      sort_key           = "timestamp"
      sort_key_type      = "N"
    }
    name            = "user-audit-index"
    projection_type = "ALL"  # Full projection for complete audit records
    read_capacity   = 10
    write_capacity  = 50
  },
  {
    attributes = {
      partition_key      = "action_type"
      partition_key_type = "S"
      sort_key           = "timestamp"
      sort_key_type      = "N"
    }
    name            = "action-type-index"
    projection_type = "ALL"
    read_capacity   = 10
    write_capacity  = 50
  }
]

pitr = {
  enabled = true  # Critical for compliance and audit trail recovery
}

primary_index = {
  type               = "compound"
  partition_key      = "entity_id"
  partition_key_type = "S"
  sort_key           = "timestamp"
  sort_key_type      = "N"  # Number type for chronological ordering
}

region = "us-east-1"

stream = {
  enabled   = true
  view_type = "NEW_AND_OLD_IMAGES"  # Full stream for real-time audit monitoring
}

ttl = {
  enabled = false  # Audit logs retained indefinitely for compliance
}
