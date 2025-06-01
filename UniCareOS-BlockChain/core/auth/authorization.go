package auth

import (
	"unicareos/core/wallet"
	"unicareos/core/audit"
	"fmt"
	"time"
)

type Authorizer struct {
	WalletVerifier wallet.SignatureVerifier
	EthosVerifier  *EthosVerifier
	AuditLogger    audit.AuditLogger
}

type AuthorizationResult struct {
	Authorized bool
	Reason     string
}

// AuthorizeAction checks wallet signature and Ethos token.
func (a *Authorizer) AuthorizeAction(payload []byte, signature []byte, walletAddr string, ethosToken string) AuthorizationResult {
	if !a.WalletVerifier.VerifySignature(payload, signature, walletAddr) {
		a.AuditLogger.LogEvent(audit.AuditEvent{
			EventType: "SignatureVerification",
			EntityID:  walletAddr,
			Result:    "failure",
			Reason:    "Invalid wallet signature",
			Metadata:  map[string]string{},
			Timestamp:  time.Now(),
		})
		return AuthorizationResult{false, "Invalid wallet signature"}
	}
	claims, err := a.EthosVerifier.VerifyEthosToken(ethosToken)
	if err != nil {
		a.AuditLogger.LogEvent(audit.AuditEvent{
			EventType: "EthosTokenVerification",
			EntityID:  walletAddr,
			Result:    "failure",
			Reason:    err.Error(),
			Metadata:  map[string]string{},
			Timestamp:  time.Now(),
		})
		return AuthorizationResult{false, "Invalid Ethos token: " + err.Error()}
	}
	// Additional claim checks (roles, expiry, etc.) can be added here
	a.AuditLogger.LogEvent(audit.AuditEvent{
		EventType: "Authorization",
		EntityID:  walletAddr,
		Result:    "success",
		Reason:    "Authorized",
		Metadata:  map[string]string{"roles": fmt.Sprintf("%v", claims.Roles)},
		Timestamp:  time.Now(),
	})
	return AuthorizationResult{true, "Authorized"}
}
