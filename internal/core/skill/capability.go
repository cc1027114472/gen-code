package skill

import "llmtrace/pkg/skillaudit"

type CapabilityAudit struct {
	Verified bool
	Summary  string
}

func VerifyCapability(root string, id string) CapabilityAudit {
	audit := skillaudit.VerifyCapability(root, id)
	return CapabilityAudit{
		Verified: audit.Verified,
		Summary:  audit.Summary,
	}
}
