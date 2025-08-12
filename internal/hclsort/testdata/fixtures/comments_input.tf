# [removed] Top level floating comment (extra newline before a block)

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
# [removed] Top level trailing comment


# Leading comment
locals {
  # [removed] Comment inside a locals block with extra newline before an attribute

  # Leading attribute comment
  some_attribute = "value"
  # [removed] Trailing attribute comment
}
# [removed] Trailing attribute comment
