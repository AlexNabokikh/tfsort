# Leading comment
resource "example" "example" {
  # Leading comment before the block
  some_block {

    # Comment inside block with extra newline before an attribute

    # This is a comment before the attribute
    some_attribute = "value"
  }

  # Block level floating comment

}

# Leading comment
locals {
  # Leading attribute comment
  some_attribute = "value"
}
