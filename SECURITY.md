# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue
2. Email the maintainer directly or use GitHub's private vulnerability reporting
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

## Release Signing & Verification

All releases are signed with GPG to ensure authenticity and integrity. The Terraform CLI automatically verifies signatures during `terraform init`.

### Verifying Releases Manually

1. Download the release artifacts:
   - `terraform-provider-turingpi_X.Y.Z_SHA256SUMS`
   - `terraform-provider-turingpi_X.Y.Z_SHA256SUMS.sig`
   - The zip file for your platform

2. Import the public key from the [Terraform Registry](https://registry.terraform.io/providers/freed-dev-llc/turingpi)

3. Verify the signature:
   ```bash
   gpg --verify terraform-provider-turingpi_X.Y.Z_SHA256SUMS.sig terraform-provider-turingpi_X.Y.Z_SHA256SUMS
   ```

4. Verify the checksum:
   ```bash
   sha256sum -c terraform-provider-turingpi_X.Y.Z_SHA256SUMS
   ```

### Key Management

- GPG keys are rotated periodically for security
- Old public keys remain in the registry to allow verification of previous releases
- The current signing key is registered with the Terraform Registry

## Security Considerations

This provider handles sensitive data:

- **Credentials**: BMC username and password are used for authentication
- **Network Access**: Communicates with Turing Pi BMC over HTTPS
- **Firmware Files**: Handles firmware images for flashing

### Best Practices

1. **Use environment variables** for credentials instead of hardcoding in `.tf` files:
   ```bash
   export TURINGPI_USERNAME="root"
   export TURINGPI_PASSWORD="your-password"
   ```

2. **Never commit** `terraform.tfstate` files containing credentials to version control
   - Add `*.tfstate` and `*.tfstate.backup` to `.gitignore`
   - Use remote state backends with encryption (S3, GCS, Terraform Cloud)

3. **Use HTTPS** endpoints (the default) for BMC communication

4. **Verify firmware images** before flashing to nodes

5. **Enable TLS verification** (default) - only use `insecure = true` in development environments

## Supply Chain Security

This repository implements security best practices:

- **Pinned Actions**: All GitHub Actions are pinned to SHA commits
- **Dependabot**: Automated security updates for Go modules and Actions
- **Signed Releases**: All releases are GPG-signed
- **Branch Protection**: Main branch requires review and passing CI

## Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Release**: Depends on severity and complexity
