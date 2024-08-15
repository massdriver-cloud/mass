// Auto-generated variable declarations from massdriver.yaml
variable "md_metadata" {
  type = object({
    default_tags = object({
      managed-by  = string
      md-manifest = string
      md-package  = string
      md-project  = string
      md-target   = string
    })
    deployment = object({
      id = string
    })
    name_prefix = string
    observability = object({
      alarm_webhook_url = string
    })
    package = object({
      created_at             = string
      deployment_enqueued_at = string
      previous_status        = string
      updated_at             = string
    })
    target = object({
      contact_email = string
    })
  })
}
// Auto-generated variable declarations from massdriver.yaml
variable "draft_node_foo" {
  type = object({
    data = object({
      name = string
    })
    specs = object({})
  })
}
variable "foo" {
  type = object({
    bar = number
    qux = optional(number)
  })
  default = null
}
variable "resource_name" {
  type    = string
  default = null
}
variable "resource_type" {
  type = string
}
