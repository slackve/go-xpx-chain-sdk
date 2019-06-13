// Copyright 2018 ProximaX Limited. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package sdk

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/json-iterator/go"
	"github.com/proximax-storage/go-xpx-utils/str"
	"strings"
	"unsafe"
)

const NamespaceBit uint64 = 1 << 63

type NamespaceId struct {
	baseInt64
}

func NewNamespaceId(id uint64) (*NamespaceId, error) {
	if id != 0 && !hasBits(id, NamespaceBit) {
		return nil, ErrWrongBitNamespaceId
	}

	return NewNamespaceIdNoCheck(id), nil
}

func NewNamespaceIdNoCheck(id uint64) *NamespaceId {
	namespaceId := NamespaceId{baseInt64(id)}
	return &namespaceId
}

func (m *NamespaceId) Type() BlockchainIdType {
	return NamespaceBlockchainIdType
}

func (m *NamespaceId) Id() uint64 {
	return uint64(m.baseInt64)
}

func (m *NamespaceId) String() string {
	return m.toHexString()
}

func (m *NamespaceId) toHexString() string {
	return uint64ToHex(m.Id())
}

func (m *NamespaceId) Equals(id *NamespaceId) bool {
	return *m == *id
}

// returns namespace id from passed namespace name
// should be used for creating root, child and grandchild namespace ids
// to create root namespace pass namespace name in format like 'rootname'
// to create child namespace pass namespace name in format like 'rootname.childname'
// to create grand child namespace pass namespace name in format like 'rootname.childname.grandchildname'
func NewNamespaceIdFromName(namespaceName string) (*NamespaceId, error) {
	if list, err := GenerateNamespacePath(namespaceName); err != nil {
		return nil, err
	} else {
		l := len(list)

		if l == 0 {
			return nil, ErrInvalidNamespaceName
		}

		return list[l-1], nil
	}
}

type NamespaceIds struct {
	List []*NamespaceId
}

func (ref *NamespaceIds) MarshalJSON() (buf []byte, err error) {
	buf = []byte(`{"namespaceIds": [`)

	for i, nsId := range ref.List {
		if i > 0 {
			buf = append(buf, ',')
		}

		buf = append(buf, []byte(`"`+nsId.toHexString()+`"`)...)
	}

	buf = append(buf, ']', '}')

	return
}

func (ref *NamespaceIds) IsEmpty(ptr unsafe.Pointer) bool {
	return len((*NamespaceIds)(ptr).List) == 0
}

func (ref *NamespaceIds) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	if (*NamespaceIds)(ptr) == nil {
		ptr = (unsafe.Pointer)(&NamespaceIds{})
	}

	if iter.ReadNil() {
		*((*unsafe.Pointer)(ptr)) = nil
	} else {
		if iter.WhatIsNext() == jsoniter.ArrayValue {
			iter.Skip()
			newIter := iter.Pool().BorrowIterator([]byte("{}"))
			defer iter.Pool().ReturnIterator(newIter)
			v := newIter.Read()
			list := make([]*NamespaceId, 0)
			for _, val := range v.([]*NamespaceId) {
				list = append(list, val)
			}
			(*NamespaceIds)(ptr).List = list
		}
	}
}

func (ref *NamespaceIds) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	buf, err := (*NamespaceIds)(ptr).MarshalJSON()
	if err == nil {
		_, err = stream.Write(buf)
		//	todo: log error in future
	}

}

// NamespaceAlias contains aliased mosaicId or address and type of alias
type NamespaceAlias struct {
	mosaicId *MosaicId
	address  *Address
	Type     AliasType
}

func NewNamespaceAlias(dto *namespaceAliasDTO) (*NamespaceAlias, error) {
	alias := NamespaceAlias{}

	alias.Type = dto.Type

	switch alias.Type {
	case AddressAliasType:
		a, err := NewAddressFromBase32(dto.Address)
		if err != nil {
			return nil, err
		}

		alias.address = a
	case MosaicAliasType:
		mosaicId, err := dto.MosaicId.toStruct()
		if err != nil {
			return nil, err
		}
		alias.mosaicId = mosaicId
	}

	return &alias, nil
}

func (ref *NamespaceAlias) Address() *Address {
	return ref.address
}

func (ref *NamespaceAlias) MosaicId() *MosaicId {
	return ref.mosaicId
}

func (ref *NamespaceAlias) String() string {
	switch ref.Type {
	case AddressAliasType:
		return str.StructToString(
			"NamespaceAlias",
			str.NewField("Address", str.StringPattern, ref.Address()),
			str.NewField("Type", str.IntPattern, ref.Type),
		)
	case MosaicAliasType:
		return str.StructToString(
			"NamespaceAlias",
			str.NewField("MosaicId", str.StringPattern, ref.MosaicId()),
			str.NewField("Type", str.IntPattern, ref.Type),
		)
	}
	return str.StructToString(
		"NamespaceAlias",
		str.NewField("Type", str.IntPattern, ref.Type),
	)
}

type NamespaceInfo struct {
	NamespaceId *NamespaceId
	Active      bool
	TypeSpace   NamespaceType
	Depth       int
	Levels      []*NamespaceId
	Alias       *NamespaceAlias
	Parent      *NamespaceInfo
	Owner       *PublicAccount
	StartHeight Height
	EndHeight   Height
}

func (ref *NamespaceInfo) String() string {
	return str.StructToString(
		"NamespaceInfo",
		str.NewField("NamespaceId", str.StringPattern, ref.NamespaceId),
		str.NewField("Active", str.BooleanPattern, ref.Active),
		str.NewField("TypeSpace", str.IntPattern, ref.TypeSpace),
		str.NewField("Depth", str.IntPattern, ref.Depth),
		str.NewField("Levels", str.StringPattern, ref.Levels),
		str.NewField("Alias", str.StringPattern, ref.Alias),
		str.NewField("Parent", str.StringPattern, ref.Parent),
		str.NewField("Owner", str.StringPattern, ref.Owner),
		str.NewField("StartHeight", str.StringPattern, ref.StartHeight),
		str.NewField("EndHeight", str.StringPattern, ref.EndHeight),
	)
}

type NamespaceName struct {
	NamespaceId *NamespaceId
	Name        string
	ParentId    *NamespaceId /* Optional NamespaceId my be nil */
}

func (n *NamespaceName) String() string {
	return str.StructToString(
		"NamespaceName",
		str.NewField("NamespaceId", str.StringPattern, n.NamespaceId),
		str.NewField("Name", str.StringPattern, n.Name),
		str.NewField("ParentId", str.StringPattern, n.ParentId),
	)
}

// returns an array of big ints representation if namespace ids from passed namespace path
// to create root namespace pass namespace name in format like 'rootname'
// to create child namespace pass namespace name in format like 'rootname.childname'
// to create grand child namespace pass namespace name in format like 'rootname.childname.grandchildname'
func GenerateNamespacePath(name string) ([]*NamespaceId, error) {
	parts := strings.Split(name, ".")

	if len(parts) == 0 {
		return nil, ErrInvalidNamespaceName
	}

	if len(parts) > 3 {
		return nil, ErrNamespaceTooManyPart
	}

	var (
		namespaceId = NewNamespaceIdNoCheck(0)
		path        = make([]*NamespaceId, 0)
		err         error
	)

	for _, part := range parts {
		if !regValidNamespace.MatchString(part) {
			return nil, ErrInvalidNamespaceName
		}

		if namespaceId, err = generateNamespaceId(part, namespaceId); err != nil {
			return nil, err
		} else {
			path = append(path, namespaceId)
		}
	}

	return path, nil
}

func NewAddressFromNamespace(namespaceId *NamespaceId) (*Address, error) {
	// 0x91 | namespaceId on 8 bytes | 16 bytes 0-pad = 25 bytes
	a := fmt.Sprintf("%X", int(AliasAddress))

	namespaceB := make([]byte, 8)
	binary.LittleEndian.PutUint64(namespaceB, namespaceId.Id())

	a += hex.EncodeToString(namespaceB)
	a += strings.Repeat("00", 16)

	return NewAddressFromBase32(a)
}
