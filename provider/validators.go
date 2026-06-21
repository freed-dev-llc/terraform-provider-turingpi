package provider

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// validateNodeID enforces the BMC's 1-4 node range. Shared by every resource
// and data source that takes a node number so the invariant lives in one place.
var validateNodeID schema.SchemaValidateDiagFunc = validation.ToDiagFunc(validation.IntBetween(1, 4))

// hostnameLabelRE matches a single lowercase RFC-1123 DNS label (a node
// hostname): letters, digits, and hyphens; 1-63 chars; no leading or trailing
// hyphen.
var hostnameLabelRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// validateHostname accepts an empty string (the provider fills in a default
// such as turing-cp-N) or a valid lowercase RFC-1123 hostname label. An invalid
// hostname is rejected at plan time rather than failing later inside the node's
// OS config apply.
var validateHostname schema.SchemaValidateDiagFunc = validation.ToDiagFunc(
	func(i interface{}, k string) ([]string, []error) {
		v, ok := i.(string)
		if !ok {
			return nil, []error{fmt.Errorf("%s: expected a string", k)}
		}
		if v == "" || hostnameLabelRE.MatchString(v) {
			return nil, nil
		}
		return nil, []error{fmt.Errorf(
			"%s: %q must be a valid lowercase RFC-1123 hostname label (letters, digits, hyphens; 1-63 chars; no leading or trailing hyphen) or empty to use the default",
			k, v)}
	},
)
