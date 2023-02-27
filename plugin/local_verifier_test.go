package plugin

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalVerifier_VerifySignature(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "q-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	arbitraryPluginDefinition := &PluginDefinition{
		Name:    "arbitrary-plugin",
		Version: "1.0.0",
		Config:  nil,
	}
	pubKeyFile := path.Join(tmpDir, "pubkey")
	if err := os.WriteFile(pubKeyFile, signerPubKey, 0644); err != nil {
		t.Fatal(err)
	}
	sigFile := path.Join(tmpDir, arbitraryPluginDefinition.SignatureFileName())
	if err := os.WriteFile(sigFile, validSignature, 0644); err != nil {
		t.Fatal(err)
	}

	testObject, err := NewLocalVerifier(pubKeyFile, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, testObject.VerifySignature(arbitraryPluginDefinition, arbitraryChecksum))
}
