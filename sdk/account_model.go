// Copyright 2018 ProximaX Limited. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package sdk

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"github.com/proximax-storage/go-xpx-utils/str"
	"github.com/proximax-storage/nem2-crypto-go"
	"math/big"
	"strings"
)

const EmptyPublicKey = "0000000000000000000000000000000000000000000000000000000000000000"

type Account struct {
	*PublicAccount
	*crypto.KeyPair
}

// signs given transaction
// returns a signed transaction
func (a *Account) Sign(tx Transaction) (*SignedTransaction, error) {
	return signTransactionWith(tx, a)
}

func (a *Account) SignWithCosignatures(tx *AggregateTransaction, cosignatories []*Account) (*SignedTransaction, error) {
	return signTransactionWithCosignatures(tx, a, cosignatories)
}

// signs aggregate signature transaction
// returns signed cosignature transaction
func (a *Account) SignCosignatureTransaction(tx *CosignatureTransaction) (*CosignatureSignedTransaction, error) {
	return signCosignatureTransaction(a, tx)
}

type PublicAccount struct {
	Address   *Address
	PublicKey string
}

func (ref *PublicAccount) String() string {
	return fmt.Sprintf(`Address: %+v, PublicKey: "%s"`, ref.Address, ref.PublicKey)
}

type AccountType uint8

// AccountType enums
const (
	UnlinkedAccount AccountType = iota
	MainAccount
	RemoteAccount
	RemoteUnlinkedAccount
)

type AccountProperties struct {
	Address            *Address
	AllowedAddresses   []*Address
	AllowedMosaicId    []*MosaicId
	AllowedEntityTypes []TransactionType
	BlockedAddresses   []*Address
	BlockedMosaicId    []*MosaicId
	BlockedEntityTypes []TransactionType
}

func (a *AccountProperties) String() string {
	return str.StructToString(
		"AccountProperties",
		str.NewField("Address", str.StringPattern, a.Address),
		str.NewField("AllowedAddresses", str.StringPattern, a.AllowedAddresses),
		str.NewField("AllowedMosaicId", str.StringPattern, a.AllowedMosaicId),
		str.NewField("AllowedEntityTypes", str.StringPattern, a.AllowedEntityTypes),
		str.NewField("BlockedAddresses", str.StringPattern, a.BlockedAddresses),
		str.NewField("BlockedMosaicId", str.StringPattern, a.BlockedMosaicId),
		str.NewField("BlockedEntityTypes", str.StringPattern, a.BlockedEntityTypes),
	)
}

type AccountInfo struct {
	Address         *Address
	AddressHeight   *big.Int
	PublicKey       string
	PublicKeyHeight *big.Int
	AccountType     AccountType
	LinkedAccount   *PublicAccount
	Mosaics         []*Mosaic
	Reputation      float64
}

func (a *AccountInfo) String() string {
	return str.StructToString(
		"AccountInfo",
		str.NewField("Address", str.StringPattern, a.Address),
		str.NewField("AddressHeight", str.StringPattern, a.AddressHeight),
		str.NewField("PublicKey", str.StringPattern, a.PublicKey),
		str.NewField("PublicKeyHeight", str.StringPattern, a.PublicKeyHeight),
		str.NewField("AccountType", str.IntPattern, a.AccountType),
		str.NewField("LinkedAccount", str.StringPattern, a.LinkedAccount),
		str.NewField("Mosaics", str.StringPattern, a.Mosaics),
		str.NewField("Reputation", str.FloatPattern, a.Reputation),
	)
}

type Address struct {
	Type    NetworkType
	Address string
}

func (ad *Address) String() string {
	return str.StructToString(
		"",
		str.NewField("Type", str.IntPattern, ad.Type),
		str.NewField("Address", str.StringPattern, ad.Address),
	)
}

func (ad *Address) Pretty() string {
	res := ""
	for i := 0; i < 6; i++ {
		res += ad.Address[i*6:i*6+6] + "-"
	}
	res += ad.Address[len(ad.Address)-4:]
	return res
}

type MultisigAccountInfo struct {
	Account          PublicAccount
	MinApproval      int32
	MinRemoval       int32
	Cosignatories    []*PublicAccount
	MultisigAccounts []*PublicAccount
}

func (ref *MultisigAccountInfo) String() string {
	return str.StructToString(
		"MultisigAccountInfo",
		str.NewField("Account", str.StringPattern, ref.Account),
		str.NewField("MinApproval", str.IntPattern, ref.MinApproval),
		str.NewField("MinRemoval", str.IntPattern, ref.MinRemoval),
		str.NewField("Cosignatories", str.StringPattern, ref.Cosignatories),
		str.NewField("MultisigAccounts", str.StringPattern, ref.MultisigAccounts),
	)
}

type MultisigAccountGraphInfo struct {
	MultisigAccounts map[int32][]*MultisigAccountInfo
}

// returns new account generated for passed network type
func NewAccount(networkType NetworkType) (*Account, error) {
	kp, err := crypto.NewKeyPairByEngine(crypto.CryptoEngines.DefaultEngine)
	if err != nil {
		return nil, err
	}

	pa, err := NewAccountFromPublicKey(kp.PublicKey.String(), networkType)
	if err != nil {
		return nil, err
	}

	return &Account{pa, kp}, nil
}

// returns new account from private key for passed network type
func NewAccountFromPrivateKey(pKey string, networkType NetworkType) (*Account, error) {
	k, err := crypto.NewPrivateKeyfromHexString(pKey)
	if err != nil {
		return nil, err
	}

	kp, err := crypto.NewKeyPair(k, nil, nil)
	if err != nil {
		return nil, err
	}

	pa, err := NewAccountFromPublicKey(kp.PublicKey.String(), networkType)
	if err != nil {
		return nil, err
	}

	return &Account{pa, kp}, nil
}

// returns a public account from public key for passed network type
func NewAccountFromPublicKey(pKey string, networkType NetworkType) (*PublicAccount, error) {
	ad, err := NewAddressFromPublicKey(pKey, networkType)
	if err != nil {
		return nil, err
	}
	return &PublicAccount{ad, pKey}, nil
}

// returns address entity from passed address string for passed network type
func NewAddress(address string, networkType NetworkType) *Address {
	address = strings.Replace(address, "-", "", -1)
	address = strings.ToUpper(address)
	return &Address{networkType, address}
}

func NewAddressFromRaw(address string) (*Address, error) {
	pH, err := base32.StdEncoding.DecodeString(address)
	if err != nil {
		return nil, err
	}

	if nType, ok := addressNet[pH[0]]; ok {
		return NewAddress(address, nType), nil
	}

	return nil, ErrInvalidAddress
}

// returns an address from public key for passed network type
func NewAddressFromPublicKey(pKey string, networkType NetworkType) (*Address, error) {
	ad, err := generateEncodedAddress(pKey, networkType)
	if err != nil {
		return nil, err
	}

	return NewAddress(ad, networkType), nil
}

func NewAddressFromEncoded(encoded string) (*Address, error) {
	pH, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	parsed := base32.StdEncoding.EncodeToString(pH)
	ad, err := NewAddressFromRaw(parsed)
	if err != nil {
		return nil, err
	}

	return ad, nil
}

func EncodedStringToAddresses(addresses ...string) ([]*Address, error) {
	result := make([]*Address, len(addresses))
	for i, a := range addresses {
		var err error = nil
		result[i], err = NewAddressFromEncoded(a)
		if err != nil {
			return nil, err
		}

	}

	return result, nil
}

const NUM_CHECKSUM_BYTES = 4

func GenerateChecksum(b []byte) ([]byte, error) {
	// step 1: sha3 hash of (input
	sha3StepThreeHash, err := crypto.HashesSha3_256(b)
	if err != nil {
		return nil, err
	}

	// step 2: get the first NUM_CHECKSUM_BYTES bytes of (1)
	return sha3StepThreeHash[:NUM_CHECKSUM_BYTES], nil
}
