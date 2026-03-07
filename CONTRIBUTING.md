# Contributing to Terraform Provider for Turing Pi

Thank you for your interest in contributing to this project!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR-USERNAME/terraform-provider-turingpi.git`
3. Create a branch: `git checkout -b feature/your-feature`

## Development Setup

```bash
# Install dependencies
go mod tidy

# Build
go build -o terraform-provider-turingpi

# Run tests
go test -v ./...

# Run tests with race detection
go test -v -race ./...
```

## Pre-commit Hooks

This project uses [pre-commit](https://pre-commit.com/) to enforce code quality before commits reach CI.

### Installation

```bash
# Install pre-commit (choose one)
pip install pre-commit
# or: brew install pre-commit
# or: pipx install pre-commit

# Install the hooks
pre-commit install
pre-commit install --hook-type commit-msg

# Run against all files (optional, for first-time setup)
pre-commit run --all-files
```

### What the hooks check

- **Go**: formatting (gofmt), vetting, module tidiness, golangci-lint
- **Terraform**: formatting, validation, TFLint, terraform-docs
- **General**: trailing whitespace, YAML/JSON validity, large files, merge conflicts
- **Security**: detect-secrets for accidental credential commits
- **Commits**: conventional commit message format

## Making Changes

1. Write clear, concise commit messages
2. Add tests for new functionality
3. Ensure all tests pass before submitting
4. Update documentation if needed

## Code Style

- Follow standard Go conventions (`gofmt`, `golint`)
- Run `golangci-lint run` before submitting
- Keep functions focused and well-documented

## Pull Request Process

1. Update the CHANGELOG.md with your changes
2. Ensure CI checks pass
3. Request review from maintainers
4. Address any feedback

## Release Process (Maintainers)

Releases are automated via Makefile targets. All releases are GPG-signed and published to the [Terraform Registry](https://registry.terraform.io/providers/freed-dev-llc/turingpi).

### Creating a Release

```bash
# Full release: updates docs, runs tests, commits, tags, and pushes
make release VERSION=1.0.10
```

This command will:
1. Update version references in `README.md`, `docs/`, and `examples/`
2. Run the full test suite
3. Commit the version updates (GPG-signed)
4. Create a signed tag `v1.0.10`
5. Push to origin (triggers GitHub Actions release workflow)

### Partial Release Prep

```bash
# Just update version numbers (no commit/tag)
make release-prep VERSION=1.0.10
```

### Post-Release

After the release workflow completes:
1. Verify the release on [GitHub Releases](https://github.com/freed-dev-llc/terraform-provider-turingpi/releases)
2. Confirm it appears on [Terraform Registry](https://registry.terraform.io/providers/freed-dev-llc/turingpi)
3. Update `CHANGELOG.md` with release notes

### Requirements

- GPG key configured for commit/tag signing (`git config commit.gpgsign true`)
- GPG key registered on GitHub for verified badges
- Push access to the repository

## Testing Against Real Hardware

If you have access to a Turing Pi 2.5:

1. Set environment variables:
   ```bash
   export TURINGPI_USERNAME=root
   export TURINGPI_PASSWORD=turing
   export TURINGPI_ENDPOINT=https://your-turingpi-ip
   ```

2. Create a test Terraform configuration
3. Run `terraform plan` and `terraform apply`

## Reporting Issues

- Use the issue templates provided
- Include Terraform version, provider version, and Go version
- Provide minimal reproduction steps

## Branch Protection (Maintainers)

The `main` branch has protection rules configured in GitHub. Recommended settings:

### Required Status Checks

Enable "Require status checks to pass before merging" with these checks:
- `build` (Go build and test)
- `lint` (golangci-lint)
- `gosec` (security scanning)
- `tflint` (Terraform example validation)

### Additional Protections

- **Require pull request reviews**: At least 1 approving review
- **Dismiss stale reviews**: When new commits are pushed
- **Require review from Code Owners**: Enabled (see CODEOWNERS)
- **Require signed commits**: Recommended for verified releases
- **Require linear history**: Optional, keeps history clean
- **Do not allow bypassing**: Even admins should follow the rules

### Setting Up Branch Protection

1. Go to **Settings > Branches > Branch protection rules**
2. Click **Add rule** for `main`
3. Configure the settings above
4. Save changes

## Questions?

Open a discussion or issue if you have questions about contributing.
