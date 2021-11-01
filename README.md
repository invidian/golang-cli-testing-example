# Go CLI testing example

This repository provides a template on how to create a testable CLI applications in Go language.

As an example, this application provides simple compression/decompression functionality to be able to
deal with reading files, environment variables and parsing flags, which is common for CLI tools.

CI for this repository requires 100% mutation score on the code to ensure high quality of test code.

`Makefile` and `.github/workflows/ci.yml` also provides good example on wiring up Go project into the CI.
