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
