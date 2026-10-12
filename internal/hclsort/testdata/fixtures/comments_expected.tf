# Top level floating comment (extra newline before a block)

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
# Top level trailing comment

# Leading comment
locals {
  # Leading attribute comment for alpha
  alpha_attribute = "alpha-value"

  # Comment before zeta, with extra newline before the attribute

  # Leading attribute comment for zeta
  zeta_attribute = "zeta-value"
  # Trailing attribute comment
}
# Trailing comment at end of file
