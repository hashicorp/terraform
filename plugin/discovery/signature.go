package discovery

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

// verifyProviderSignature checks the ASCII-armored signature of the SHA256SUMS
// file against the author's key. The trust signature is also verified if
// present.
//
// The tier is determined by the following logic:
// -  If the author key is HashiCorp's public key, the tier is Official
// -  If a trust signature is verified for the author key, the tier is Partner.
// -  Otherwise (including on failure), the tier defaults to Community.
func verifyProviderSignature(shasums, shasumsSignature []byte, authorKeyArmor, trustSignatureArmor string) (*openpgp.Entity, ProviderTier, error) {
	// Read the provider author's ASCII-armored public key.
	el, err := openpgp.ReadArmoredKeyRing(
		strings.NewReader(authorKeyArmor))
	if err != nil {
		return nil, providerTierCommunity, err
	}

	// Verify that the SHA256SUMS.sig file is a signature of the SHA256SUMS
	// file using the provider author's public key.
	entity, err := openpgp.CheckDetachedSignature(
		el, bytes.NewReader(shasums), bytes.NewReader(shasumsSignature))
	if err != nil {
		return nil, providerTierCommunity, err
	}

	// Check if this is signed with the HashiCorp key, and if so, return the
	// tier as Official. We verify the signature again to protect against key
	// ID/fingerprint collisions.
	hashicorp, err := openpgp.ReadArmoredKeyRing(
		strings.NewReader(HashicorpPublicKey))
	if err != nil {
		return nil, providerTierCommunity, err
	}
	hashicorpEntity, err := openpgp.CheckDetachedSignature(
		hashicorp, bytes.NewReader(shasums), bytes.NewReader(shasumsSignature))
	if err == nil {
		return hashicorpEntity, providerTierOfficial, err
	}

	// If there's a trust signature present, we verify that the trust signature
	// is signing the public key data of the author key with the
	// HashicorpPartnersKey. If this verification passes, then the tier is
	// Partner.
	if trustSignatureArmor != "" {
		// Create a keyring for the HashicorpPartnersKey.
		hashicorpPartnersKeyring, err := openpgp.ReadArmoredKeyRing(
			strings.NewReader(HashicorpPartnersKey))
		if err != nil {
			return nil, providerTierCommunity,
				fmt.Errorf("Error creating hashicorpPartnersKeyring: %s", err)
		}

		// Extract the raw key data from authorKeyArmor.
		authorKey, err := armor.Decode(
			strings.NewReader(authorKeyArmor))
		if err != nil {
			return nil, providerTierCommunity,
				fmt.Errorf("Error parsing authorKeyArmor: %s", err)
		}

		// Parse the trustSignatureArmor.
		trustSignature, err := armor.Decode(
			strings.NewReader(trustSignatureArmor))
		if err != nil {
			return nil, providerTierCommunity,
				fmt.Errorf("Error parsing trustSignatureArmor: %s", err)
		}

		// Check that trustSignatureArmor is a valid signature of authorKey
		// with HashicorpPartnersKey.
		_, err = openpgp.CheckDetachedSignature(
			hashicorpPartnersKeyring, authorKey.Body, trustSignature.Body)
		if err != nil {
			return nil, providerTierCommunity,
				fmt.Errorf("Error verifying trust signature: %s", err)

		}

		return entity, providerTierPartner, nil
	}

	return entity, providerTierCommunity, nil
}

// entityString extracts the key ID and identity name(s) from an openpgp.Entity
// for printing to the UI and logs.
func entityString(entity *openpgp.Entity) string {
	if entity == nil {
		return ""
	}

	keyID := "n/a"
	if entity.PrimaryKey != nil {
		keyID = entity.PrimaryKey.KeyIdString()
	}

	var names []string
	for _, identity := range entity.Identities {
		names = append(names, identity.Name)
	}

	return fmt.Sprintf("%s %s", keyID, strings.Join(names, ", "))
}
