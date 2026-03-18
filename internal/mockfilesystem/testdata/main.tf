resource "random_pet" "name" {
  keepers = {
    # An example resource w/ JSON Schema input
    pet_name = "${var.md_metadata.name_prefix}"
  }
}
