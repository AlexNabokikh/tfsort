# tfsort

[![badge-gh-release](https://github.com/AlexNabokikh/tfsort/actions/workflows/release.yml/badge.svg)](https://github.com/AlexNabokikh/tfsort/actions/workflows/release.yml/badge.svg)
[![badge-gh-tests](https://github.com/AlexNabokikh/tfsort/actions/workflows/test.yml/badge.svg)](https://github.com/AlexNabokikh/tfsort/actions/workflows/test.yml/badge.svg)
[![go-report-card](https://goreportcard.com/badge/github.com/AlexNabokikh/tfsort)](https://goreportcard.com/report/github.com/AlexNabokikh/tfsort)
[![go-test-coverage](https://github.com/AlexNabokikh/tfsort/wiki/coverage.svg)](https://raw.githack.com/wiki/AlexNabokikh/tfsort/coverage.html)

![Logo](files/logo.png)

`tfsort` is a command-line utility designed for meticulous engineers who prefer to keep their Terraform variables and outputs sorted in alphabetical order.

`tfsort` also corrects any spacing issues between the blocks and removes any leading or trailing new lines in the file.

## Demo

![Demo](files/demo.gif)

### Homebrew

To install tfsort using Homebrew, run the following command:

1. Add the tap:

```bash
brew tap alexnabokikh/tfsort
```

2. Install tfsort:

```bash
brew install tfsort
```

### Binary release

To install tfsort, you can download the latest binary release from the [releases page](https://github.com/AlexNabokikh/tfsort/releases).

Alternatively, you can build from the source by cloning the repository and running `go build`.

## Usage

The basic usage of tfsort is as follows:

```bash
tfsort --file <path-to-tf-file> [--out <path-to-output-file>] [--dry-run]
```

Available flags:

- `--file`: the path to the .tf file you want to sort. This flag is optional.
- `--out`: the path to the output file. If not provided, tfsort will overwrite the input file.
- `--dry-run`: preview the changes without altering the original file.

## Examples

Here's an example of using tfsort to sort a Terraform file called `main.tf`:

```bash
tfsort variables.tf
```

This will sort the resources in `variables.tf` in place.
You can also use the `--out` flag to specify an output file:

```bash
tfsort --file variables.tf --out sorted.tf
```

This will create a new file called `sorted.tf` with the sorted resources.

## Author

This project was created by [Alexander Nabokikh](https://www.linkedin.com/in/nabokih/).

## License

This software is available under the following licenses:

- **[Apache 2.0](https://github.com/AlexNabokikh/tfsort/blob/master/LICENSE)**
